package client

import (
	"fmt"
	"io"
	"net"
	"network/bufpool"
	"network/register"
	"sync"
	"time"
)

type connio struct {
	handler     ClientHandler
	conn        net.Conn
	readTag     int
	readTagSize int
	buf         []byte
	sync.Mutex
}

func (c *connio) write(p []byte) error {
	c.Lock()
	if nil != c.conn {
		buf := register.DefFactory.GetTranslator().Pack(c, p)
		_, err := c.conn.Write(buf)
		c.Unlock()
		return err
	}
	c.Unlock()
	return io.EOF
}

func (c *connio) connect() {
	go func() {
		conn, err := net.DialTimeout(c.handler.Network(), c.handler.Addr(), c.handler.Timeout())
		if nil != err {
			go func() {
				<-time.After(time.Second)
				c.connect()
			}()
			return
		}

		c.Lock()
		if nil != c.conn {
			c.conn.Close()
		}
		c.conn = conn
		switch c.handler.Network() {
		case "tcp":
			go func() { c.loopTCPRead(conn.(*net.TCPConn)) }()
		default:
			panic(fmt.Errorf("unsupport network:%v", c.handler.Network()))
		}
		c.Unlock()
	}()
}

func (c *connio) loopTCPRead(conn *net.TCPConn) {
	sb := bufpool.NewSlidingBuffer(0)
	for {
		n, err := conn.Read(sb.GetFreeData())
		if nil != err {
			c.connect()
			return
		}
		sb.AddWritePos(n)

		// unpack
		num := 0
		readBuf := sb.GetWrited(0)
		readBufLen := len(readBuf)

		for {
			unpacketN, packetData := register.DefFactory.GetTranslator().UnPack(c, readBuf)
			if 0 == unpacketN {
				break
			}

			if nil != packetData {
				num++
				c.handler.OnRead(packetData)
			}

			readBuf = readBuf[unpacketN:]
			readBufLen -= unpacketN
			if readBufLen <= 0 {
				break
			}
		}

		// 处理剩下的数据
		l := len(readBuf)
		if l == 0 {
			sb.Reset()
		} else if l > 0 {
			sb1 := bufpool.NewSlidingBuffer(0)
			sb1.Write(readBuf)

			sb.ReleaseSlidingBuffer()
			sb = sb1
		}
	}
}

func (c *connio) SetReadTag(tag, tagSize int) {
	c.readTag = tag
	c.readTagSize = tagSize
}

func (c *connio) GetReadTag() (int, int) {
	return c.readTag, c.readTagSize
}
