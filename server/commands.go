package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func runServer() error {
	daemonService := NewDaemonService()
	go func() {
		if err := daemonService.Start(); err != nil {
			log.Fatalf("failed to start daemon service: %v", err)
		}
	}()

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
	if useTLS {
		procArgs = append(procArgs, "--tls")
	}
	if useTor {
		procArgs = append(procArgs, "--tor")
	}
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

func stopServer() error {
	client := NewDaemonClient()
	return client.Shutdown()
}
