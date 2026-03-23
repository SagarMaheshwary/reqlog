package parser

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestJSONParser_ExtractField(t *testing.T) {
	p := JSONParser{}

	tests := []struct {
		name     string
		line     string
		keys     []string
		expected string
		found    bool
	}{
		{
			name:     "single key match",
			line:     `{"status":"ok","user":"123"}`,
			keys:     []string{"status"},
			expected: "ok",
			found:    true,
		},
		{
			name:     "multiple keys first match wins",
			line:     `{"user":"123","status":"ok"}`,
			keys:     []string{"status", "user"},
			expected: "ok", // key order matters, not JSON order
			found:    true,
		},
		{
			name:     "number value",
			line:     `{"status":200}`,
			keys:     []string{"status"},
			expected: "200",
			found:    true,
		},
		{
			name:     "nested key",
			line:     `{"req":{"id":"abc"}}`,
			keys:     []string{"req.id"},
			expected: "abc",
			found:    true,
		},
		{
			name:     "key not found",
			line:     `{"foo":"bar"}`,
			keys:     []string{"status"},
			expected: "",
			found:    false,
		},
		{
			name:     "invalid json",
			line:     `invalid-json`,
			keys:     []string{"status"},
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := p.ExtractField(tt.line, tt.keys)

			if ok != tt.found {
				t.Fatalf("expected found=%v, got %v", tt.found, ok)
			}

			if val != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, val)
			}
		})
	}
}

func TestJSONParser_Parse(t *testing.T) {
	p := JSONParser{}

	tests := []struct {
		name        string
		line        string
		service     string
		expectError bool
	}{
		{
			name:        "valid with time",
			line:        `{"time":"2024-03-10T12:00:00Z","status":"ok"}`,
			service:     "svc",
			expectError: false,
		},
		{
			name:        "valid with timestamp",
			line:        `{"timestamp":"2024-03-10T12:00:00Z","status":"ok"}`,
			service:     "svc",
			expectError: false,
		},
		{
			name:        "valid with ts",
			line:        `{"ts":"2024-03-10T12:00:00Z","status":"ok"}`,
			service:     "svc",
			expectError: false,
		},
		{
			name:        "missing timestamp",
			line:        `{"status":"ok"}`,
			service:     "svc",
			expectError: true,
		},
		{
			name:        "invalid timestamp format",
			line:        `{"time":"not-a-time"}`,
			service:     "svc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line, tt.service)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if entry.Service != tt.service {
				t.Errorf("expected service %s, got %s", tt.service, entry.Service)
			}

			if entry.Timestamp.IsZero() {
				t.Error("expected valid timestamp")
			}
		})
	}
}

func TestExtractJSONTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "time key",
			line:     `{"time":"2024-03-10T12:00:00Z"}`,
			expected: true,
		},
		{
			name:     "timestamp key",
			line:     `{"timestamp":"2024-03-10T12:00:00Z"}`,
			expected: true,
		},
		{
			name:     "ts key",
			line:     `{"ts":"2024-03-10T12:00:00Z"}`,
			expected: true,
		},
		{
			name:     "invalid format",
			line:     `{"time":"invalid"}`,
			expected: false,
		},
		{
			name:     "missing key",
			line:     `{"foo":"bar"}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, ok := extractJSONTimestamp(tt.line)

			if ok != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, ok)
			}

			if ok && ts.IsZero() {
				t.Fatal("expected valid timestamp")
			}
		})
	}
}

func TestBuildJSONMessage(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "simple fields sorted",
			line:     `{"time":"2024-03-10T12:00:00Z","b":2,"a":1}`,
			expected: "a=1 b=2",
		},
		{
			name:     "string and number",
			line:     `{"time":"2024-03-10T12:00:00Z","status":"ok","code":200}`,
			expected: "code=200 status=ok",
		},
		{
			name:     "ignores timestamp fields",
			line:     `{"time":"2024-03-10T12:00:00Z","timestamp":"x","a":1}`,
			expected: "a=1",
		},
		{
			name:     "non-object json returns raw",
			line:     `"just a string"`,
			expected: `"just a string"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildJSONMessage(tt.line)

			if msg != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, msg)
			}
		})
	}
}

func TestFormatJSONField(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		key      string
		expected string
	}{
		{
			name:     "string value",
			line:     `{"status":"ok"}`,
			key:      "status",
			expected: "status=ok",
		},
		{
			name:     "number value",
			line:     `{"code":200}`,
			key:      "code",
			expected: "code=200",
		},
		{
			name:     "boolean value",
			line:     `{"success":true}`,
			key:      "success",
			expected: "success=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := gjson.Get(tt.line, tt.key)
			result := formatJSONField(tt.key, val)

			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
