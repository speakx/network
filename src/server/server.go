package server

import (
	"fmt"
	"net"
	"network/bufpool"
	"network/idmaker"
	"network/reactor"
	"network/register"
	"time"

	"environment/logger"
)

type Server struct {
	sid          int32
	lnFD         int32
	listener     net.Listener
	epollReactor *reactor.EpollReactor
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

func (s *Server) GetFD() int32 {
	return s.lnFD
}

func (s *Server) DoRead() (*bufpool.SlidingBuffer, error) {
	// accept
	conn, err := s.listener.Accept()
	if nil != err {
		logger.Error("Server Accept err:", err)
		return nil, err
	}
	logger.Info("Server.Accept addr:", conn.RemoteAddr())

	// create session
	session := register.DefFactory.CreateSession()
	err = session.(reactor.Session).InitBaseSession(conn, idmaker.MakeSidSafeLock(), s.epollReactor)
	if nil != err {
		logger.Error("Server Accept -> CreateSession err:", err)
		conn.Close()
		return nil, err
	}
	s.epollReactor.AddHandler(reactor.MASK_READ, session)
	session.OnOpen()

	return bufpool.NewSlidingBuffer(0), nil
}

func (s *Server) DoWrite() (int, error) {
	return 0, nil
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

	// register listen epoll
	s.epollReactor.CreateReactor()
	s.epollReactor.AddHandler(reactor.MASK_READ, s)
	s.epollReactor.LoopReactor()

	return nil
}
