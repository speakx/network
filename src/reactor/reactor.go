package reactor

import (
	"net"
)

const (
	MASK_UNKNOWN      = 0x0
	MASK_READ         = 0x1
	MASK_WRITE        = 0x2
	MASK_READOVERFLOW = 0x4
)

type Reactor interface {
	CreateReactor() error
	DestroyReactor()
	LoopReactor()
	AddHandler(opt uint32, handler Handler)
	ModHandler(opt uint32, handler Handler)
	DelHandler(handler Handler)
	AddReadOverFlow(handler Handler)
}

type Handler interface {
	Info() string
	GetId() int32
	GetFd() int32
	SetRWState(rw uint32)
	ModRWState(rw uint32, add bool) uint32
	GetRWState() uint32

	DoRead() (int, error)
	DoWrite() (bool, error)
	DoClose()

	OnOpen()
	OnClose()
	OnRead()
}

type TranslaterBuffer interface {
	SetReadTag(tag, tagSize int)
	GetReadTag() (int, int)
}

type Translator interface {
	Pack(handler TranslaterBuffer, data []byte) []byte
	UnPack(handler TranslaterBuffer, buf []byte) (int, []byte)
	GetHeadTag(handler TranslaterBuffer) (int, int)
}

type Session interface {
	InitBaseSession(conn net.Conn, sid int32, reactor Reactor) error
}
