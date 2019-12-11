package session

import (
	"net"
	"network/reactor"
)

type TCPSession struct {
	*baseSession
}

func NewTCPSession(conn *net.TCPConn, sid int32, reactor reactor.Reactor) (*TCPSession, error) {
	syscallConn, err := conn.SyscallConn()
	if nil != err {
		return nil, err
	}

	var syscallFD uintptr
	err = syscallConn.Control(func(fd uintptr) { syscallFD = fd })
	if nil != err {
		return nil, err
	}

	s := &TCPSession{}
	s.baseSession = newBaseSession(sid, conn, int32(syscallFD), reactor)
	return s, nil
}
