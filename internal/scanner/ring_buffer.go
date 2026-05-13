package scanner

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type ringBuffer struct {
	buf  []domain.LogEntry
	head int
	size int
	cap  int
}

func NewRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		buf: make([]domain.LogEntry, capacity),
		cap: capacity,
	}
}

func (r *ringBuffer) Push(line domain.LogEntry) {
	if r.cap == 0 {
		return
	}
	if r.size < r.cap {
		// buf[head..head+size) are valid; write at head+size
		r.buf[(r.head+r.size)%r.cap] = line
		r.size++
	} else {
		// full: overwrite oldest slot, advance head
		r.buf[r.head] = line
		r.head = (r.head + 1) % r.cap
	}
}

func (r *ringBuffer) Drain() []domain.LogEntry {
	if r.size == 0 {
		return nil
	}
	out := make([]domain.LogEntry, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.head+i)%r.cap]
	}
	r.head = 0
	r.size = 0
	return out
}
