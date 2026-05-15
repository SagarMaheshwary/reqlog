package scanner

import (
	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type contextEngine struct {
	beforeBuf      *ringBuffer
	lineProcessor  *LineProcessor
	entryCollector *EntryCollector
	contextSize    int
	afterRemaining int
	stopAfterMatch bool
}

type ContextLine struct {
	Line    string
	Service string
	Entry   *domain.LogEntry
	IsMatch bool
}

func NewContextEngine(lp *LineProcessor, ec *EntryCollector, size int) *contextEngine {
	var beforeBuf *ringBuffer

	if size > 0 {
		beforeBuf = NewRingBuffer(size)
	}

	return &contextEngine{
		contextSize:    size,
		beforeBuf:      beforeBuf,
		lineProcessor:  lp,
		entryCollector: ec,
	}
}

func (c *contextEngine) Handle(in ContextLine) bool {
	if c.contextSize == 0 {
		if !in.IsMatch {
			return true
		}
		return c.entryCollector.Add(in.Entry)
	}

	if in.IsMatch && !c.stopAfterMatch {
		for _, e := range c.beforeBuf.Drain() {
			e.IsContext = true
			c.entryCollector.AddContext(&e)
		}

		continueReading := c.entryCollector.Add(in.Entry)
		if !continueReading {
			c.stopAfterMatch = true
		}

		c.afterRemaining = c.contextSize

		return true
	}

	entry, ok := c.lineProcessor.ParseOnly(in.Line, in.Service)
	if !ok {
		return true
	}

	entry.IsContext = true
	wasAfterContext := false

	if c.afterRemaining > 0 {
		c.entryCollector.AddContext(entry)

		c.afterRemaining--
		wasAfterContext = true
	}

	if c.beforeBuf != nil && !wasAfterContext {
		c.beforeBuf.Push(*entry)
	}

	if c.stopAfterMatch && c.afterRemaining == 0 {
		return false
	}

	return true
}
