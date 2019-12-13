package client

import (
	"network/bufpool"
	"time"
)

type ClientHandler interface {
	OnRead(sb *bufpool.SlidingBuffer)
	Network() string
	Addr() string
	Timeout() time.Duration
}

type BaseClient struct {
	network string
	addr    string
	timeout time.Duration
	cio     *connio
}

type BaseClientHandler interface {
	InitClient(network, addr string, timeout time.Duration, c ClientHandler)
}

func TCPClientConnect(addr string, timeout time.Duration, c ClientHandler) {
	bufpool.InitBufPool(1024*1024, 512, 1)
	c.(BaseClientHandler).InitClient("tcp", addr, timeout, c)
}

func (b *BaseClient) Network() string {
	return b.network
}

func (b *BaseClient) Addr() string {
	return b.addr
}

func (b *BaseClient) Timeout() time.Duration {
	return b.timeout
}

func (b *BaseClient) InitClient(network, addr string, timeout time.Duration, c ClientHandler) {
	b.network = "tcp"
	b.addr = addr
	b.timeout = timeout
	b.cio = &connio{handler: c}
	b.cio.connect()
}

func (b *BaseClient) OnRead(sb *bufpool.SlidingBuffer) {
}
