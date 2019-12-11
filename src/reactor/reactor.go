package reactor

import (
	"network/bufpool"
)

const (
	MASK_READ  = 0x1
	MASK_WRITE = 0x2
)

var NetworkHandlerImp NetworkHandler

type Reactor interface {
	AddHandler(opt int, fd, pad int32)
	ModHandler(opt int, fd, pad int32)
	DelHandler(fd int32)
}

type NetworkHandler interface {
	OnRead(session Session, sb *bufpool.SlidingBuffer)
	OnReadFinish(session Session)
}

type Session interface {
	Sid() int32
	FD() int32
	Addr() string
	Close()
	OnRead() (*bufpool.SlidingBuffer, error)
	OnWrite() (int, error)
	SendString(v string) error
}

func InitNetworkHandler(handler NetworkHandler) {
	NetworkHandlerImp = handler
}
