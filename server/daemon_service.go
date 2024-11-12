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
}

func (d *DaemonService) shutdownHandler(c *gin.Context) {
	c.Status(http.StatusOK)
	go func() {
		time.Sleep(500 * time.Millisecond)
		d.listener.Close()
		os.Exit(0)
	}()
}
