package main

import (
	"log"

	"github.com/spf13/cobra"
)

func init() {
	startCmd.Flags().IntVarP(&port, "port", "p", 1080, "Port to listen on")
	rootCmd.AddCommand(startCmd)
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

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server",
	Long:  `Start the reverse SOCKS5 proxy server and listen for incoming agent connections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return startServer()
	},
}
