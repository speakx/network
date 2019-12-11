package server

import (
	"net"
	"network/reactor"
	"network/session"
	"time"
)

type TCPServer struct {
	*Server
}

func NewTCPServer(maxEvent, bufSize, bufPoolSize int, bufPoolRecyleDur time.Duration) *TCPServer {
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

	s := &TCPServer{}
	s.Server = newServer(
		maxEvent, bufSize, bufPoolSize, bufPoolRecyleDur,
		s.onAccept)
	return s
}

func (tcps *TCPServer) Run(addr string) error {
	return tcps.run("tcp", addr)
}

func (tcps *TCPServer) onAccept(conn net.Conn) (reactor.Session, error) {
	tcpConn := conn.(*net.TCPConn)
	return session.NewTCPSession(tcpConn, tcps.genSessionid(), tcps)
}
