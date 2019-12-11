package server

import (
	"errors"
	"net"
	"network/bufpool"
	"network/reactor"
	"time"

	"environment/logger"

	"golang.org/x/sys/unix"
)

type Server struct {
	events        []unix.EpollEvent
	sidToSessions map[int32]reactor.Session
	sidCounter    int32
	listener      net.Listener
	fdEpoll       int
	onAcceptfn    func(net.Conn) (reactor.Session, error)
}

func newServer(maxEvent, bufSize, bufPoolSize int, bufPoolRecyleDur time.Duration,
	onAcceptfn func(net.Conn) (reactor.Session, error)) *Server {
	bufpool.InitBufPool(bufSize, bufPoolSize, bufPoolRecyleDur)
	return &Server{
		events:        make([]unix.EpollEvent, maxEvent),
		sidToSessions: make(map[int32]reactor.Session),
		sidCounter:    0,
		onAcceptfn:    onAcceptfn,
	}
}

func (s *Server) run(network, addr string) error {
	// create listener
	ln, err := net.Listen(network, addr)
	if nil != err {
		logger.Error("Server Listen err:", err)
		return err
	}
	var fdListen uintptr
	syscallListen, err := ln.(*net.TCPListener).SyscallConn()
	if nil != err {
		logger.Error("Server Listen.SyscallConn err:", err)
		return err
	}
	err = syscallListen.Control(func(fd uintptr) { fdListen = fd })
	if nil != err {
		logger.Error("Server Listen.SyscallConn.Control err:", err)
		return err
	}
	s.listener = ln

	// create epoll
	fdEpoll, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if nil != err {
		logger.Error("Server EpollCreate1 err:", err)
		return err
	}
	defer unix.Close(fdEpoll)
	s.fdEpoll = fdEpoll

	// register listen epoll
	s.AddHandler(reactor.MASK_READ, int32(fdListen), int32(0))
	s.handleEpollEvent(fdEpoll)

	return nil
}

func (s *Server) genSessionid() int32 {
	s.sidCounter++
	return s.sidCounter
}

func (s *Server) addSession(session reactor.Session) {
	s.sidToSessions[session.Sid()] = session
	logger.Info("Server session add addr:", session.Addr())
}

func (s *Server) removeSession(session reactor.Session) {
	s.DelHandler(int32(session.FD()))

	delete(s.sidToSessions, session.Sid())
	session.Close()
	logger.Info("Server session remove addr:", session.Addr())
}

func (s *Server) sidToSession(sid int32) reactor.Session {
	session, _ := s.sidToSessions[sid]
	return session
}

func (s *Server) onListen() error {
	// do accept
	conn, err := s.listener.Accept()
	if nil != err {
		logger.Error("Server Accept err:", err)
		return err
	}
	logger.Info("Server.Accept addr:", conn.RemoteAddr())

	// make session
	session, err := s.onAcceptfn(conn)
	if nil != err {
		logger.Error("Server onAccept err:", err)
		return err
	}
	if nil == session {
		logger.Error("Server onAccept session is nil")
		return errors.New("create session failed")
	}
	s.addSession(session)
	logger.Info("Server.Accept session sid:", session.Sid(), " fd:", session.FD())

	// register session.read
	s.AddHandler(reactor.MASK_READ, session.FD(), session.Sid())

	return nil
}
