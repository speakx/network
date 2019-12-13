package reactor

import (
	"environment/logger"
	"fmt"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	handlerEventOnRead = iota
	handlerEventOnClose
)

type handlerEvent struct {
	handler Handler
	event   int
}

type EpollReactor struct {
	epollFD             int
	events              []unix.EpollEvent
	padToHandler        map[int32]Handler
	onHandlerEventChan  chan *handlerEvent
	onReadGoroutineLock chan bool
	sync.Mutex
}

func NewEpollReactor(maxEvent, goroutineLimit int) *EpollReactor {
	return &EpollReactor{
		events:              make([]unix.EpollEvent, maxEvent),
		padToHandler:        make(map[int32]Handler),
		onHandlerEventChan:  make(chan *handlerEvent, maxEvent),
		onReadGoroutineLock: make(chan bool, goroutineLimit),
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
		Fd:     fd,
		Pad:    pad,
	}
	return unix.EpollCtl(e.epollFD, epollOp, int(fd), event)
}

func (e *EpollReactor) unregisterEpollHandler(fd, pad int32) error {
	event := &unix.EpollEvent{
		Fd:  fd,
		Pad: pad,
	}
	return unix.EpollCtl(e.epollFD, unix.EPOLL_CTL_DEL, int(fd), event)
}

func (e *EpollReactor) CreateReactor() error {
	go func() {
		e.loopHandlerEvent()
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

		logger.Debug("Reactor.Epoll.Wait events loop start n:", n, " err:", err)
		for i := 0; i < n; i++ {
			// error
			if e.events[i].Events&unix.POLLERR == unix.POLLERR || e.events[i].Events&unix.POLLHUP == unix.POLLHUP || e.events[i].Events&unix.POLLNVAL == unix.POLLNVAL || e.events[i].Events&unix.POLLRDHUP == unix.POLLRDHUP {
				if e.events[i].Events&unix.POLLERR == unix.POLLERR {
					logger.Info("Reactor.Epoll.Wait event[", i, "]:POLLERR event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				} else if e.events[i].Events&unix.POLLHUP == unix.POLLHUP {
					logger.Info("Reactor.Epoll.Wait event[", i, "]:POLLHUP event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				} else if e.events[i].Events&unix.POLLNVAL == unix.POLLNVAL {
					logger.Info("Reactor.Epoll.Wait event[", i, "]:POLLNVAL event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				} else if e.events[i].Events&unix.POLLRDHUP == unix.POLLRDHUP {
					logger.Info("Reactor.Epoll.Wait event[", i, "]:POLLRDHUP event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				}
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event[", i, "]:POLLERR|POLLHUP|POLLNVAL|POLLRDHUP event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}
				e.DelHandler(handler)
				e.onHandlerEventChan <- &handlerEvent{event: handlerEventOnClose, handler: handler}
				continue
			}

			// read  || e.events[i].Events&unix.POLLPRI == unix.POLLPRI
			if e.events[i].Events&unix.POLLIN == unix.POLLIN {
				logger.Debug("Reactor.Epoll.Wait event[", i, "]:POLLIN|POLLPRI event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad)
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event[", i, "]:POLLIN|POLLPRI event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}

				sb, err := handler.DoRead()
				logger.Debug("Reactor.Epoll.Wait event[", i, "]:POLLIN|POLLPRI event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " read:", sb.WriteLen(), " err:", err)
				if nil != err {
					logger.Error("Reactor.Epoll.Wait event[", i, "]:POLLIN|POLLPRI event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " err:", err)
					e.DelHandler(handler)
					e.onHandlerEventChan <- &handlerEvent{event: handlerEventOnClose, handler: handler}
				}

				if nil != sb && sb.WriteLen() > 0 {
					e.onHandlerEventChan <- &handlerEvent{event: handlerEventOnRead, handler: handler}
				}
			}

			// write
			if e.events[i].Events&unix.POLLOUT == unix.POLLOUT {
				logger.Debug("Reactor.Epoll.Wait event[", i, "]:POLLOUT event.fd:", e.events[i].Fd, " event.Pad:", e.events[i].Pad)
				handler := e.findHandler(e.events[i].Pad)
				if nil == handler {
					logger.Error("Reactor.Epoll.Wait event[", i, "]:POLLOUT event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " hander not found")
					e.unregisterEpollHandler(e.events[i].Fd, e.events[i].Pad)
					continue
				}

				n, err := handler.DoWrite()
				logger.Debug("Reactor.Epoll.Wait event[", i, "]:POLLOUT event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " write:", n, " err:", err)
				if nil != err {
					logger.Error("Reactor.Epoll.Wait event[", i, "]:POLLOUT event.fd:", e.events[i].Fd, " event.sid:", e.events[i].Pad, " err:", err)
					e.DelHandler(handler)
					e.onHandlerEventChan <- &handlerEvent{event: handlerEventOnClose, handler: handler}
				}
			}
		}
		logger.Debug("Reactor.Epoll.Wait events loop end n:", n, " err:", err)
	}
}

func (e *EpollReactor) loopHandlerEvent() {
	for {
		he, ok := <-e.onHandlerEventChan
		if false == ok {
			return
		}

		e.goroutineAcquire()
		go func() {
			switch he.event {
			case handlerEventOnRead:
				he.handler.OnRead()
			case handlerEventOnClose:
				he.handler.OnClose()
				he.handler.DoClose()
			default:
				panic(fmt.Errorf("unsupport handler.event:%v", he.event))
			}
			e.goroutineRelease()
		}()
	}
}

func (e *EpollReactor) goroutineAcquire() {
	e.onReadGoroutineLock <- true
}

func (e *EpollReactor) goroutineRelease() {
	<-e.onReadGoroutineLock
}

func (e *EpollReactor) registerHandler(handler Handler) {
	e.Lock()
	e.padToHandler[handler.GetId()] = handler
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler register fd:", handler.GetFD(), " sid:", handler.GetId(), " ", handler.Info())
}

func (e *EpollReactor) unregisterHandler(handler Handler) {
	e.Lock()
	delete(e.padToHandler, handler.GetId())
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler unregister fd:", handler.GetFD(), " sid:", handler.GetId(), " ", handler.Info())
}

func (e *EpollReactor) findHandler(pad int32) Handler {
	e.Lock()
	handler, _ := e.padToHandler[pad]
	e.Unlock()
	return handler
}
