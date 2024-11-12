package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type DaemonService struct {
	router   *gin.Engine
	listener net.Listener
}

func NewDaemonService() *DaemonService {
	return &DaemonService{
		router: gin.Default(),
	}
}

func (d *DaemonService) Start() error {
	var err error
	d.listener, err = net.Listen("tcp", "127.0.0.1:9191")
	if err != nil {
		return fmt.Errorf("failed to listen on port 9191: %v", err)
	}

	d.setupRoutes()

	return d.router.RunListener(d.listener)
}

func (d *DaemonService) setupRoutes() {
	d.router.POST("/shutdown", d.shutdownHandler)
	d.router.GET("/connections", d.connectionsHandler)
	d.router.POST("/close", d.closeConnectionHandler)
}

func (d *DaemonService) shutdownHandler(c *gin.Context) {
	c.Status(http.StatusOK)
	go func() {
		time.Sleep(500 * time.Millisecond)
		d.listener.Close()
		os.Exit(0)
	}()
}

func (d *DaemonService) connectionsHandler(c *gin.Context) {
	infos := make([]ConnectionHandlerInfo, 0, len(connections))
	for id, handler := range connections {
		infos = append(infos, ConnectionHandlerInfo{
			ID:         id,
			IP:         handler.conn.RemoteAddr().(*net.TCPAddr).IP.String(),
			ListenAddr: handler.socksClientListener.Addr().String(),
		})
	}
	c.JSON(http.StatusOK, infos)
}

func (d *DaemonService) closeConnectionHandler(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handler, ok := connections[req.ID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}
	handler.Close()
	c.Status(http.StatusOK)
}
