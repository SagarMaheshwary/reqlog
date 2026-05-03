package scanner

import (
	"testing"
	"time"
)

func mustParseRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestTimeParser_Parse_SuccessCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		source   string
		expected time.Time
	}{
		{
			name:     "RFC3339",
			input:    "2024-01-02T15:04:05.999999999Z",
			source:   "app1",
			expected: mustParseRFC3339("2024-01-02T15:04:05.999999999Z"),
		},
		{
			name:     "Unix seconds",
			input:    "1700000000",
			source:   "app1",
			expected: time.Unix(1700000000, 0),
		},
		{
			name:     "Unix milliseconds",
			input:    "1700000000000",
			source:   "app1",
			expected: time.UnixMilli(1700000000000),
		},
		{
			name:     "Unix microseconds",
			input:    "1700000000000000",
			source:   "app1",
			expected: time.UnixMicro(1700000000000000),
		},
		{
			name:     "Unix nanoseconds",
			input:    "1700000000000000000",
			source:   "app1",
			expected: time.Unix(0, 1700000000000000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewTimeParser()

			ts, ok := p.Parse(tt.input, tt.source)
			if !ok {
				t.Fatalf("expected success, got failure")
			}

			if !ts.Equal(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, ts)
			}
		})
	}
}

func TestTimeParser_Parse_InvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		source string
	}{
		{"empty string", "", "app1"},
		{"random string", "not-a-time", "app1"},
		{"too short unix", "123", "app1"},
		{"unsupported length", "170000000000", "app1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewTimeParser()

			_, ok := p.Parse(tt.input, tt.source)
			if ok {
				t.Fatalf("expected failure for input: %s", tt.input)
			}
		})
	}
}

func TestTimeParser_Parse_CachingBehavior(t *testing.T) {
	tests := []struct {
		name  string
		steps []struct {
			input    string
			source   string
			expectOK bool
		}
	}{
		{
			name: "cache RFC then fail on unix",
			steps: []struct {
				input    string
				source   string
				expectOK bool
			}{
				{"2024-01-02T15:04:05Z", "app1", true}, // detect RFC
				{"1700000000", "app1", false},          // should fail (cached RFC)
			},
		},
		{
			name: "cache unix then fail on RFC",
			steps: []struct {
				input    string
				source   string
				expectOK bool
			}{
				{"1700000000", "app1", true},            // detect unix
				{"2024-01-02T15:04:05Z", "app1", false}, // should fail
			},
		},
		{
			name: "different sources do not share cache",
			steps: []struct {
				input    string
				source   string
				expectOK bool
			}{
				{"1700000000", "app1", true},
				{"2024-01-02T15:04:05Z", "app2", true}, // separate cache
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewTimeParser()

			for i, step := range tt.steps {
				_, ok := p.Parse(step.input, step.source)
				if ok != step.expectOK {
					t.Fatalf("step %d: expected ok=%v, got %v (input=%s, source=%s)",
						i, step.expectOK, ok, step.input, step.source)
				}
			}
		})
	}
}
