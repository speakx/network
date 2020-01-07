package bufpool

type SlidingBuffer struct {
	data     []byte // 数据缓冲区
	writePos int
	readPos  int
	release  bool
}

func NewSlidingBufferWithData(data []byte, n int) *SlidingBuffer {
	ret := &SlidingBuffer{
		data:     data,
		writePos: n,
		release:  false,
	}
	return ret
}

func NewSlidingBufferWithDataNoRelease(data []byte) *SlidingBuffer {
	ret := &SlidingBuffer{
		data:     data,
		writePos: len(data),
		release:  false,
	}
	return ret
}

func NewSlidingBuffer(preAlloc int) *SlidingBuffer {
	ret := &SlidingBuffer{
		data: DefBufPool.Alloc(),
		// data:    make([]byte, preAlloc),
		release: true,
	}
	return ret
}

func (s *SlidingBuffer) ReleaseSlidingBuffer() {
	if true == s.release {
		DefBufPool.Collect(s.data)
	}
}

func (s *SlidingBuffer) Reset() {
	s.writePos = 0
	s.readPos = 0
}

func (s *SlidingBuffer) GetData() []byte {
	return s.data
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

func (s *SlidingBuffer) GetWriteLen() int {
	return s.writePos
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
