package bufpool

type CircleSlidingBuffer struct {
	slidingBuffers []*SlidingBuffer
	curWritePos    int
	curReadPos     int
}

func NewCircleSlidingBuffer(count int) *CircleSlidingBuffer {
	return &CircleSlidingBuffer{
		slidingBuffers: make([]*SlidingBuffer, count),
		curWritePos:    0,
		curReadPos:     0,
	}
}

func (c *CircleSlidingBuffer) PushSlidingBuffer(sb *SlidingBuffer) bool {
	if c.curWritePos < c.curReadPos {
		return false
	}

	c.slidingBuffers[c.curWritePos] = sb
	c.curWritePos++

	if c.curWritePos >= len(c.slidingBuffers) {
		c.curWritePos = 0
	}
	return true
}

func (c *CircleSlidingBuffer) FrontSlidingBuffer() *SlidingBuffer {
	if c.curReadPos == c.curWritePos {
		return nil
	}
	return c.slidingBuffers[c.curReadPos]
}

func (c *CircleSlidingBuffer) RemoveFrontSlidingBuffer() {
	sb := c.slidingBuffers[c.curReadPos]
	sb.Release()
	c.curReadPos++
}
