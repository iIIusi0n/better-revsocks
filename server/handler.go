package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"time"

	"github.com/hashicorp/yamux"
)

var (
	MagicBytes = []byte{0x1b, 0xc3, 0xbd, 0x0f}
	port       int
)

type ConnectionHandler struct {
	conn                net.Conn
	socksClientListener net.Listener
	session             *yamux.Session
	healthChan          chan bool
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
	return &ConnectionHandler{
		conn:       conn,
		healthChan: make(chan bool),
	}
}

func (h *ConnectionHandler) setupListener() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Printf("Failed to create TCP listener: %v", err)
		return err
	}
	h.socksClientListener = listener
	log.Printf("SOCKS listener started on %s", listener.Addr().String())
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

func runServer() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to create main listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Server started, listening on %s for agents", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}

		log.Printf("New connection from %v", conn.RemoteAddr())
		if err := validateMagicBytes(conn); err != nil {
			log.Printf("Magic bytes validation failed for %v: %s", conn.RemoteAddr(), err)
			conn.Close()
			continue
		}

		go handleConnection(conn)
	}
}

func startServer(args []string) error {
	procArgs := []string{os.Args[0], "run"}
	procArgs = append(procArgs, "-p", fmt.Sprintf("%d", port))
	procArgs = append(procArgs, args...)

	proc, err := os.StartProcess(os.Args[0], procArgs, &os.ProcAttr{
		Files: []*os.File{nil, nil, nil},
	})
	if err != nil {
		return fmt.Errorf("failed to start daemon process: %v", err)
	}
	log.Printf("Started daemon process with PID %d", proc.Pid)
	return nil
}
