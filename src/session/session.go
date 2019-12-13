package session

import (
	"environment/logger"
	"fmt"
	"net"
	"network/bufpool"
	"network/reactor"
)

type BaseSession struct {
	sid         int32
	connFD      int32
	conn        net.Conn
	reactor     reactor.Reactor
	WriteBuffer *bufpool.CircleSlidingBuffer
	ReadBuffer  *bufpool.CircleSlidingBuffer
}

func (b *BaseSession) InitBaseSession(conn net.Conn, sid int32, reactor reactor.Reactor) error {
	var connFD uintptr
	switch realConn := conn.(type) {
	case *net.TCPConn:
		syscallConn, err := realConn.SyscallConn()
		if nil != err {
			return err
		}

		err = syscallConn.Control(func(fd uintptr) { connFD = fd })
		if nil != err {
			return err
		}
	case *net.UDPConn:
		panic(fmt.Errorf("accept net.UDPConn not supported"))
	}

	b.sid = sid
	b.connFD = int32(connFD)
	b.conn = conn
	b.reactor = reactor
	b.WriteBuffer = bufpool.NewCircleSlidingBuffer(1024 * 2)
	b.ReadBuffer = bufpool.NewCircleSlidingBuffer(1024 * 2)

	return nil
}

func (b *BaseSession) Info() string {
	return fmt.Sprintf("Session remote addr:%v", b.conn.RemoteAddr())
}

func (b *BaseSession) GetId() int32 {
	return b.sid
}

func (b *BaseSession) GetFD() int32 {
	return b.connFD
}

func (b *BaseSession) DoRead() (*bufpool.SlidingBuffer, error) {
	buf := bufpool.DefBufPool.Alloc()
	n, err := b.conn.Read(buf)
	sb := bufpool.NewSlidingBufferWithData(buf, n)
	if false == b.ReadBuffer.PushSlidingBuffer(sb) {
		logger.Error("BaseSession.DoRead overflow, info:", b.Info())
		sb.ReleaseSlidingBuffer()
		sb = nil
	}
	return sb, err
}

func (b *BaseSession) DoWrite() (int, error) {
	write := 0
	for {
		sb := b.WriteBuffer.FrontSlidingBuffer()
		if nil == sb {
			break
		}

		writeBytes := sb.GetWrited(0)
		n1 := len(writeBytes)
		n, err := b.conn.Write(writeBytes)
		needWriteBytes := sb.GetWrited(n)
		write += n

		if len(needWriteBytes) == 0 {
			b.WriteBuffer.RemoveFrontSlidingBuffer()
		}

		if nil != err {
			return write, err
		}

		if n < n1 {
			b.reactor.AddHandler(reactor.MASK_READ|reactor.MASK_WRITE, b)
			return write, nil
		}
	}

	b.reactor.ModHandler(reactor.MASK_READ, b)
	return write, nil
}

func (b *BaseSession) DoClose() {
	logger.Debug("BaseSession.DoClose info:", b.Info())
	if nil != b.conn {
		b.conn.Close()
		b.conn = nil
	}
	b.reactor = nil

	for {
		sb := b.WriteBuffer.FrontSlidingBuffer()
		if nil == sb {
			break
		}
		sb.ReleaseSlidingBuffer()
		b.WriteBuffer.RemoveFrontSlidingBuffer()
	}
	for {
		sb := b.ReadBuffer.FrontSlidingBuffer()
		if nil == sb {
			break
		}
		sb.ReleaseSlidingBuffer()
		b.ReadBuffer.RemoveFrontSlidingBuffer()
	}
}

func (b *BaseSession) OnOpen() {
}

func (b *BaseSession) OnClose() {
}

func (b *BaseSession) OnRead() {
}

func (b *BaseSession) SendString(v string) error {
	sb := bufpool.NewSlidingBuffer(0)
	sb.Write([]byte(v))
	if false == b.WriteBuffer.PushSlidingBuffer(sb) {
		return fmt.Errorf("session send string, circle sliding buffer overflow")
	}
	b.reactor.ModHandler(
		reactor.MASK_READ|reactor.MASK_WRITE,
		b)
	return nil
}
