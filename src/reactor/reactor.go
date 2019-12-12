package reactor

import (
	"net"
	"network/bufpool"
)

const (
	MASK_READ  = 0x1
	MASK_WRITE = 0x2
)

type Reactor interface {
	CreateReactor() error
	DestroyReactor()
	LoopReactor()
	AddHandler(opt int, handler Handler)
	ModHandler(opt int, handler Handler)
	DelHandler(handler Handler)
}

type Handler interface {
	Info() string
	GetId() int32
	GetFD() int32

	DoRead() (*bufpool.SlidingBuffer, error)
	DoWrite() (int, error)
	DoClose()

	OnOpen()
	OnClose()
	OnRead()
}

type Session interface {
	InitBaseSession(conn net.Conn, sid int32, reactor Reactor) error
}
