package bufpool

import (
	"fmt"
	"sync"
	"time"
)

var DefBufPool *BufPool
var onceBufPool sync.Once

type BufPool struct {
	bufSize        int
	recycleDur     time.Duration
	allocCounter   int
	collectCounter int
	pool           sync.Pool
}

func InitBufPool(bufSize int, poolSize int, recycleDur time.Duration) {
	onceBufPool.Do(func() {
		DefBufPool = &BufPool{
			bufSize:    bufSize,
			recycleDur: recycleDur,
			pool:       sync.Pool{},
		}
		DefBufPool.loop()
	})
}

func (b *BufPool) Alloc() []byte {
	buf := b.pool.Get()
	if nil == buf {
		b.allocCounter++
		return make([]byte, b.bufSize)
	}
	return buf.([]byte)
}

func (b *BufPool) Collect(buf []byte) {
	b.collectCounter++
	b.pool.Put(buf)
}

func (b *BufPool) loop() {
	go func() {
		for {
			<-time.After(b.recycleDur)
			fmt.Printf("pool alloc:%v collect:%v\n", b.allocCounter, b.collectCounter)
		}
	}()
}
