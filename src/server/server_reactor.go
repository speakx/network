package server

import (
	"environment/logger"
	"fmt"
	"network/reactor"

	"golang.org/x/sys/unix"
)

func (s *Server) AddHandler(opt int, fd, pad int32) {
	err := s.registerEpollHandler(unix.EPOLL_CTL_ADD, opt, fd, pad)
	if nil != err {
		logger.Error("Server EpollCtl Add err:", err)
		panic(err)
	}
	logger.Debug("Server.AddHandler fd:", fd, " opt:", opt, " pad:", pad)
}

func (s *Server) ModHandler(opt int, fd, pad int32) {
	err := s.registerEpollHandler(unix.EPOLL_CTL_MOD, opt, fd, pad)
	if nil != err {
		logger.Error("Server EpollCtl Mod err:", err)
		panic(err)
	}
	logger.Debug("Server.ModHandler fd:", fd, " opt:", opt, " pad:", pad)
}

func (s *Server) DelHandler(fd int32) {
	err := unix.EpollCtl(s.fdEpoll, unix.EPOLL_CTL_DEL, int(fd), nil)
	if nil != err {
		logger.Warn("Server EpollCtl Del err:", err)
	}
	logger.Debug("Server.DelHandler fd:", fd)
}

func (s *Server) registerEpollHandler(epollOp, opt int, fd, pad int32) error {
	events := uint32(0)
	if opt&reactor.MASK_READ == reactor.MASK_READ {
		events |= unix.POLLIN
	}
	if opt&reactor.MASK_WRITE == reactor.MASK_WRITE {
		events |= unix.POLLOUT
	}

	event := &unix.EpollEvent{
		Events: events,
		Pad:    pad,
	}
	return unix.EpollCtl(s.fdEpoll, epollOp, int(fd), event)
}

func (s *Server) handleEpollEvent(fd int) {
	logger.Debug("Server.EpollWait fd:", fd)
	for {
		n, err := unix.EpollWait(fd, s.events, -1)
		if nil != err {
			logger.Error("Server.EpollWait n:", n, " err:", err)
			panic(err)
		}

		logger.Debug("Server.EpollWait n:", n, " err:", err)
		for i := 0; i < n; i++ {
			switch s.events[i].Events {
			case unix.POLLIN:
				logger.Debug("Server.EpollWait event:POLLIN event.fd:", s.events[i].Fd, " event:", s.events[i])
				if 0 == s.events[i].Pad {
					// Listen -> Accept
					err := s.onListen()
					if nil != err {
						panic(err)
					}
				} else {
					// Read & Read.EOF(FIN)
					session := s.sidToSession(s.events[i].Pad)
					if nil == session {
						panic(fmt.Errorf("POLLIN sid:%v to session not found", s.events[i].Pad))
					}

					sb, err := session.OnRead()
					logger.Debug("POLLIN sid:", session.Sid(), " read:", sb.WriteLen(), " err:", err)
					if nil == err {
						if nil != reactor.NetworkHandlerImp {
							reactor.NetworkHandlerImp.OnRead(session, sb)
						}
					} else {
						s.removeSession(session)
					}
				}
			case unix.POLLOUT:
				// Write
				logger.Debug("Server.EpollWait event:POLLOUT event.fd:", s.events[i].Fd)
				session := s.sidToSession(s.events[i].Pad)
				if nil == session {
					panic(fmt.Errorf("POLLOUT sid:%v to session not found", s.events[i].Pad))
				}

				n, err := session.OnWrite()
				logger.Debug("POLLOUT sid:", session.Sid(), " write:", n, " err:", err)
				if nil != err {
					s.removeSession(session)
				}
			case unix.POLLPRI:
				logger.Error("Server.EpollWait event:POLLPRI event.fd:", s.events[i].Fd)
			case unix.POLLRDHUP:
				logger.Error("Server.EpollWait event:POLLRDHUP event.fd:", s.events[i].Fd)
			case unix.POLLERR:
				logger.Error("Server.EpollWait event:POLLERR event.fd:", s.events[i].Fd)
			case unix.POLLHUP:
				logger.Error("Server.EpollWait event:POLLHUP event.fd:", s.events[i].Fd)
			case unix.POLLNVAL:
				logger.Error("Server.EpollWait event:POLLNVAL event.fd:", s.events[i].Fd)
			}
		}
	}
}
