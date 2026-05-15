package formatter

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestFormat_HighlightSearchKey(t *testing.T) {
	entry := domain.LogEntry{
		Timestamp: time.Now(),
		Service:   "api",
		Message:   "hello",
		Fields: map[string]any{
			"request_id": "abc123",
			"level":      "info",
		},
	}

	f := &Formatter{
		colorizer:    NewColorizer(),
		searchKeys:   []string{"request_id"},
		serviceWidth: len("api"),
		output:       OutputPretty,
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
		Message:   "fail",
		Fields: map[string]any{
			"level": "error",
		},
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

	f := NewFormatter(entries, []string{"request_id"}, OutputPretty)

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

func TestSortKVByPriority(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []kv
	}{
		{
			name: "prioritizes level then request_id",
			input: map[string]any{
				"extra":      "foo",
				"request_id": "abc",
				"level":      "warn",
			},
			expected: []kv{
				{key: "level", value: "warn"},
				{key: "request_id", value: "abc"},
				{key: "extra", value: "foo"},
			},
		},
		{
			name: "alphabetical fallback for equal priority",
			input: map[string]any{
				"zeta":  "1",
				"alpha": "2",
			},
			expected: []kv{
				{key: "alpha", value: "2"},
				{key: "zeta", value: "1"},
			},
		},
		{
			name: "mixed priority and alphabetical",
			input: map[string]any{
				"beta":       "b",
				"level":      "info",
				"request_id": "r1",
				"alpha":      "a",
			}, expected: []kv{
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

func TestFormatter_Format_ContextEntry(t *testing.T) {
	f := NewFormatter(nil, nil, OutputPretty)

	entry := domain.LogEntry{
		Timestamp: mustParseTime(t, "2024-03-10T12:00:00Z"),
		Service:   "auth",
		Message:   "user=123 status=ok",
		IsContext: true,
	}

	got := f.Format(entry)

	if !strings.Contains(got, dim) {
		t.Fatalf("expected formatted output to contain dim ANSI code, got %q", got)
	}

	if !strings.Contains(got, "user") {
		t.Fatalf("expected formatted output to contain message fields, got %q", got)
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()

	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}

	return ts
}

func TestFormatter_OutputJSON(t *testing.T) {
	tests := []struct {
		name   string
		entry  domain.LogEntry
		assert func(t *testing.T, m map[string]any)
	}{
		{
			name: "basic json output",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339("2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "login success",
				Fields: map[string]any{
					"user": "123",
				},
				IsContext: false,
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["service"] != "auth" {
					t.Fatalf("service mismatch: %v", m["service"])
				}
				if m["message"] != "login success" {
					t.Fatalf("message mismatch: %v", m["message"])
				}
				if m["user"] != "123" {
					t.Fatalf("field user missing or wrong: %v", m["user"])
				}
			},
		},

		{
			name: "context flag is included",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339("2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "login",
				IsContext: true,
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["context"] != true {
					t.Fatalf("expected context=true, got %v", m["context"])
				}
			},
		},

		{
			name: "reserved keys are prefixed into fields.*",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339("2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "test",
				Fields: map[string]any{
					"timestamp": "override-ts",
					"service":   "override-service",
					"message":   "override-message",
					"context":   "override-context",
				},
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["timestamp"] == "override-ts" {
					t.Fatalf("timestamp should NOT be overwritten")
				}
				if m["service"] == "override-service" {
					t.Fatalf("service should NOT be overwritten")
				}
				if m["message"] == "override-message" {
					t.Fatalf("message should NOT be overwritten")
				}
				if m["context"] == "override-context" {
					t.Fatalf("context should NOT be overwritten")
				}

				if m["fields.timestamp"] != "override-ts" {
					t.Fatalf("expected fields.timestamp, got %v", m["fields.timestamp"])
				}
			},
		},

		{
			name: "multiple fields preserved",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339("2024-03-10T12:00:00Z"),
				Service:   "svc",
				Message:   "ok",
				Fields: map[string]any{
					"a": 1,
					"b": true,
					"c": 1.5,
				},
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["a"] != float64(1) && m["a"] != 1 {
					t.Fatalf("unexpected a: %v", m["a"])
				}
				if m["b"] != true {
					t.Fatalf("unexpected b: %v", m["b"])
				}
				if m["c"] != 1.5 {
					t.Fatalf("unexpected c: %v", m["c"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(nil, nil, OutputJSON)

			out := f.Format(tt.entry)

			var decoded map[string]any
			if err := json.Unmarshal([]byte(out), &decoded); err != nil {
				t.Fatalf("invalid json output: %v\nraw=%s", err, out)
			}

			tt.assert(t, decoded)
		})
	}
}

func TestFormatter_OutputJSON_MarshalError(t *testing.T) {
	f := NewFormatter(nil, nil, OutputJSON)

	entry := domain.LogEntry{
		Raw: "raw-log-line",
		Fields: map[string]any{
			"bad": make(chan int), // json.Marshal unsupported type
		},
	}

	out := f.Format(entry)

	var decoded map[string]any

	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("expected valid fallback json, got error: %v", err)
	}

	if decoded["error"] != "failed to marshal log entry" {
		t.Fatalf(
			"expected marshal error message, got %v",
			decoded["error"],
		)
	}

	if decoded["raw"] != "raw-log-line" {
		t.Fatalf(
			"expected raw field %q, got %v",
			"raw-log-line",
			decoded["raw"],
		)
	}

	details, ok := decoded["details"].(string)
	if !ok || details == "" {
		t.Fatalf("expected non-empty details field, got %v", decoded["details"])
	}
}

func mustParseRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}
