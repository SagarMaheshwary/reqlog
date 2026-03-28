package formatter

import (
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestNewFormatter_ServiceWidth(t *testing.T) {
	entries := []domain.LogEntry{
		{Service: "api"},
		{Service: "order-service"},
		{Service: "inv"},
	}

	f := NewFormatter(entries)

	expected := len("order-service")
	if f.serviceWidth != expected {
		t.Fatalf("expected serviceWidth %d, got %d", expected, f.serviceWidth)
	}
}

func TestPadAfter(t *testing.T) {
	f := &Formatter{serviceWidth: 10}

	tests := []struct {
		service  string
		expected int // expected number of spaces
	}{
		{"api", 7},
		{"service", 3},
		{"verylongsvc", 0}, // longer than width → no padding
	}

	for _, tt := range tests {
		padding := f.padAfter(tt.service)

		if len(padding) != tt.expected {
			t.Fatalf("service=%s expected padding %d, got %d",
				tt.service, tt.expected, len(padding))
		}
	}
}

func TestFormat_OutputStructure(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 10, 0, 0, time.UTC)

	entry := domain.LogEntry{
		Timestamp: ts,
		Service:   "api",
		Message:   "test message",
	}

	entries := []domain.LogEntry{
		entry,
		{Service: "longer-service"},
	}

	f := NewFormatter(entries)

	out := f.Format(entry)

	if !strings.Contains(out, ts.Format(time.RFC3339)) {
		t.Fatalf("expected timestamp in output")
	}

	if !strings.Contains(out, "[api]") {
		t.Fatalf("expected service [api] in output")
	}

	if !strings.Contains(out, "test message") {
		t.Fatalf("expected message in output")
	}

	if !strings.Contains(out, " | ") {
		t.Fatalf("expected ' | ' separator in output")
	}
}

func TestFormat_Alignment(t *testing.T) {
	ts := time.Now()

	entries := []domain.LogEntry{
		{Timestamp: ts, Service: "api", Message: "one"},
		{Timestamp: ts, Service: "longer-service", Message: "two"},
	}

	f := NewFormatter(entries)

	out1 := f.Format(entries[0])
	out2 := f.Format(entries[1])

	i1 := strings.Index(out1, "|")
	i2 := strings.Index(out2, "|")

	if i1 != i2 {
		t.Fatalf("expected aligned pipes, got %d and %d", i1, i2)
	}
}

func TestPadAfter_NoPaddingNeeded(t *testing.T) {
	f := &Formatter{serviceWidth: 3}

	padding := f.padAfter("abcd")

	if padding != "" {
		t.Fatalf("expected no padding, got %q", padding)
	}
}
