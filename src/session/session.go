package session

import (
	"fmt"
	"net"
	"network/bufpool"
	"network/reactor"
)

type baseSession struct {
	sid                 int32
	conn                net.Conn
	fd                  int32
	reactor             reactor.Reactor
	circleSlidingBuffer *bufpool.CircleSlidingBuffer
}

func newBaseSession(sid int32, conn net.Conn, fd int32, reactor reactor.Reactor) *baseSession {
	return &baseSession{
		sid:                 sid,
		conn:                conn,
		fd:                  fd,
		reactor:             reactor,
		circleSlidingBuffer: bufpool.NewCircleSlidingBuffer(1024),
	}
}

func (b *baseSession) Sid() int32 {
	return b.sid
}

func (b *baseSession) FD() int32 {
	return b.fd
}

func (b *baseSession) Addr() string {
	return b.conn.RemoteAddr().String()
}

func (b *baseSession) Close() {
	if nil != b.conn {
		b.conn.Close()
	}
	b.reactor = nil
}

func (b *baseSession) OnRead() (*bufpool.SlidingBuffer, error) {
	buf := bufpool.DefBufPool.Alloc()
	n, err := b.conn.Read(buf)
	return bufpool.NewSlidingBufferWithData(buf, n), err
}

func (b *baseSession) OnWrite() (int, error) {
	sb := b.circleSlidingBuffer.FrontSlidingBuffer()
	if nil == sb {
		b.reactor.ModHandler(reactor.MASK_READ, b.FD(), b.Sid())
		return 0, nil
	}

	writeBytes := sb.Read(0)
	n, err := b.conn.Write(writeBytes)
	needWriteBytes := sb.Read(n)

	if len(needWriteBytes) == 0 {
		b.circleSlidingBuffer.RemoveFrontSlidingBuffer()
		b.reactor.ModHandler(reactor.MASK_READ, b.FD(), b.Sid())
	} else {
		b.reactor.AddHandler(reactor.MASK_READ|reactor.MASK_WRITE, b.FD(), b.Sid())
	}
	return n, err
}

func (b *baseSession) SendString(v string) error {
	sb := bufpool.NewSlidingBuffer(0)
	sb.Write([]byte(v))
	if false == b.circleSlidingBuffer.PushSlidingBuffer(sb) {
		return fmt.Errorf("session send string, circle sliding buffer overflow")
	}
	b.reactor.ModHandler(reactor.MASK_READ|reactor.MASK_WRITE, b.FD(), b.Sid())
	return nil
}
