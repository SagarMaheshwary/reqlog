package parser

import (
	"testing"
	"time"
)

func TestTextParser_Parse(t *testing.T) {
	p := TextParser{}

	tests := []struct {
		name        string
		line        string
		service     string
		expectError bool
		expectedMsg string
	}{
		{
			name:        "valid log line",
			line:        "2024-03-10T12:00:00Z user logged_in success",
			service:     "auth",
			expectError: false,
			expectedMsg: "user logged_in success",
		},
		{
			name:        "invalid format (too short)",
			line:        "invalid",
			service:     "svc",
			expectError: true,
		},
		{
			name:        "invalid timestamp",
			line:        "not-a-time message here",
			service:     "svc",
			expectError: true,
		},
		{
			name:        "timestamp only (no message)",
			line:        "2024-03-10T12:00:00Z",
			service:     "svc",
			expectError: true,
		},
		{
			name:        "message with multiple spaces",
			line:        "2024-03-10T12:00:00Z hello   world",
			service:     "svc",
			expectError: false,
			expectedMsg: "hello world", // strings.Fields normalizes spacing
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

			expectedTime, _ := time.Parse(time.RFC3339, "2024-03-10T12:00:00Z")

			if !entry.Timestamp.Equal(expectedTime) {
				t.Errorf("timestamp mismatch")
			}

			if entry.Service != tt.service {
				t.Errorf("expected service %s, got %s", tt.service, entry.Service)
			}

			if entry.Message != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, entry.Message)
			}
		})
	}
}

func TestTextParser_ExtractField(t *testing.T) {
	p := TextParser{}

	tests := []struct {
		name     string
		line     string
		keys     []string
		expected string
		found    bool
	}{
		{
			name:     "single key match",
			line:     "2024-03-10T12:00:00Z user=123 status=ok",
			keys:     []string{"status"},
			expected: "ok",
			found:    true,
		},
		{
			name:     "multiple keys first match in line",
			line:     "2024-03-10T12:00:00Z a=1 b=2",
			keys:     []string{"b", "a"},
			expected: "1", // line order wins
			found:    true,
		},
		{
			name:     "no match",
			line:     "2024-03-10T12:00:00Z foo=bar",
			keys:     []string{"status"},
			expected: "",
			found:    false,
		},
		{
			name:     "ignore non key-value parts",
			line:     "2024-03-10T12:00:00Z hello world status=ok",
			keys:     []string{"status"},
			expected: "ok",
			found:    true,
		},
		{
			name:     "invalid line",
			line:     "invalid",
			keys:     []string{"key"},
			expected: "",
			found:    false,
		},
		{
			name:     "value with equals",
			line:     "2024-03-10T12:00:00Z token=abc=def",
			keys:     []string{"token"},
			expected: "abc=def",
			found:    true,
		},
		{
			name:     "quoted value (before stripQuotes integration)",
			line:     `2024-03-10T12:00:00Z status="ok"`,
			keys:     []string{"status"},
			expected: "ok",
			found:    true,
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

func TestStripQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double quotes",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "single quotes",
			input:    `'hello'`,
			expected: "hello",
		},
		{
			name:     "no quotes",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "mismatched quotes",
			input:    `"hello`,
			expected: `"hello`,
		},
		{
			name:     "single char quoted",
			input:    `"a"`,
			expected: "a",
		},
		{
			name:     "empty quoted",
			input:    `""`,
			expected: "",
		},
		{
			name:     "value with equals inside quotes",
			input:    `"abc=def"`,
			expected: "abc=def",
		},
		{
			name:     "only one char",
			input:    `"`,
			expected: `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripQuotes(tt.input)

			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
