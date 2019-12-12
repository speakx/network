package server

import (
	"time"
)

type TCPServer struct {
	*Server
}

func NewTCPServer() *TCPServer {
	return &TCPServer{}
}

func (tcps *TCPServer) Run(addr string, maxEvent, bufSize, bufPoolSize int, bufPoolRecyleDur time.Duration) error {
	if 0 == maxEvent {
		maxEvent = 0xFFFFF
	}
	if 0 == bufSize {
		bufSize = 1024 * 1024
	}
	if 0 == bufPoolSize {
		bufPoolSize = 1024 * 2
	}
	if 0 == bufPoolRecyleDur {
		bufPoolRecyleDur = 1 * time.Second
	}

	tcps.Server = newServer(maxEvent, bufSize, bufPoolSize, bufPoolRecyleDur)
	return tcps.run("tcp", addr)
}
