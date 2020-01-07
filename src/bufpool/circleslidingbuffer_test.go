package bufpool

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestCircleslidingBufferWriteOverFlow(t *testing.T) {
	count := int64(3)
	csb := NewCircleSlidingBuffer(int(count))

	i := int64(1)
	for index := 0; index < 15; index++ {
		s := strconv.FormatInt(i, 10)
		buf := []byte(s)
		sb := NewSlidingBufferWithData(buf, len(buf))
		ret := csb.PushSlidingBuffer(sb)
		r, w, _ := csb.ReadWrite()
		t.Logf("write:%v ret:%v r:%v w:%v", i, ret, r, w)

		if CirclePushOverflow == ret && i != (count+1) {
			t.Errorf("write overflow failed.")
		}

		if CirclePushOK == ret {
			i++
		} else {
			break
		}
	}

	t.Logf("write:%v ok", i)
}

func TestCircleslidingBufferReadOverFlow(t *testing.T) {
	count := int64(3)
	csb := NewCircleSlidingBuffer(int(count))

	for index := int64(1); index <= 2; index++ {
		s := strconv.FormatInt(index, 10)
		buf := []byte(s)
		sb := NewSlidingBufferWithData(buf, len(buf))
		ret := csb.PushSlidingBuffer(sb)
		r, w, _ := csb.ReadWrite()
		t.Logf("write:%v ret:%v r:%v w:%v", index, ret, r, w)
	}

	for index := 0; index < 15; index++ {
		sb := csb.PopFrontSlidingBuffer()

		if nil == sb && index != 2 {
			t.Errorf("read overflow failed. i:%v", index)
		}

		if nil == sb {
			break
		}

		buf := sb.GetWrited(0)
		i, _ := strconv.ParseInt(string(buf), 10, 64)
		r, w, _ := csb.ReadWrite()
		t.Logf("read:%v r:%v w:%v", i, r, w)
	}
}

func TestCircleslidingBufferReadWrite(t *testing.T) {
	count := int64(3)
	csb := NewCircleSlidingBuffer(int(count))

	i := int64(0)
	for index := 0; index < 3; index++ {
		s := strconv.FormatInt(int64(i), 10)
		i++
		buf := []byte(s)
		ret := csb.PushSlidingBuffer(NewSlidingBufferWithData(buf, len(buf)))
		r, w, c := csb.ReadWrite()
		t.Logf("write ret:%v v:%v r:%v w:%v c:%v", ret, s, r, w, c)
	}

	// 写Overflow
	ret := csb.PushSlidingBuffer(NewSlidingBufferWithData(nil, 0))
	r, w, c := csb.ReadWrite()
	t.Logf("write ret:%v v:%v r:%v w:%v c:%v", ret, 0, r, w, c)
	if ret != CirclePushOverflow {
		t.Errorf("write failed, must be overflow. r:%v w:%v c:%v", r, w, c)
		return
	}

	// 读一个
	sb := csb.PopFrontSlidingBuffer()
	v, _ := strconv.ParseInt(string(sb.GetWrited(0)), 10, 64)
	if v != 0 {
		t.Errorf("read failed, must be value:0. v:%v r:%v w:%v c:%v", v, r, w, c)
		return
	}
	r, w, c = csb.ReadWrite()
	t.Logf("read v:%v r:%v w:%v c:%v", v, r, w, c)

	// 写一个
	s := strconv.FormatInt(int64(i), 10)
	i++
	buf := []byte(s)
	ret = csb.PushSlidingBuffer(NewSlidingBufferWithData(buf, len(buf)))
	r, w, c = csb.ReadWrite()
	t.Logf("write ret:%v v:%v r:%v w:%v c:%v", ret, s, r, w, c)
	if ret == CirclePushOverflow {
		t.Errorf("write overflow. r:%v w:%v c:%v", r, w, c)
		return
	}

	// 写Overflow
	ret = csb.PushSlidingBuffer(NewSlidingBufferWithData(nil, 0))
	r, w, c = csb.ReadWrite()
	t.Logf("write ret:%v v:%v r:%v w:%v c:%v", ret, 0, r, w, c)
	if ret != CirclePushOverflow {
		t.Errorf("write failed, must be overflow. r:%v w:%v c:%v", r, w, c)
		return
	}
}

func TestCircleslidingBuffer(t *testing.T) {
	csb := NewCircleSlidingBuffer(3)
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		cur := int64(-1)
		max := int64(0xFFFF)
		for {
			lock.Lock()
			sb := csb.PopFrontSlidingBuffer()
			if nil == sb {
				r, w, c := csb.ReadWrite()
				t.Logf("read v:nil r:%v w:%v c:%v", r, w, c)
				lock.Unlock()
				continue
			}

			buf := sb.GetWrited(0)
			i, _ := strconv.ParseInt(string(buf), 10, 64)
			r, w, c := csb.ReadWrite()
			// t.Logf("read v:%v r:%v w:%v c:%v", i, r, w, c)

			if cur+1 != i {
				t.Errorf("error cur:%v i:%v r:%v w:%v c:%v", cur, i, r, w, c)
				lock.Unlock()
				break
			}
			if c < 0 {
				t.Errorf("error cur:%v i:%v r:%v w:%v c:%v", cur, i, r, w, c)
				lock.Unlock()
				break
			}
			cur = i

			if cur%1000 == 0 {
				fmt.Printf("read v:%v r:%v w:%v c:%v\n", i, r, w, c)
			}
			lock.Unlock()

			if cur > max {
				break
			}
		}
		wg.Done()
	}()

	go func() {
		i := int64(0)
		for {
			s := strconv.FormatInt(i, 10)
			buf := []byte(s)
			lock.Lock()
			sb := NewSlidingBufferWithData(buf, len(buf))
			if CirclePushOK == csb.PushSlidingBuffer(sb) {
				// fmt.Printf("write:%v\n", i)
				i++
			} else {
				// fmt.Printf("push:%v overflow\n", i)
			}
			lock.Unlock()
		}
	}()

	wg.Wait()
}
