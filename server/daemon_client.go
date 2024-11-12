package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

func (c *Client) call(method, path string, body io.Reader) ([]byte, error) {
	path = "http://127.0.0.1:9191" + path
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send %s request: %v", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) Shutdown() error {
	_, err := c.call("POST", "/shutdown", nil)
	return err
}

func (c *Client) ListConnections() ([]ConnectionHandlerInfo, error) {
	data, err := c.call("GET", "/connections", nil)
	if err != nil {
		return nil, err
	}

	var infos []ConnectionHandlerInfo
	return infos, json.Unmarshal(data, &infos)
}

func (c *Client) CloseConnection(id string) error {
	_, err := c.call("POST", "/close", strings.NewReader(fmt.Sprintf(`{"id": "%s"}`, id)))
	return err
}
