package formatter

import (
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestFormat(t *testing.T) {
	ts := time.Date(2026, 3, 21, 14, 5, 9, 0, time.UTC)
	entry := domain.LogEntry{
		Timestamp: ts,
		Service:   "auth-service",
		Message:   "request started",
	}

	c := NewColorizer()
	serviceColor := c.Color(entry.Service)

	got := Format(entry, c)

	want := dim + tsColor + "2026-03-21T14:05:09Z" + reset +
		" " +
		serviceColor + "[auth-service]" + reset +
		" | " +
		msgColor + "request started" + reset

	if got != want {
		t.Fatalf("unexpected formatted string:\nwant: %q\ngot:  %q", want, got)
	}
}
