package main

import (
	"io"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/hashicorp/yamux"
)

var MagicBytes = []byte{0x1b, 0xc3, 0xbd, 0x0f}

func copyClientConnToServer(clientConn, serverConn net.Conn) {
	go func() {
		io.Copy(clientConn, serverConn)
		clientConn.Close()
	}()

	go func() {
		io.Copy(serverConn, clientConn)
		serverConn.Close()
	}()
}

func handleConnection(conn net.Conn) {
	socksClientListener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Printf("Error creating listener: %s", err)
		return
	}
	defer socksClientListener.Close()

	log.Printf("Listening on %s", socksClientListener.Addr().String())

	session, err := yamux.Client(conn, nil)
	if err != nil {
		log.Printf("Error creating yamux client: %s", err)
		return
	}
	defer session.Close()

	healthChan := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := session.Ping()
				if err != nil {
					healthChan <- false
					return
				}
			}
		}
	}()

	for {
		acceptChan := make(chan net.Conn)
		errChan := make(chan error)

		go func() {
			clientConn, err := socksClientListener.Accept()
			if err != nil {
				errChan <- err
			} else {
				acceptChan <- clientConn
			}
		}()

		select {
		case <-healthChan:
			log.Printf("Connection closed")
			return
		case clientConn := <-acceptChan:
			serverConn, err := session.Open()
			if err != nil {
				log.Printf("Error opening session: %s", err)
				clientConn.Close()
				continue
			}
			go copyClientConnToServer(clientConn, serverConn)
		case err := <-errChan:
			log.Printf("Error accepting connection: %s", err)
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":1080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	log.Printf("Listening on %s for agents", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}

		magic := make([]byte, len(MagicBytes))
		_, err = conn.Read(magic)
		if err != nil {
			log.Printf("Error reading magic bytes: %s", err)
			conn.Close()
			continue
		}

		if reflect.DeepEqual(magic, MagicBytes) == false {
			log.Printf("Invalid magic bytes")
			conn.Close()
			continue
		}

		go handleConnection(conn)
	}
}
