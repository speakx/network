package bufpool

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

var DefBufPool *BufPool
var onceBufPool sync.Once

type BufPool struct {
	bufSize         int
	poolSize        int
	recycleDur      time.Duration
	allocator       chan []byte
	collector       chan []byte
	allocCounter    int
	collectCounter  int
	releaseCounter  int
	runtimePoolSize int
}

func InitBufPool(bufSize int, poolSize int, recycleDur time.Duration) {
	onceBufPool.Do(func() {
		DefBufPool = &BufPool{
			bufSize:    bufSize,
			poolSize:   poolSize,
			recycleDur: recycleDur,
			allocator:  make(chan []byte),
			collector:  make(chan []byte),
		}
		DefBufPool.loop()
	})
}

func (b *BufPool) Alloc() []byte {
	return <-b.allocator
}

func (b *BufPool) Collect(buf []byte) {
	b.collector <- buf
}

func (b *BufPool) String() string {
	return fmt.Sprintf("buf pool %v/%v alloc:%v collect:%v release:%v",
		b.runtimePoolSize, b.poolSize, b.allocCounter, b.collectCounter, b.releaseCounter)
}

func (b *BufPool) loop() {
	go func() {
		pool := list.New()
		for i := 0; i < b.poolSize; i++ {
			pool.PushBack(make([]byte, b.bufSize))
		}

		for {
			if pool.Len() <= 0 {
				pool.PushBack(make([]byte, b.bufSize))
				b.allocCounter++
			}
			e := pool.Front()

			select {
			case buf := <-b.collector:
				if pool.Len() < b.poolSize {
					pool.PushBack(buf)
					b.collectCounter++
				} else {
					b.releaseCounter++
				}
			case b.allocator <- e.Value.([]byte):
				pool.Remove(e)
			case <-time.After(b.recycleDur):
				if pool.Len() > b.poolSize {
					for i := 0; i < 10 && pool.Len() > b.poolSize; i++ {
						pool.Remove(pool.Back())
					}
				}
				b.runtimePoolSize = pool.Len()
			}
		}
	}()
}
