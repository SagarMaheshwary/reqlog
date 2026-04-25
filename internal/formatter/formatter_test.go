package formatter

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestParseMessage_RemovesMessageKey(t *testing.T) {
	msg := "level=info message=Hello request_id=abc123 extra=xyz"
	main, pairs := parseMessage(msg)

	if main != "Hello" {
		t.Fatalf("expected main message 'Hello', got %q", main)
	}

	for _, p := range pairs {
		if p.key == "message" || p.key == "msg" {
			t.Fatalf("message key should be removed from pairs")
		}
	}

	if len(pairs) != 3 { // level, request_id, extra
		t.Fatalf("expected 3 kv pairs, got %d", len(pairs))
	}
}

func TestParseMessage_SortsByPriority(t *testing.T) {
	msg := "extra=foo request_id=abc123 level=warn alpha=bar"
	_, pairs := parseMessage(msg)

	if pairs[0].key != "level" {
		t.Fatalf("expected first key 'level', got %q", pairs[0].key)
	}
	if pairs[1].key != "request_id" {
		t.Fatalf("expected second key 'request_id', got %q", pairs[1].key)
	}
	// rest are sorted alphabetically
	if pairs[2].key != "alpha" || pairs[3].key != "extra" {
		t.Fatalf("expected remaining keys sorted alphabetically, got %v", pairs[2:])
	}
}

func TestFormat_HighlightSearchKey(t *testing.T) {
	entry := domain.LogEntry{
		Timestamp: time.Now(),
		Service:   "api",
		Message:   "request_id=abc123 level=info message=hello",
	}

	f := &Formatter{
		colorizer:    NewColorizer(),
		searchKeys:   []string{"request_id"},
		serviceWidth: len("api"),
	}

	out := f.Format(entry)

	// Ensure the search key is bold (contains ANSI code for bold)
	if !strings.Contains(out, "\033[1mrequest_id\033[0m") {
		t.Fatalf("expected search key 'request_id' to be bold")
	}
}

func TestFormat_ColorLevel(t *testing.T) {
	entry := domain.LogEntry{
		Timestamp: time.Now(),
		Service:   "api",
		Message:   "level=error message=fail",
	}

	f := &Formatter{
		colorizer:    NewColorizer(),
		searchKeys:   nil,
		serviceWidth: len("api"),
	}

	out := f.Format(entry)

	if !strings.Contains(out, f.colorizer.Red("error")) {
		t.Fatalf("expected 'error' to be colored red")
	}
}

func TestFormat_OutputStructure(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 10, 0, 0, time.UTC)

	entry := domain.LogEntry{
		Timestamp: ts,
		Service:   "api",
		Message:   "level=info message=test request_id=xyz",
	}

	entries := []domain.LogEntry{
		entry,
		{Service: "longer-service"},
	}

	f := NewFormatter(entries, []string{"request_id"})

	out := f.Format(entry)

	if !strings.Contains(out, ts.Format(tsFormat)) {
		t.Fatalf("expected timestamp in output")
	}

	if !strings.Contains(out, "[api]") {
		t.Fatalf("expected service [api] in output")
	}

	if !strings.Contains(out, "test") {
		t.Fatalf("expected main message 'test' in output")
	}

	if !strings.Contains(out, " | ") {
		t.Fatalf("expected ' | ' separator")
	}

	// Ensure key/value parts include request_id
	if !strings.Contains(out, "request_id") {
		t.Fatalf("expected 'request_id' in output")
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantMainMsg  string
		wantKVKeys   []string
		wantKVValues []string
	}{
		{
			name:         "message key removed and main msg extracted",
			input:        "level=info message=Hello request_id=abc123 extra=xyz",
			wantMainMsg:  "Hello",
			wantKVKeys:   []string{"level", "request_id", "extra"},
			wantKVValues: []string{"info", "abc123", "xyz"},
		},
		{
			name:         "no message key, text treated as main message",
			input:        "just some log text level=warn",
			wantMainMsg:  "just some log text",
			wantKVKeys:   []string{"level"},
			wantKVValues: []string{"warn"},
		},
		{
			name:         "message key with spaces preserved",
			input:        "message=Hello world level=info request_id=xyz",
			wantMainMsg:  "Hello world",
			wantKVKeys:   []string{"level", "request_id"},
			wantKVValues: []string{"info", "xyz"},
		},
		{
			name:         "multiline value handled",
			input:        "time_taken=13ms message=hit level=info",
			wantMainMsg:  "hit",
			wantKVKeys:   []string{"level", "time_taken"},
			wantKVValues: []string{"info", "13ms"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainMsg, kvs := parseMessage(tt.input)

			if mainMsg != tt.wantMainMsg {
				t.Errorf("mainMsg = %q; want %q", mainMsg, tt.wantMainMsg)
			}

			if len(kvs) != len(tt.wantKVKeys) {
				t.Fatalf("got %d kv pairs, want %d", len(kvs), len(tt.wantKVKeys))
			}

			for i, kv := range kvs {
				if kv.key != tt.wantKVKeys[i] || kv.value != tt.wantKVValues[i] {
					t.Errorf("kv[%d] = {%q, %q}; want {%q, %q}", i, kv.key, kv.value, tt.wantKVKeys[i], tt.wantKVValues[i])
				}
			}
		})
	}
}

func TestSortKVByPriority(t *testing.T) {
	tests := []struct {
		name     string
		input    []kv
		expected []kv
	}{
		{
			name: "prioritizes level then request_id",
			input: []kv{
				{key: "extra", value: "foo"},
				{key: "request_id", value: "abc"},
				{key: "level", value: "warn"},
			},
			expected: []kv{
				{key: "level", value: "warn"},
				{key: "request_id", value: "abc"},
				{key: "extra", value: "foo"},
			},
		},
		{
			name: "alphabetical fallback for equal priority",
			input: []kv{
				{key: "zeta", value: "1"},
				{key: "alpha", value: "2"},
			},
			expected: []kv{
				{key: "alpha", value: "2"},
				{key: "zeta", value: "1"},
			},
		},
		{
			name: "mixed priority and alphabetical",
			input: []kv{
				{key: "beta", value: "b"},
				{key: "level", value: "info"},
				{key: "request_id", value: "r1"},
				{key: "alpha", value: "a"},
			},
			expected: []kv{
				{key: "level", value: "info"},
				{key: "request_id", value: "r1"},
				{key: "alpha", value: "a"},
				{key: "beta", value: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortKVByPriority(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v; want %v", got, tt.expected)
			}
		})
	}
}
