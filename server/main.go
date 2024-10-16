package main

import (
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

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
		panic(err)
	}
	defer socksClientListener.Close()

	log.Printf("Listening on %s", socksClientListener.Addr().String())

	session, err := yamux.Client(conn, nil)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	for {
		clientConn, err := socksClientListener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			break
		}

		serverConn, err := session.Open()
		if err != nil {
			panic(err)
		}

		copyClientConnToServer(clientConn, serverConn)
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

		go handleConnection(conn)
	}
}
