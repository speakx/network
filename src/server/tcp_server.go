package server

import (
	"runtime"
	"time"
)

type TCPServer struct {
	*Server
}

func NewTCPServer() *TCPServer {
	return &TCPServer{}
}

func (tcps *TCPServer) Run(addr string, maxEvent, goroutineLimit, bufSize, bufPoolSize int, bufPoolRecyleDur time.Duration) error {
	if 0 == maxEvent {
		maxEvent = 0xFFF
	}
	if 0 == goroutineLimit {
		goroutineLimit = runtime.NumCPU()
	}
	if 0 == bufSize {
		bufSize = 1024 * 256
	}
	if 0 == bufPoolSize {
		bufPoolSize = 1024 * 2
	}
	if 0 == bufPoolRecyleDur {
		bufPoolRecyleDur = 2 * time.Second
	}

	tcps.Server = newServer(maxEvent, goroutineLimit, bufSize, bufPoolSize, bufPoolRecyleDur)
	return tcps.run("tcp", addr)
}
