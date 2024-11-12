package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math/big"
	"net"
	"reflect"
	"time"

	"github.com/hashicorp/yamux"
)

var (
	MagicBytes = []byte{0x1b, 0xc3, 0xbd, 0x0f}
	port       int
	useTLS     bool
	useTor     bool
)

var connections = make(map[string]*ConnectionHandler)

type ConnectionHandler struct {
	conn                net.Conn
	socksClientListener net.Listener
	session             *yamux.Session
	healthChan          chan bool
}

type ConnectionHandlerInfo struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	ListenAddr string `json:"listen_addr"`
}

func copyData(dst, src net.Conn) {
	written, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("Error copying data: %v", err)
	}
	log.Printf("Copied %d bytes", written)
	dst.Close()
}

func copyClientConnToServer(clientConn, serverConn net.Conn) {
	log.Printf("Starting bidirectional copy between %v and %v", clientConn.RemoteAddr(), serverConn.RemoteAddr())
	go copyData(clientConn, serverConn)
	go copyData(serverConn, clientConn)
}

func NewConnectionHandler(conn net.Conn) *ConnectionHandler {
	log.Printf("New connection handler created for connection from %v", conn.RemoteAddr())
	id := generateConnectionID(conn)
	connections[id] = &ConnectionHandler{
		conn:       conn,
		healthChan: make(chan bool),
	}
	return connections[id]
}

func (h *ConnectionHandler) setupListener() error {
	var listener net.Listener
	var err error

	listener, err = net.Listen("tcp", ":0")
	if err != nil {
		log.Printf("Failed to create listener: %v", err)
		return err
	}

	h.socksClientListener = listener
	log.Printf("Listener started on %s (TLS: %v)", listener.Addr().String(), useTLS)
	return nil
}

func (h *ConnectionHandler) setupYamuxSession() error {
	log.Printf("Setting up yamux session for connection from %v", h.conn.RemoteAddr())
	session, err := yamux.Client(h.conn, nil)
	if err != nil {
		log.Printf("Failed to create yamux client: %v", err)
		return err
	}
	h.session = session
	log.Printf("Yamux session established successfully")
	return nil
}

func (h *ConnectionHandler) monitorHealth() {
	log.Printf("Starting health monitor for connection from %v", h.conn.RemoteAddr())
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if _, err := h.session.Ping(); err != nil {
			log.Printf("Health check failed: %v", err)
			h.healthChan <- false
			return
		}
	}
}

func (h *ConnectionHandler) handleClientConnection() error {
	acceptChan := make(chan net.Conn)
	errChan := make(chan error)

	go func() {
		log.Printf("Waiting for client connection on %v", h.socksClientListener.Addr())
		clientConn, err := h.socksClientListener.Accept()
		if err != nil {
			log.Printf("Failed to accept client connection: %v", err)
			errChan <- err
		} else {
			log.Printf("Accepted client connection from %v", clientConn.RemoteAddr())
			acceptChan <- clientConn
		}
	}()

	select {
	case <-h.healthChan:
		log.Printf("Connection health check failed")
		return io.EOF
	case clientConn := <-acceptChan:
		return h.establishServerConnection(clientConn)
	case err := <-errChan:
		return err
	}
}

func (h *ConnectionHandler) establishServerConnection(clientConn net.Conn) error {
	log.Printf("Opening new yamux stream for client %v", clientConn.RemoteAddr())
	serverConn, err := h.session.Open()
	if err != nil {
		log.Printf("Failed to open yamux stream: %v", err)
		clientConn.Close()
		return err
	}
	log.Printf("Successfully opened yamux stream")
	go copyClientConnToServer(clientConn, serverConn)
	return nil
}

func (h *ConnectionHandler) Close() {
	delete(connections, generateConnectionID(h.conn))
	h.session.Close()
	h.conn.Close()
	h.socksClientListener.Close()
}

func handleConnection(conn net.Conn) {
	log.Printf("Handling new connection from %v", conn.RemoteAddr())
	handler := NewConnectionHandler(conn)

	if err := handler.setupListener(); err != nil {
		log.Printf("Error creating listener: %s", err)
		return
	}
	defer handler.socksClientListener.Close()

	if err := handler.setupYamuxSession(); err != nil {
		log.Printf("Error creating yamux client: %s", err)
		return
	}
	defer handler.session.Close()

	go handler.monitorHealth()

	for {
		if err := handler.handleClientConnection(); err != nil {
			if err == io.EOF {
				log.Printf("Connection closed from %v", conn.RemoteAddr())
			} else {
				log.Printf("Error handling connection from %v: %s", conn.RemoteAddr(), err)
			}
			return
		}
	}
}

func validateMagicBytes(conn net.Conn) error {
	log.Printf("Validating magic bytes from %v", conn.RemoteAddr())
	magic := make([]byte, len(MagicBytes))
	if _, err := conn.Read(magic); err != nil {
		log.Printf("Failed to read magic bytes: %v", err)
		return err
	}
	if !reflect.DeepEqual(magic, MagicBytes) {
		log.Printf("Invalid magic bytes received: %v", magic)
		return io.EOF
	}
	log.Printf("Magic bytes validated successfully")
	return nil
}

func generateConnectionID(conn net.Conn) string {
	ip := conn.RemoteAddr()
	h := crc32.NewIEEE()
	h.Write([]byte(ip.String()))
	return fmt.Sprintf("%x", h.Sum32())
}

func generateTLSConfig() (*tls.Config, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Self-Signed Cert"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	tlsCert, err := tls.X509KeyPair(certPEM, privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}, nil
}
