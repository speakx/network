package idmaker

import "sync/atomic"

var seq int32 = 0

func MakeSidSafeLock() int32 {
	return atomic.AddInt32(&seq, 1)
}
