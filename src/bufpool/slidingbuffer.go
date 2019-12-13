package bufpool

type SlidingBuffer struct {
	data     []byte // 数据缓冲区
	writePos int
	readPos  int
}

func NewSlidingBufferWithData(data []byte, n int) *SlidingBuffer {
	ret := &SlidingBuffer{
		data:     data,
		writePos: n,
	}
	return ret
}

func NewSlidingBuffer(preAlloc int) *SlidingBuffer {
	ret := &SlidingBuffer{
		data: DefBufPool.Alloc(),
	}
	return ret
}

func (s *SlidingBuffer) ReleaseSlidingBuffer() {
	DefBufPool.Collect(s.data)
	// s.data = nil
	// s.writePos = 0
	// s.readPos = 0
}

func (s *SlidingBuffer) Reset() {
	s.writePos = 0
	s.readPos = 0
}

func (s *SlidingBuffer) WriteLen() int {
	return s.writePos
}

func (s *SlidingBuffer) Write(data []byte) int {
	n := len(data)
	if n > (len(s.data) - s.writePos) {
		n = len(s.data) - s.writePos
	}

	copy(s.data[s.writePos:], data[:n])
	s.writePos += n
	return n
}

func (s *SlidingBuffer) GetWrited(n int) []byte {
	if n != 0 {
		s.readPos += n
		return s.data[s.readPos:s.writePos]
	}
	return s.data[s.readPos:s.writePos]
}

func (s *SlidingBuffer) GetFreeData() []byte {
	return s.data[s.writePos:]
}

func (s *SlidingBuffer) AddWritePos(n int) {
	s.writePos += n
}
