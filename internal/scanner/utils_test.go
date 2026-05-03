package scanner

import (
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func ptr(t time.Time) *time.Time {
	return &t
}

func mustParseTime(layout, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name       string
		found      string
		search     string
		ignoreCase bool
		expected   bool
	}{
		{"exact match", "abc", "abc", false, true},
		{"case mismatch", "ABC", "abc", false, false},
		{"ignore case match", "ABC", "abc", true, true},
		{"different values", "abc", "xyz", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := match(tt.found, tt.search, tt.ignoreCase)
			if result != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPassesSince(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		entryTime time.Time
		since     time.Time
		expected  bool
	}{
		{"since zero", now, time.Time{}, true},
		{"after since", now, now.Add(-1 * time.Minute), true},
		{"before since", now.Add(-10 * time.Minute), now.Add(-1 * time.Minute), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := domain.LogEntry{Timestamp: tt.entryTime}
			result := passesSince(&entry, tt.since)

			if result != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseSince(t *testing.T) {
	fixedNow := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  *time.Time
	}{
		{
			name:     "empty string",
			input:    "",
			expected: ptr(time.Time{}),
		},
		{
			name:      "invalid input",
			input:     "abc",
			expectErr: true,
		},
		{
			name:     "duration 5m",
			input:    "5m",
			expected: ptr(fixedNow.Add(-5 * time.Minute)),
		},
		{
			name:     "zero duration",
			input:    "0s",
			expected: ptr(fixedNow),
		},
		{
			name:  "RFC3339",
			input: "2026-04-29T19:44:06Z",
			expected: mustParseTime(
				time.RFC3339Nano,
				"2026-04-29T19:44:06Z",
			),
		},
		{
			name:  "date only",
			input: "2026-04-29",
			expected: mustParseTime(
				"2006-01-02",
				"2026-04-29",
			),
		},
		{
			name:     "unix seconds",
			input:    "1710943200",
			expected: ptr(time.Unix(1710943200, 0)),
		},
		{
			name:      "invalid unix length",
			input:     "171094320000",
			expectErr: true,
		},
		{
			name:      "invalid duration format",
			input:     "5minutes",
			expectErr: true,
		},
		{
			name:      "invalid date format",
			input:     "29-04-2026",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSince(tt.input, fixedNow)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "invalid --since") {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected != nil {
				if !got.Equal(*tt.expected) {
					t.Fatalf("expected %v, got %v", *tt.expected, got)
				}
			}
		})
	}
}

func TestAsciiLower(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected byte
	}{
		{"uppercase A", 'A', 'a'},
		{"uppercase Z", 'Z', 'z'},
		{"lowercase a", 'a', 'a'},
		{"lowercase z", 'z', 'z'},
		{"number", '5', '5'},
		{"symbol", '#', '#'},
		{"mixed char", 'M', 'm'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := asciiLower(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContainsFoldASCII(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "Hello World",
			substr:   "world",
			expected: true,
		},
		{
			name:     "case insensitive reverse",
			s:        "hello world",
			substr:   "WORLD",
			expected: true,
		},
		{
			name:     "no match",
			s:        "hello world",
			substr:   "abc",
			expected: false,
		},
		{
			name:     "substring at start",
			s:        "Hello",
			substr:   "he",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "Hello",
			substr:   "LO",
			expected: true,
		},
		{
			name:     "full string match",
			s:        "Hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "single character match",
			s:        "abc",
			substr:   "B",
			expected: true,
		},
		{
			name:     "single character no match",
			s:        "abc",
			substr:   "D",
			expected: false,
		},
		{
			name:     "repeated pattern",
			s:        "aaaAAA",
			substr:   "AaA",
			expected: true,
		},
		{
			name:     "special characters unchanged",
			s:        "hello-world",
			substr:   "WORLD",
			expected: true,
		},
		{
			name:     "numbers",
			s:        "abc123",
			substr:   "123",
			expected: true,
		},
		{
			name:     "partial overlap no match",
			s:        "abc",
			substr:   "ac",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsFoldASCII(tt.s, tt.substr)
			if result != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
