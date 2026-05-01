package scanner

import (
	"container/heap"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func match(foundID, SearchValue string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(foundID, SearchValue)
	}
	return foundID == SearchValue
}

func passesSince(entry *domain.LogEntry, sinceTime time.Time) bool {
	if sinceTime.IsZero() {
		return true
	}
	return !entry.Timestamp.Before(sinceTime)
}

func parseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf(
		"invalid --since value %q\n\nExamples:\n  --since 10m\n  --since 1h\n  --since 2026-04-29T19:44:06Z\n  --since 2026-04-29",
		s,
	)
}

func asciiLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func containsFoldASCII(s, substr string) bool {
	n := len(substr)
	if n == 0 {
		return true
	}
	if n > len(s) {
		return false
	}

	first := asciiLower(substr[0])

	for i := 0; i <= len(s)-n; i++ {
		if asciiLower(s[i]) != first {
			continue
		}

		ok := true
		for j := 1; j < n; j++ {
			if asciiLower(s[i+j]) != asciiLower(substr[j]) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}

	return false
}

func logScanError(out io.Writer, path string, err error) {
	fmt.Fprintf(out, "error scanning %s: %v\n", path, err)
}

func drainHeap(h *entryHeap) []domain.LogEntry {
	out := make([]domain.LogEntry, 0, h.Len())
	for h.Len() > 0 {
		out = append(out, heap.Pop(h).(domain.LogEntry))
	}
	return out
}
