package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Client struct {
	client *http.Client
}

func NewDaemonClient() *Client {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("tcp", "127.0.0.1:9191")
				},
			},
		},
	}
}

func (c *Client) call(method, path string, body io.Reader) error {
	path = "http://127.0.0.1:9191" + path
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send %s request: %v", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Shutdown() error {
	return c.call("POST", "/shutdown", nil)
}
