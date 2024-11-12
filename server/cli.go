package main

import (
	"log"

	"github.com/spf13/cobra"
)

func init() {
	runCmd.Flags().IntVarP(&port, "port", "p", 1080, "Port to listen on")
	runCmd.Flags().BoolVar(&useTLS, "tls", false, "Use TLS for connections")
	runCmd.Flags().BoolVar(&useTor, "tor", false, "Use Tor for connections")
	runCmd.MarkFlagsMutuallyExclusive("tls", "tor")

	startCmd.Flags().IntVarP(&port, "port", "p", 1080, "Port to listen on")
	startCmd.Flags().BoolVar(&useTLS, "tls", false, "Use TLS for connections")
	startCmd.Flags().BoolVar(&useTor, "tor", false, "Use Tor for connections")
	startCmd.MarkFlagsMutuallyExclusive("tls", "tor")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(closeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "revsocks",
	Short: "A reverse SOCKS5 proxy server",
	Long:  `A server component of the reverse SOCKS5 proxy system that accepts connections from agents.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the proxy server in foreground",
	Long:  `Start the reverse SOCKS5 proxy server and listen for incoming agent connections in foreground.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server in background",
	Long:  `Start the reverse SOCKS5 proxy server and listen for incoming agent connections in background.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return startServer(args)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the proxy server running in background",
	Long:  `Stop the reverse SOCKS5 proxy server that was started in background mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopServer()
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all active connections",
	Long:  `List all active connections and their IDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listConnections()
	},
}

var closeCmd = &cobra.Command{
	Use:   "close",
	Short: "Close a connection by ID",
	Long:  `Close a connection by its ID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return closeConnection(args[0])
	},
}
