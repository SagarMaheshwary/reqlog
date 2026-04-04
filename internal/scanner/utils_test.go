package scanner

import (
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

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
	tests := []struct {
		name   string
		input  string
		isZero bool
	}{
		{"empty string", "", true},
		{"invalid duration", "abc", true},
		{"valid duration", "5m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSince(tt.input)

			if tt.isZero && !result.IsZero() {
				t.Fatal("expected zero time")
			}

			if !tt.isZero && result.IsZero() {
				t.Fatal("expected non-zero time")
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
