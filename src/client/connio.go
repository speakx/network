package client

import (
	"fmt"
	"io"
	"net"
	"network/bufpool"
	"sync"
	"time"
)

type connio struct {
	handler ClientHandler
	conn    net.Conn
	readBuf *bufpool.SlidingBuffer
	sync.Mutex
}

func (c *connio) write(p []byte) (n int, err error) {
	c.Lock()
	if nil != c.conn {
		n, err = c.conn.Write(p)
	} else {
		err = io.EOF
	}
	c.Unlock()
	return n, err
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
	for {
		buf := bufpool.NewSlidingBuffer(0)
		n, err := conn.Read(buf.GetFreeData())
		if nil != err {
			c.connect()
			return
		}
		buf.AddWritePos(n)

		if nil == c.readBuf {
			c.readBuf = buf
			c.handler.OnRead(c.readBuf)
			c.readBuf.ReleaseSlidingBuffer()
			c.readBuf = nil
		} else {
			n1 := c.readBuf.Write(buf.GetWrited(0))
			c.handler.OnRead(c.readBuf)
			c.readBuf.ReleaseSlidingBuffer()
			c.readBuf = nil

			if len(buf.GetWrited(n1)) > 0 {
				c.readBuf = bufpool.NewSlidingBuffer(0)
				c.readBuf.Write(buf.GetWrited(0))
			}
		}
	}
}
