package server

import (
	"fmt"
	"net"
	"network/bufpool"
	"network/idmaker"
	"network/reactor"
	"network/register"
	"sync"
	"time"

	"environment/logger"
)

type Server struct {
	sid          int32
	lnFD         int32
	rw           uint32
	listener     net.Listener
	epollReactor *reactor.EpollReactor
	sync.Mutex
}

func newServer(maxEvent, goroutineLimit, bufSize, bufPoolSize int, bufPoolRecyleDur time.Duration) *Server {
	bufpool.InitBufPool(bufSize, bufPoolSize, bufPoolRecyleDur)
	return &Server{
		sid:          idmaker.MakeSidSafeLock(),
		epollReactor: reactor.NewEpollReactor(maxEvent, goroutineLimit),
	}
}

func (s *Server) Info() string {
	return fmt.Sprintf("Server addr:%v", s.listener.Addr())
}

func (s *Server) GetId() int32 {
	return s.sid
}

func (s *Server) GetFd() int32 {
	return s.lnFD
}

func (s *Server) SetRWState(rw uint32) {
}

func (s *Server) ModRWState(rw uint32, add bool) uint32 {
	s.Lock()
	if true == add {
		s.rw |= rw
	} else {
		s.rw &^= rw
	}
	rw = s.rw
	s.Unlock()
	return rw
}

func (s *Server) GetRWState() uint32 {
	return reactor.MASK_READ
}

func (s *Server) SetReadTag(tag int) {
}

func (s *Server) GetReadTag() int {
	return 0
}

func (s *Server) DoRead() (int, error) {
	// accept
	conn, err := s.listener.Accept()
	if nil != err {
		logger.Error("Server Accept err:", err)
		return 0, err
	}
	logger.Info("Server.Accept addr:", conn.RemoteAddr())

	// create session
	session := register.DefFactory.CreateSession()
	err = session.(reactor.Session).InitBaseSession(conn, idmaker.MakeSidSafeLock(), s.epollReactor)
	if nil != err {
		logger.Error("Server Accept -> CreateSession err:", err)
		conn.Close()
		return 0, err
	}

	session.SetRWState(reactor.MASK_READ)
	s.epollReactor.AddHandler(session.GetRWState(), session)
	session.OnOpen()

	return 1, nil
}

func (s *Server) DoWrite() (bool, error) {
	return false, nil
}

func (s *Server) DoClose() {
	logger.Debug("Server.DoClose info:", s.Info())
	s.listener.Close()
	s.listener = nil

	s.epollReactor.DestroyReactor()
	s.epollReactor = nil
}

func (s *Server) OnOpen() {
}

func (s *Server) OnClose() {
}

func (s *Server) OnRead() {
}

func (s *Server) run(network, addr string) error {
	// create listener
	var err error
	s.listener, err = net.Listen(network, addr)
	if nil != err {
		logger.Error("Server Listen err:", err)
		return err
	}

	// get fd
	var fdListen uintptr
	syscallListen, err := s.listener.(*net.TCPListener).SyscallConn()
	if nil != err {
		logger.Error("Server Listen.SyscallConn err:", err)
		return err
	}
	err = syscallListen.Control(func(fd uintptr) { fdListen = fd })
	if nil != err {
		logger.Error("Server Listen.SyscallConn.Control err:", err)
		return err
	}
	s.lnFD = int32(fdListen)
	s.SetRWState(reactor.MASK_READ)

	// register listen epoll
	s.epollReactor.CreateReactor()
	s.epollReactor.AddHandler(s.GetRWState(), s)
	s.epollReactor.LoopReactor()

	return nil
}
