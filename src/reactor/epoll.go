package reactor

import (
	"environment/logger"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	handlerUnknown = iota
	handlerEventDoRead
	handlerEventDoWrite
	handlerEventDoClose
	handlerEventOnRead
)

type handlerEvent struct {
	handler  Handler
	event    int
	overFlow bool
}

type EpollReactor struct {
	epollFD            int
	events             []unix.EpollEvent
	padToHandler       map[int32]Handler
	onReactDoReadChan  chan *handlerEvent
	onReactDoWriteChan chan *handlerEvent
	onReactOnReadChan  chan *handlerEvent
	padToReadOverFlow  map[int32]Handler
	eventPool          *sync.Pool
	sync.Mutex
}

func NewEpollReactor(maxEvent, goroutineLimit int) *EpollReactor {
	er := &EpollReactor{
		events:             make([]unix.EpollEvent, maxEvent),
		padToHandler:       make(map[int32]Handler),
		onReactDoReadChan:  make(chan *handlerEvent, maxEvent),
		onReactDoWriteChan: make(chan *handlerEvent, maxEvent),
		onReactOnReadChan:  make(chan *handlerEvent, maxEvent),
		padToReadOverFlow:  make(map[int32]Handler),
		eventPool:          &sync.Pool{New: func() interface{} { return &handlerEvent{} }},
	}

	go er.loopReactDoRead()
	go er.loopReactDoWrite()
	for i := 0; i < goroutineLimit; i++ {
		go er.loopReactOnRead()
	}

	return er
}

func (e *EpollReactor) AddHandler(opt uint32, handler Handler) {
	e.registerHandler(handler)
	err := e.registerEpollHandler(unix.EPOLL_CTL_ADD, opt, handler.GetFd(), handler.GetId())
	if nil != err {
		logger.Error("Reactor.Epoll.AddHandler fd:", handler.GetFd(), " sid:", handler.GetId(), " opt:", opt, " err:", err)
		panic(err)
	}
	logger.Debug("Reactor.Epoll.AddHandler fd:", handler.GetFd(), " sid:", handler.GetId(), " opt:", opt)
}

func (e *EpollReactor) ModHandler(opt uint32, handler Handler) {
	err := e.registerEpollHandler(unix.EPOLL_CTL_MOD, opt, handler.GetFd(), handler.GetId())
	if nil != err {
		logger.Warn("Reactor.Epoll.ModHandler fd:", handler.GetFd(), " sid:", handler.GetId(), " opt:", opt, " err:", err)
		return
	}
	logger.Debug("Reactor.Epoll.ModHandler fd:", handler.GetFd(), " sid:", handler.GetId(), " opt:", opt)
}

func (e *EpollReactor) DelHandler(handler Handler) {
	err := e.unregisterEpollHandler(handler.GetFd(), handler.GetId())
	if nil != err {
		logger.Warn("Reactor.Epoll.DelHandler err:", err)
	}
	e.unregisterHandler(handler)
	logger.Debug("Reactor.Epoll.DelHandler fd:", handler.GetFd())
}

func (e *EpollReactor) AddReadOverFlow(handler Handler) {
	e.Lock()
	e.padToReadOverFlow[handler.GetId()] = handler
	e.Unlock()
}

func (e *EpollReactor) registerEpollHandler(epollOp int, opt uint32, fd, pad int32) error {
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
	logger.Debug("Reactor.Epoll.registerEpollHandler fd:", fd, " sid:", pad, " opt:", opt)
	return unix.EpollCtl(e.epollFD, epollOp, int(fd), event)
}

func (e *EpollReactor) unregisterEpollHandler(fd, pad int32) error {
	event := &unix.EpollEvent{
		Fd:  fd,
		Pad: pad,
	}
	logger.Debug("Reactor.Epoll.unregisterEpollHandler fd:", fd, " sid:", pad)
	return unix.EpollCtl(e.epollFD, unix.EPOLL_CTL_DEL, int(fd), event)
}

func (e *EpollReactor) allocHandlerEvent(event int, handler Handler) *handlerEvent {
	he := e.eventPool.Get().(*handlerEvent)
	he.event = event
	he.handler = handler
	return he
}

func (e *EpollReactor) CreateReactor() error {
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

		logger.Debug("Reactor.Epoll.Wait <START> events loop n:", n, " err:", err)
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
				e.onReactDoReadChan <- &handlerEvent{event: handlerEventDoClose, handler: handler}
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

				e.Lock()
				rw := handler.ModRWState(MASK_READ, false)
				e.registerEpollHandler(unix.EPOLL_CTL_MOD, rw, handler.GetFd(), handler.GetId())
				e.Unlock()
				e.onReactDoReadChan <- &handlerEvent{event: handlerEventDoRead, handler: handler}
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

				e.Lock()
				rw := handler.ModRWState(MASK_WRITE, false)
				e.registerEpollHandler(unix.EPOLL_CTL_MOD, rw, handler.GetFd(), handler.GetId())
				e.Unlock()
				e.onReactDoWriteChan <- &handlerEvent{event: handlerEventDoWrite, handler: handler}
			}
		}
		logger.Debug("Reactor.Epoll.Wait <END> events loop n:", n, " err:", err)
	}
}

func (e *EpollReactor) loopReactDoRead() {
	for {
		he, ok := <-e.onReactDoReadChan
		if false == ok {
			return
		}

		if handlerEventDoClose == he.event {
			e.DelHandler(he.handler)
			he.handler.OnClose()
			he.handler.DoClose()
			continue
		}

		n, err := he.handler.DoRead()
		logger.Debug("Reactor.Epoll.Loop <DoRead> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " read packet:", n, " read err:", err)
		if nil != err {
			logger.Error("Reactor.Epoll.Loop <DoRead> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " err:", err)
			e.DelHandler(he.handler)
			he.handler.OnClose()
			he.handler.DoClose()
			continue
		}

		if (he.handler.GetRWState() & MASK_READOVERFLOW) == MASK_READOVERFLOW {
			logger.Debug("Reactor.Epoll.Loop <DoRead> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " read packet:", n, " read err:", err, " MASK_READOVERFLOW")
			e.Lock()
			he.handler.ModRWState(MASK_READOVERFLOW, false)
			e.Unlock()
			e.onReactOnReadChan <- &handlerEvent{event: handlerEventOnRead, handler: he.handler, overFlow: true}
			continue
		}

		if n > 0 {
			e.onReactOnReadChan <- &handlerEvent{event: handlerEventOnRead, handler: he.handler}
		}

		e.Lock()
		rw := he.handler.ModRWState(MASK_READ, true)
		e.registerEpollHandler(unix.EPOLL_CTL_MOD, rw, he.handler.GetFd(), he.handler.GetId())
		e.Unlock()
	}
}

func (e *EpollReactor) loopReactDoWrite() {
	for {
		he, ok := <-e.onReactDoWriteChan
		if false == ok {
			return
		}

		continueWrite, err := he.handler.DoWrite()
		logger.Debug("Reactor.Epoll.Loop <DoWrite> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " continue:", continueWrite, " err:", err)
		if nil != err {
			logger.Error("Reactor.Epoll.Loop <DoWrite> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " err:", err)
			e.onReactDoReadChan <- &handlerEvent{event: handlerEventDoClose, handler: he.handler}
		}

		if true == continueWrite {
			e.Lock()
			rw := he.handler.ModRWState(MASK_WRITE, true)
			e.registerEpollHandler(unix.EPOLL_CTL_MOD, rw, he.handler.GetFd(), he.handler.GetId())
			e.Unlock()
		}
	}
}

func (e *EpollReactor) loopReactOnRead() {
	for {
		he, ok := <-e.onReactOnReadChan
		if false == ok {
			return
		}

		logger.Debug("Reactor.Epoll.Loop <OnRead> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " overflow:", he.overFlow)
		he.handler.OnRead()

		if he.overFlow {
			logger.Debug("Reactor.Epoll.Loop <OnRead> event.fd:", he.handler.GetFd(), " event.sid:", he.handler.GetId(), " MASK_READOVERFLOW")

			e.Lock()
			rw := he.handler.ModRWState(MASK_READ, true)
			e.registerEpollHandler(unix.EPOLL_CTL_MOD, rw, he.handler.GetFd(), he.handler.GetId())
			e.Unlock()
		}
	}
}

func (e *EpollReactor) registerHandler(handler Handler) {
	e.Lock()
	e.padToHandler[handler.GetId()] = handler
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler register fd:", handler.GetFd(), " sid:", handler.GetId(), " ", handler.Info())
}

func (e *EpollReactor) unregisterHandler(handler Handler) {
	e.Lock()
	delete(e.padToHandler, handler.GetId())
	e.Unlock()
	logger.Info("Reactor.Epoll.Handler unregister fd:", handler.GetFd(), " sid:", handler.GetId(), " ", handler.Info())
}

func (e *EpollReactor) findHandler(pad int32) Handler {
	e.Lock()
	handler, _ := e.padToHandler[pad]
	e.Unlock()
	return handler
}
