package session

import (
	"container/list"
	"environment/logger"
	"fmt"
	"net"
	"network/bufpool"
	"network/reactor"
	"network/register"
	"sync"
)

type BaseSession struct {
	sid                int32
	connFD             int32
	conn               net.Conn
	rw                 uint32
	readTag            int
	readTagSize        int
	reactor            reactor.Reactor
	onReadPacketBuffer *bufpool.CircleSlidingBuffer
	// WriteBuffer        *bufpool.CircleSlidingBuffer
	// onReadPacketBuffer *list.List // 通过List实现读缓存，对内存开销太大
	writeBuffer *list.List
	readBuffer  *bufpool.SlidingBuffer
	readBuffer1 *bufpool.SlidingBuffer
	sync.Mutex
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
	// b.onReadPacketBuffer = list.New()
	b.writeBuffer = list.New() //bufpool.NewCircleSlidingBuffer(1024 * 2)
	b.onReadPacketBuffer = bufpool.NewCircleSlidingBuffer(1024 * 2)
	b.readBuffer = bufpool.NewSlidingBuffer(0)
	b.readBuffer1 = bufpool.NewSlidingBuffer(0)
	b.readTag, b.readTagSize = register.DefFactory.GetTranslator().GetHeadTag(b)

	return nil
}

func (b *BaseSession) Info() string {
	return fmt.Sprintf("Session remote addr:%v", b.conn.RemoteAddr())
}

func (b *BaseSession) GetId() int32 {
	return b.sid
}

func (b *BaseSession) GetFd() int32 {
	return b.connFD
}

func (b *BaseSession) SetRWState(rw uint32) {
	b.rw = rw
}

func (b *BaseSession) ModRWState(rw uint32, add bool) uint32 {
	// b.Lock()
	if true == add {
		b.rw |= rw
	} else {
		b.rw &^= rw
	}
	rw = b.rw
	// b.Unlock()
	return rw
}

func (b *BaseSession) GetRWState() uint32 {
	return b.rw
}

func (b *BaseSession) SetReadTag(tag, tagSize int) {
	b.readTag = tag
	b.readTagSize = tagSize
}

func (b *BaseSession) GetReadTag() (int, int) {
	return b.readTag, b.readTagSize
}

func (b *BaseSession) DoRead() (int, error) {
	// num := 0
	// for {
	// 	// read tcp
	// 	needSize := b.readTagSize - b.readBuffer.GetWriteLen()
	// 	buf := b.readBuffer.GetData()
	// 	buf = buf[b.readBuffer.GetWriteLen():b.readTagSize]
	// 	n, err := b.conn.Read(buf)
	// 	b.readBuffer.AddWritePos(n)
	// 	if nil != err {
	// 		return 0, err
	// 	}
	// 	if n < needSize {
	// 		return num, nil
	// 	}

	// 	// unpack
	// 	unpacketN, packetData := register.DefFactory.GetTranslator().UnPack(b, b.readBuffer.GetWrited(0))
	// 	if 0 == unpacketN {
	// 		break
	// 	}

	// 	if nil != packetData {
	// 		num++
	// 		sb := bufpool.NewSlidingBuffer(0)
	// 		sb.Write(packetData)

	// 		b.Lock()
	// 		if false == b.onReadPacketBuffer.PushSlidingBuffer(sb) {
	// 			r, w := b.onReadPacketBuffer.ReadWrite()
	// 			logger.Errorf("BaseSession.DoRead overflow, info:%v r:%v w:%v", b.Info(), r, w)
	// 			sb.ReleaseSlidingBuffer()
	// 			sb = nil
	// 		}
	// 		b.Unlock()
	// 		b.readBuffer.Reset()
	// 	}
	// }

	// return num, nil

	// read tcp
	logger.Debug("BaseSession.DoRead start")
	defer logger.Debug("BaseSession.DoRead end")
	num := 0
	n, err := b.conn.Read(b.readBuffer.GetFreeData())
	b.readBuffer.AddWritePos(n)
	if nil != err {
		return 0, err
	}

	// unpack
	readBuf := b.readBuffer.GetWrited(0)
	circlePushRet := bufpool.CirclePushOK

	for {
		unpacketN, packetData := register.DefFactory.GetTranslator().UnPack(b, readBuf)
		if 0 == unpacketN {
			break
		}

		if nil != packetData {
			buf := make([]byte, len(packetData))
			copy(buf, packetData)
			sb := bufpool.NewSlidingBufferWithDataNoRelease(packetData) // packetData

			b.Lock()
			circlePushRet = b.onReadPacketBuffer.PushSlidingBuffer(sb)
			// b.onReadPacketBuffer.PushBack(sb)
			if bufpool.CirclePushOverflow == circlePushRet {
				r, w, n := b.onReadPacketBuffer.ReadWrite()
				b.Unlock()

				logger.Warnf("BaseSession.DoRead err overflow, info:%v r:%v w:%v n:%v -> num:%v", b.Info(), r, w, n, num)
				sb.ReleaseSlidingBuffer()
				b.ModRWState(reactor.MASK_READOVERFLOW, true)
			} else {
				num++
				b.Unlock()
			}
		}

		if bufpool.CirclePushOK != circlePushRet {
			break
		}

		if unpacketN > 0 {
			readBuf = b.readBuffer.GetWrited(unpacketN)
		}
		if len(readBuf) <= 0 {
			break
		}
	}

	l := len(readBuf)
	if l == 0 {
		b.readBuffer.Reset()
	} else if l > 0 {
		b.readBuffer1.Write(readBuf)
		readBuffer := b.readBuffer
		b.readBuffer = b.readBuffer1
		b.readBuffer1 = readBuffer
		b.readBuffer1.Reset()
	}

	return num, nil

	// // read tcp
	// sb := bufpool.NewSlidingBuffer(0)
	// n, err := b.conn.Read(sb.GetFreeData())
	// sb.AddWritePos(n)
	// if nil != err {
	// 	return 0, err
	// }

	// if false == b.onReadPacketBuffer.PushSlidingBuffer(sb) {
	// 	r, w := b.onReadPacketBuffer.ReadWrite()
	// 	logger.Errorf("BaseSession.DoRead overflow, info:%v r:%v w:%v", b.Info(), r, w)
	// 	sb.ReleaseSlidingBuffer()
	// 	sb = nil
	// }

	// return 1, nil
}

func (b *BaseSession) DoWrite() (bool, error) {
	continueWrite := false
	// sb := b.WriteBuffer.FrontSlidingBuffer()
	// if nil == sb {
	// 	break
	// }

	// writeBytes := sb.GetWrited(0)
	// n1 := len(writeBytes)
	// n, err := b.conn.Write(writeBytes)
	// needWriteBytes := sb.GetWrited(n)
	// write += n

	// if len(needWriteBytes) == 0 {
	// 	b.WriteBuffer.RemoveFrontSlidingBuffer()
	// }

	// if nil != err {
	// 	return write, err
	// }

	// if n < n1 {
	// 	b.reactor.AddHandler(reactor.MASK_READ|reactor.MASK_WRITE, b)
	// 	return write, nil
	// }
	b.Lock()
	for {
		e := b.writeBuffer.Front()
		if nil == e {
			break
		}

		sb := e.Value.(*bufpool.SlidingBuffer)

		buf := sb.GetWrited(0)
		n, err := b.conn.Write(buf)
		if nil != err {
			return false, err
		}

		n1 := len(sb.GetWrited(n))
		if n1 == 0 {
			b.writeBuffer.Remove(e)
			sb.ReleaseSlidingBuffer()
			continue
		} else {
			continueWrite = true
			break
		}
	}
	b.Unlock()

	return continueWrite, nil
}

func (b *BaseSession) DoClose() {
	logger.Debug("BaseSession.DoClose info:", b.Info())
	if nil != b.conn {
		b.conn.Close()
	}
}

func (b *BaseSession) OnOpen() {
}

func (b *BaseSession) OnClose() {
}

func (b *BaseSession) OnRead() {
	// r, w, n := b.onReadPacketBuffer.ReadWrite()
	// logger.Errorf("BaseSession.DoRead err OnRead, info:%v r:%v w:%v n:%v", b.Info(), r, w, n)
}

func (b *BaseSession) WriteByte(data []byte) error {
	// buf := bufpool.DefBufPool.Alloc()
	buf := register.DefFactory.GetTranslator().Pack(b, data)
	sb := bufpool.NewSlidingBufferWithData(buf, len(buf))

	rw := b.ModRWState(reactor.MASK_WRITE, true)
	b.Lock()
	b.writeBuffer.PushBack(sb)
	b.Unlock()
	// if false == b.WriteBuffer.PushSlidingBuffer(sb) {
	// 	return fmt.Errorf("session send string, circle sliding buffer overflow")
	// }
	b.reactor.ModHandler(rw, b)
	return nil
}

func (b *BaseSession) PopOnReadPacket() *bufpool.SlidingBuffer {
	var sb *bufpool.SlidingBuffer
	b.Lock()
	// e := b.onReadPacketBuffer.Front()
	// if nil != e {
	// 	sb = e.Value.(*bufpool.SlidingBuffer)
	// 	b.onReadPacketBuffer.Remove(e)
	// }
	sb = b.onReadPacketBuffer.PopFrontSlidingBuffer()
	b.Unlock()
	return sb
}
