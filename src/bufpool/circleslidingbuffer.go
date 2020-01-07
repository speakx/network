package bufpool

const (
	CirclePushOK = iota
	CirclePushOverflow
	CirclePushFull
)

type CircleSlidingBuffer struct {
	slidingBuffers []*SlidingBuffer
	curWritePos    int
	curReadPos     int
	count          int
}

func NewCircleSlidingBuffer(count int) *CircleSlidingBuffer {
	return &CircleSlidingBuffer{
		slidingBuffers: make([]*SlidingBuffer, count),
		curWritePos:    0, // 当前可写入位置
		curReadPos:     0, // 当前可读取位置
	}
}

// case A
//     curW
//       |
// -------------------------------------
// | -       - | -       - | -       - |
// -------------------------------------
//       |
//     curR
//
// case B
//                          curW
//                            |
// ------------------------------------
// | - Data1 - | - Data1 - | -      - |
// ------------------------------------
//      |
//    curR
//
// case C
//                                        curW
//                                          |
// -------------------------------------
// | - Data1 - | - Data1 - | - Data1 - |
// -------------------------------------
//      |
//    curR
//
// case D
//     curW
//       |
// -------------------------------------
// | -       - | - Data1 - | - Data1 - |
// -------------------------------------
//                  |
//                curR
//
// case E (D->E)
//                curW
//                  |
// -------------------------------------
// | - Data2 - | - Data1 - | - Data1 - |
// -------------------------------------
//                  |
//                curR
//
// case F
//     curW
//       |
// -------------------------------------
// | -       - | -       - | - Data1 - |
// -------------------------------------
//                               |
//                             curR
func (c *CircleSlidingBuffer) PushSlidingBuffer(sb *SlidingBuffer) int {
	// check case C
	if c.curWritePos >= len(c.slidingBuffers) {
		return CirclePushOverflow
	}

	// check case F
	if c.curWritePos == c.curReadPos && c.count > 0 {
		return CirclePushOverflow
	}

	c.slidingBuffers[c.curWritePos] = sb
	c.curWritePos++
	c.count++

	// check case D
	if c.curWritePos >= len(c.slidingBuffers) && c.curReadPos > 0 {
		c.curWritePos = 0
	}

	return CirclePushOK
}

func (c *CircleSlidingBuffer) PopFrontSlidingBuffer() *SlidingBuffer {
	// check case A
	if c.count == 0 {
		return nil
	}

	sb := c.slidingBuffers[c.curReadPos]
	c.curReadPos++
	c.count--

	if c.curReadPos >= len(c.slidingBuffers) {
		c.curReadPos = 0
	}

	if c.curWritePos >= len(c.slidingBuffers) && c.curReadPos > 0 {
		c.curWritePos = 0
	}
	return sb
}

func (c *CircleSlidingBuffer) FrontSlidingBuffer() *SlidingBuffer {
	if c.count == 0 {
		return nil
	}
	return c.slidingBuffers[c.curReadPos]
}

func (c *CircleSlidingBuffer) ReadWrite() (int, int, int) {
	return c.curReadPos, c.curWritePos, c.count
}
