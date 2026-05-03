package scanner

import (
	"strconv"
	"sync"
	"time"
)

type TimeParser interface {
	Parse(s string, source string) (time.Time, bool)
}

type TimestampParser func(string) (time.Time, bool)

type timeParser struct {
	mu      sync.RWMutex
	parsers map[string]TimestampParser
}

func NewTimeParser() TimeParser {
	return &timeParser{
		mu:      sync.RWMutex{},
		parsers: make(map[string]TimestampParser),
	}
}

func (t *timeParser) Parse(s string, source string) (time.Time, bool) {
	t.mu.RLock()
	if parser, ok := t.parsers[source]; ok {
		t.mu.RUnlock()
		return parser(s)
	}
	t.mu.RUnlock()

	parsers := []TimestampParser{
		parseRFC3339,
		parseUnix,
	}
	for _, parser := range parsers {
		if ts, ok := parser(s); ok {
			t.mu.Lock()
			t.parsers[source] = parser
			t.mu.Unlock()
			return ts, true
		}
	}

	return time.Time{}, false
}

func parseRFC3339(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

func parseUnix(s string) (time.Time, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	switch len(s) {
	case 10: // seconds
		return time.Unix(n, 0), true
	case 13: // milliseconds
		return time.UnixMilli(n), true
	case 16: // microseconds
		return time.UnixMicro(n), true
	case 19: // nanoseconds
		return time.Unix(0, n), true
	default:
		return time.Time{}, false
	}
}
