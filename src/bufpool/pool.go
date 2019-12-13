package bufpool

import (
	"fmt"
	"sync"
	"time"
)

var DefBufPool *BufPool
var onceBufPool sync.Once
var allocCounter int
var collectCounter int

type BufPool struct {
	recycleDur time.Duration
	pool       sync.Pool
}

func InitBufPool(bufSize int, poolSize int, recycleDur time.Duration) {
	onceBufPool.Do(func() {
		DefBufPool = &BufPool{
			recycleDur: recycleDur,
			pool: sync.Pool{New: func() interface{} {
				allocCounter++
				return make([]byte, bufSize)
			}},
		}
		DefBufPool.loop()
	})
}

func (b *BufPool) Alloc() []byte {
	return b.pool.Get().([]byte)
}

func (b *BufPool) Collect(buf []byte) {
	collectCounter++
	b.pool.Put(buf)
}

func (b *BufPool) loop() {
	go func() {
		for {
			<-time.After(b.recycleDur)
			fmt.Printf("pool alloc:%v collect:%v\n", allocCounter, collectCounter)
		}
	}()
}
