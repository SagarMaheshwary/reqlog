package formatter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
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
		output:       config.OutputPretty,
	}

	out := f.Format(entry)

	// Ensure the search key is bold (contains ANSI code for bold)
	if !strings.Contains(out, "\033[1mrequest_id\033[0m") {
		t.Fatalf("expected search key 'request_id' to be bold")
	}
}

func TestFormat_OutputStructure(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 10, 0, 0, time.UTC)

	entry := domain.LogEntry{
		Timestamp: ts,
		Service:   "api",
		Message:   "test",
		Fields: map[string]any{
			"request_id": "req-456",
			"level":      "warn",
		},
	}

	entries := []domain.LogEntry{
		entry,
		{Service: "longer-service"},
	}

	f := NewFormatter(&Opts{
		Entries:    entries,
		SearchKeys: []string{"request_id"},
		Output:     config.OutputPretty,
	})
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

func TestFormat_OutputStructure_WithHost(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 10, 0, 0, time.UTC)

	entry := domain.LogEntry{
		Timestamp: ts,
		Service:   "api",
		Message:   "test",
		Host:      "srv",
		Fields: map[string]any{
			"request_id": "req-456",
			"level":      "warn",
		},
	}

	entries := []domain.LogEntry{
		entry,
		{Service: "longer-service"},
	}

	f := NewFormatter(&Opts{
		Entries:    entries,
		SearchKeys: []string{"request_id"},
		Output:     config.OutputPretty,
	})
	out := f.Format(entry)

	if !strings.Contains(out, ts.Format(tsFormat)) {
		t.Fatalf("expected timestamp in output")
	}

	if !strings.Contains(out, "[srv:api]") {
		t.Fatalf("expected service [srv:api] in output")
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

func TestFormatter_Format_ContextEntry(t *testing.T) {
	f := NewFormatter(&Opts{
		Output: config.OutputPretty,
	})

	entry := domain.LogEntry{
		Timestamp: mustParseRFC3339(t, "2024-03-10T12:00:00Z"),
		Service:   "auth",
		Message:   "request completed",
		IsContext: true,
		Fields: map[string]any{
			"level":      "info",
			"request_id": "abc123",
			"status":     "ok",
		},
	}

	got := f.Format(entry)

	// Level should be dimmed
	assertContains(
		t,
		strings.ToLower(got),
		"info",
	)
	// Message should be dimmed
	assertContains(
		t,
		got,
		dim+"request completed"+reset,
	)

	// Fields should be dimmed
	assertContains(
		t,
		got,
		dim+f.colorizer.Cyan("request_id"),
	)
	assertContains(
		t,
		got,
		"="+dim+"abc123",
	)
	assertContains(
		t,
		got,
		dim+f.colorizer.Cyan("status"),
	)
	assertContains(
		t,
		got,
		"="+dim+"ok",
	)
}

func TestFormatter_OutputJSON(t *testing.T) {
	tests := []struct {
		name    string
		entry   domain.LogEntry
		context int
		assert  func(t *testing.T, m map[string]any)
	}{
		{
			name: "basic json output",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339(t, "2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "login success",
				Fields: map[string]any{
					"user": "123",
				},
				IsContext: false,
				Host:      "srv",
			},
			context: 0,
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
				if m["host"] != "srv" {
					t.Fatalf("field host missing or wrong: %v", m["host"])
				}
			},
		},

		{
			name: "context flag is included",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339(t, "2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "login",
				IsContext: true,
			},
			context: 1,
			assert: func(t *testing.T, m map[string]any) {
				if m["context"] != true {
					t.Fatalf("expected context=true, got %v", m["context"])
				}
			},
		},

		{
			name: "reserved keys are prefixed into fields.*",
			entry: domain.LogEntry{
				Timestamp: mustParseRFC3339(t, "2024-03-10T12:00:00Z"),
				Service:   "auth",
				Message:   "test",
				Fields: map[string]any{
					"timestamp": "override-ts",
					"service":   "override-service",
					"message":   "override-message",
					"context":   "override-context",
				},
			},
			context: 1,
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
				Timestamp: mustParseRFC3339(t, "2024-03-10T12:00:00Z"),
				Service:   "svc",
				Message:   "ok",
				Fields: map[string]any{
					"a": 1,
					"b": true,
					"c": 1.5,
				},
			},
			context: 0,
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
			f := NewFormatter(&Opts{
				Output:  config.OutputJSON,
				Context: tt.context,
			})

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
	f := NewFormatter(&Opts{
		Output: config.OutputJSON,
	})

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

func TestFormatter_OutputJSONFields(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 10, 0, 0, time.UTC)

	tests := []struct {
		name   string
		fields []string
		entry  domain.LogEntry
		assert func(t *testing.T, m map[string]any)
	}{
		{
			name:   "full json output",
			fields: nil,
			entry: domain.LogEntry{
				Timestamp: ts,
				Service:   "api",
				Message:   "hello",
				Host:      "srv1",
				IsContext: true,
				Fields: map[string]any{
					"request_id": "req-123",
				},
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["timestamp"] != ts.Format(time.RFC3339Nano) {
					t.Fatalf("unexpected timestamp: %v", m["timestamp"])
				}

				if m["service"] != "api" {
					t.Fatalf("unexpected service: %v", m["service"])
				}

				if m["message"] != "hello" {
					t.Fatalf("unexpected message: %v", m["message"])
				}

				if m["host"] != "srv1" {
					t.Fatalf("unexpected host: %v", m["host"])
				}

				if m["request_id"] != "req-123" {
					t.Fatalf("unexpected request_id: %v", m["request_id"])
				}

				if m["context"] != true {
					t.Fatalf("unexpected context: %v", m["context"])
				}
			},
		},
		{
			name:   "selected fields output",
			fields: []string{"timestamp", "service", "request_id"},
			entry: domain.LogEntry{
				Timestamp: ts,
				Service:   "api",
				Message:   "ignored",
				Fields: map[string]any{
					"request_id": "req-123",
					"level":      "warn",
				},
			},
			assert: func(t *testing.T, m map[string]any) {
				if len(m) != 3 {
					t.Fatalf("expected 3 fields, got %d: %#v", len(m), m)
				}

				if m["timestamp"] != ts.Format(time.RFC3339Nano) {
					t.Fatalf("unexpected timestamp")
				}

				if m["service"] != "api" {
					t.Fatalf("unexpected service")
				}

				if m["request_id"] != "req-123" {
					t.Fatalf("unexpected request_id")
				}

				if _, ok := m["message"]; ok {
					t.Fatalf("message should not be included")
				}
			},
		},
		{
			name:   "fields prefix accesses conflicting field names",
			fields: []string{"timestamp", "fields.timestamp", "service", "fields.service", "message", "context", "fields.context", "host", "fields.host"},
			entry: domain.LogEntry{
				Timestamp: ts,
				Service:   "api",
				IsContext: true,
				Host:      "srv",
				Fields: map[string]any{
					"timestamp": "field-ts",
					"service":   "field-service",
					"context":   "field-context",
					"host":      "field-host",
				},
			},
			assert: func(t *testing.T, m map[string]any) {
				if m["timestamp"] != ts.Format(time.RFC3339Nano) {
					t.Fatalf("unexpected timestamp: %v", m["timestamp"])
				}

				if m["service"] != "api" {
					t.Fatalf("unexpected service: %v", m["service"])
				}

				if m["fields.timestamp"] != "field-ts" {
					t.Fatalf("unexpected fields.timestamp: %v", m["fields.timestamp"])
				}

				if m["fields.service"] != "field-service" {
					t.Fatalf("unexpected fields.service: %v", m["fields.service"])
				}

				if m["context"] != true {
					t.Fatalf("unexpected context: %v", m["context"])
				}

				if m["fields.context"] != "field-context" {
					t.Fatalf("unexpected fields.context: %v", m["fields.context"])
				}

				if m["host"] != "srv" {
					t.Fatalf("unexpected host: %v", m["host"])
				}

				if m["fields.host"] != "field-host" {
					t.Fatalf("unexpected fields.host: %v", m["fields.host"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(&Opts{
				Output:  config.OutputJSON,
				Fields:  tt.fields,
				Context: 1,
			})

			out := f.Format(tt.entry)

			var decoded map[string]any
			if err := json.Unmarshal([]byte(out), &decoded); err != nil {
				t.Fatalf("invalid json: %v\n%s", err, out)
			}

			tt.assert(t, decoded)
		})
	}
}

func TestFormatter_RenderPrettyFields(t *testing.T) {
	tests := []struct {
		name string

		entry  domain.LogEntry
		fields []string

		wantLevel       bool
		wantContains    []string
		wantNotContains []string
		wantOrder       []string
	}{
		{
			name: "sorted fields and level extraction",
			entry: domain.LogEntry{
				Fields: map[string]any{
					"request_id": "abc",
					"user":       "123",
					"level":      "info",
				},
			},
			wantLevel: true,
			wantContains: []string{
				"request_id",
				"user",
			},
			wantOrder: []string{
				"request_id",
				"user",
			},
		},
		{
			name: "respects fields order",
			entry: domain.LogEntry{
				Fields: map[string]any{
					"user":       "123",
					"request_id": "abc",
					"trace_id":   "xyz",
				},
			},
			fields: []string{
				"trace_id",
				"user",
			},
			wantContains: []string{
				"trace_id",
				"user",
			},
			wantNotContains: []string{
				"request_id",
			},
			wantOrder: []string{
				"trace_id",
				"user",
			},
		},
		{
			name: "level skipped from included fields",
			entry: domain.LogEntry{
				Fields: map[string]any{
					"level": "error",
					"user":  "123",
				},
			},
			fields: []string{
				"level",
				"user",
			},
			wantLevel: true,
			wantContains: []string{
				"user",
			},
			wantNotContains: []string{
				"level",
			},
		},
		{
			name: "missing requested fields ignored",
			entry: domain.LogEntry{
				Fields: map[string]any{
					"user": "123",
				},
			},
			fields: []string{
				"trace_id",
				"user",
			},
			wantContains: []string{
				"user",
			},
			wantNotContains: []string{
				"trace_id",
			},
		},
		{
			name: "diagnostics ignores fields filter",
			entry: domain.LogEntry{
				IsDiagnostics: true,
				Fields: map[string]any{
					"user":       "123",
					"request_id": "abc",
				},
			},
			fields: []string{
				"user",
			},
			wantContains: []string{
				"user",
				"request_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(&Opts{
				Fields: tt.fields,
			})

			level, out := f.renderPrettyFields(tt.entry)

			if tt.wantLevel && level == "" {
				t.Fatal("expected level output")
			}

			if !tt.wantLevel && level != "" {
				t.Fatalf("expected no level, got %q", level)
			}

			for _, s := range tt.wantContains {
				if !strings.Contains(out, s) {
					t.Fatalf("expected %q in output: %q", s, out)
				}
			}

			for _, s := range tt.wantNotContains {
				if strings.Contains(out, s) {
					t.Fatalf("did not expect %q in output: %q", s, out)
				}
			}

			for i := 0; i < len(tt.wantOrder)-1; i++ {
				left := strings.Index(out, tt.wantOrder[i])
				right := strings.Index(out, tt.wantOrder[i+1])

				if left == -1 || right == -1 {
					t.Fatalf("order check failed: %q", out)
				}

				if left > right {
					t.Fatalf(
						"expected %q before %q in %q",
						tt.wantOrder[i],
						tt.wantOrder[i+1],
						out,
					)
				}
			}
		})
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Fatalf(
			"expected output to contain %q, got %q",
			want,
			got,
		)
	}
}

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()

	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatal(err)
	}

	return ts
}
