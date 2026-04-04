package scanner

import (
	"fmt"
	"os"
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

func parseSince(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}
	}

	return time.Now().Add(-d)
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

func logFileScanError(path string, err error) {
	fmt.Fprintf(os.Stderr, "error scanning %s: %v\n", path, err)
}
