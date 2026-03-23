package formatter

import (
	"bytes"
	"io"
	"os"
	"strings"
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

	want := dim + tsColor + "[14:05:09]" + reset +
		" " +
		serviceColor + "auth-service" + reset +
		" | " +
		msgColor + "request started" + reset

	if got != want {
		t.Fatalf("unexpected formatted string:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestPrint_PrintsAllEntriesWhenLimitIsZero(t *testing.T) {
	entries := []domain.LogEntry{
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			Service:   "auth-service",
			Message:   "started",
		},
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 1, 0, time.UTC),
			Service:   "payment-service",
			Message:   "processed",
		},
	}

	output := captureStdout(t, func() {
		Print(entries, 0)
	})

	lines := splitNonEmptyLines(output)
	if len(lines) != 2 {
		t.Fatalf("expected 2 printed lines, got %d\noutput:\n%s", len(lines), output)
	}
}

func TestPrint_RespectsLimit(t *testing.T) {
	entries := []domain.LogEntry{
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			Service:   "auth-service",
			Message:   "started",
		},
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 1, 0, time.UTC),
			Service:   "payment-service",
			Message:   "processed",
		},
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 2, 0, time.UTC),
			Service:   "order-service",
			Message:   "finished",
		},
	}

	output := captureStdout(t, func() {
		Print(entries, 2)
	})

	lines := splitNonEmptyLines(output)
	if len(lines) != 2 {
		t.Fatalf("expected 2 printed lines due to limit, got %d\noutput:\n%s", len(lines), output)
	}
}

func TestPrint_PrintsFormattedEntries(t *testing.T) {
	entries := []domain.LogEntry{
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			Service:   "auth-service",
			Message:   "started",
		},
	}

	output := captureStdout(t, func() {
		Print(entries, 1)
	})

	expected := Format(entries[0], NewColorizer()) + "\n"
	if output != expected {
		t.Fatalf("unexpected output:\nwant: %q\ngot:  %q", expected, output)
	}
}

func TestPrint_WithNegativeLimit_PrintsAllEntries(t *testing.T) {
	entries := []domain.LogEntry{
		{Timestamp: time.Now(), Service: "a", Message: "m1"},
		{Timestamp: time.Now(), Service: "b", Message: "m2"},
	}

	output := captureStdout(t, func() {
		Print(entries, -1)
	})

	lines := splitNonEmptyLines(output)
	if len(lines) != 2 {
		t.Fatalf("expected 2 printed lines, got %d", len(lines))
	}
}

func TestPrint_WithEmptyEntries_PrintsNothing(t *testing.T) {
	output := captureStdout(t, func() {
		Print(nil, 10)
	})

	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}

	return buf.String()
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	lines := make([]string, 0, len(raw))

	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
