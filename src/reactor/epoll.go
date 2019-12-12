package reactor

import (
	"environment/logger"
	"sync"

	"golang.org/x/sys/unix"
)

type EpollReactor struct {
	epollFD      int
	events       []unix.EpollEvent
	padToHandler map[int32]Handler
	onReadChan   chan Handler
	onCloseChan  chan Handler
	sync.Mutex
}

func NewEpollReactor(maxEvent int) *EpollReactor {
	return &EpollReactor{
		events:       make([]unix.EpollEvent, maxEvent),
		padToHandler: make(map[int32]Handler),
		onReadChan:   make(chan Handler, maxEvent),
		onCloseChan:  make(chan Handler, maxEvent),
	}
}

func (e *EpollReactor) AddHandler(opt int, handler Handler) {
	e.registerHandler(handler)
	err := e.registerEpollHandler(unix.EPOLL_CTL_ADD, opt, handler.GetFD(), handler.GetId())
	if nil != err {
		logger.Error("Reactor.Epoll.AddHandler err:", err)
		panic(err)
	}
	logger.Debug("Reactor.Epoll.AddHandler fd:", handler.GetFD(), " opt:", opt, " sid:", handler.GetId())
}

func (e *EpollReactor) ModHandler(opt int, handler Handler) {
	err := e.registerEpollHandler(unix.EPOLL_CTL_MOD, opt, handler.GetFD(), handler.GetId())
	if nil != err {
		logger.Error("Reactor.Epoll.ModHandler err:", err)
		panic(err)
	}
	logger.Debug("Reactor.Epoll.ModHandler fd:", handler.GetFD(), " opt:", opt, " sid:", handler.GetId())
}

func (e *EpollReactor) DelHandler(handler Handler) {
	err := e.unregisterEpollHandler(handler.GetFD(), handler.GetId())
	if nil != err {
		logger.Warn("Reactor.Epoll.DelHandler err:", err)
	}
	e.unregisterHandler(handler)
	logger.Debug("Reactor.Epoll.DelHandler fd:", handler.GetFD())
}

func (e *EpollReactor) registerEpollHandler(epollOp, opt int, fd, pad int32) error {
	events := uint32(0)
	if opt&MASK_READ == MASK_READ {
		events |= unix.POLLIN
	}
	if opt&MASK_WRITE == MASK_WRITE {
		events |= unix.POLLOUT
	}

	event := &unix.EpollEvent{
		Events: events,
		Pad:    pad,
	}
	return unix.EpollCtl(e.epollFD, epollOp, int(fd), event)
}

func (e *EpollReactor) unregisterEpollHandler(fd, pad int32) error {
	return unix.EpollCtl(e.epollFD, unix.EPOLL_CTL_DEL, int(fd), nil)
}

func (e *EpollReactor) CreateReactor() error {
	go func() {
		e.loopOnReadEvent()
	}()
	go func() {
		e.loopOnCloseEvent()
	}()

	var err error
	e.epollFD, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if nil != err {
		logger.Error("Reactor EpollCreate1 err:", err)
		return err
	}
	return nil
}

func (e *EpollReactor) DestroyReactor() {
	unix.Close(e.epollFD)
	e.epollFD = 0
}

func (e *EpollReactor) LoopReactor() {
	for {
		n, err := unix.EpollWait(e.epollFD, e.events, -1)
		if nil != err {
			logger.Error("Reactor.Epoll.Wait n:", n, " err:", err)
			panic(err)
		}

		logger.Debug("Reactor.Epoll.Wait n:", n, " err:", err)
		for i := 0; i < n; i++ {
			if e.events[i].Events&unix.POLLPRI == unix.POLLPRI {
				logger.Error("Reactor.Epoll.Wait event:POLLPRI event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
			}
			if e.events[i].Events&unix.POLLRDHUP == unix.POLLRDHUP {
				logger.Error("Reactor.Epoll.Wait event:POLLRDHUP event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
			}
			if e.events[i].Events&unix.POLLNVAL == unix.POLLNVAL {
				logger.Error("Reactor.Epoll.Wait event:POLLNVAL event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
			}

			// error
			if e.events[i].Events&unix.POLLERR == unix.POLLERR || e.events[i].Events&unix.POLLHUP == unix.POLLHUP {
				if e.events[i].Events&unix.POLLERR == unix.POLLERR {
					logger.Info("Reactor.Epoll.Wait event:POLLERR event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				} else {
					logger.Info("Reactor.Epoll.Wait event:POLLHUP event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				}
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event:POLLERR|event:POLLHUP event.fd:", e.events[i].Fd, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}
				e.DelHandler(handler)
				e.onCloseChan <- handler
				continue
			}

			// read
			if e.events[i].Events&unix.POLLIN == unix.POLLIN {
				logger.Debug("Reactor.Epoll.Wait event:POLLIN event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event:POLLIN event.fd:", e.events[i].Fd, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}

				sb, err := handler.DoRead()
				logger.Debug("Reactor.Epoll.Wait event:POLLIN event.fd:", e.events[i].Fd,
					" event.sid:", e.events[i].Pad, " read:", sb.WriteLen(), " err:", err)
				if nil != err {
					logger.Error("Reactor.Epoll.Wait POLLIN err:", err)
					e.DelHandler(handler)
					e.onCloseChan <- handler
				}

				if nil != sb && sb.WriteLen() > 0 {
					e.onReadChan <- handler
				}
			}

			// write
			if e.events[i].Events&unix.POLLOUT == unix.POLLOUT {
				logger.Debug("Reactor.Epoll.Wait event:POLLOUT event.fd:", e.events[i].Fd, " event.Pad:", e.events[i].Pad)
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event:POLLOUT event.fd:", e.events[i].Fd, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}

				n, err := handler.DoWrite()
				logger.Debug("Reactor.Epoll.Wait event:POLLOUT event.fd:", e.events[i].Fd,
					" event.sid:", e.events[i].Pad, " write:", n, " err:", err)
				if nil != err {
					e.DelHandler(handler)
				}
			}
		}
	}
}

func (e *EpollReactor) loopOnReadEvent() {
	for {
		handler, ok := <-e.onReadChan
		if false == ok {
			return
		}

		handler.OnRead()
	}
}

func (e *EpollReactor) loopOnCloseEvent() {
	for {
		handler, ok := <-e.onCloseChan
		if false == ok {
			return
		}

		handler.OnClose()
		handler.DoClose()
	}
}

func (e *EpollReactor) registerHandler(handler Handler) {
	e.Lock()
	e.padToHandler[handler.GetId()] = handler
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler register:", handler.Info())
}

func (e *EpollReactor) unregisterHandler(handler Handler) {
	e.Lock()
	delete(e.padToHandler, handler.GetId())
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler unregister:", handler.Info())
}

func (e *EpollReactor) findHandler(pad int32) Handler {
	e.Lock()
	handler, _ := e.padToHandler[pad]
	e.Unlock()
	return handler
}
