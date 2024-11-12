package main

import (
	"crypto/tls"
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

	var listener net.Listener
	var err error
	if useTLS {
		tlsConfig, err := generateTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to generate TLS config: %v", err)
		}
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	}
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

func listConnections() error {
	client := NewDaemonClient()
	infos, err := client.ListConnections()
	if err != nil {
		return err
	}

	fmt.Printf("\nActive connections:\n")
	fmt.Printf("%-10s %-15s %-21s\n", "ID", "IP", "Listen Address")
	fmt.Printf("%-10s %-15s %-21s\n", "----------", "---------------", "---------------------")
	for _, info := range infos {
		fmt.Printf("%-10s %-15s %-21s\n", info.ID, info.IP, info.ListenAddr)
	}
	fmt.Println()

	return nil
}

func closeConnection(id string) error {
	client := NewDaemonClient()
	err := client.CloseConnection(id)
	if err != nil {
		return err
	}

	fmt.Printf("Closed connection %s\n", id)
	return nil
}
