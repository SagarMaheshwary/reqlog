package scanner

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestLineProcessor_ProcessLine_TextMode(t *testing.T) {
	tp := NewTimeParser()

	tests := []struct {
		name        string
		line        string
		searchValue string
		ignoreCase  bool
		keys        []string
		expectOK    bool
	}{
		{
			name:        "valid match",
			line:        "2024-03-10T12:00:00Z request_id=req123 success message",
			searchValue: "req123",
			keys:        []string{"request_id"},
			expectOK:    true,
		},
		{
			name:        "no match",
			line:        "2024-03-10T12:00:00Z request_id=req123 success message",
			searchValue: "req999",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "no match (bypass fast prefilter)",
			line:        "2024-03-10T12:00:00Z request_id=req123 success message",
			searchValue: "req123",
			keys:        []string{"trace_id"},
			expectOK:    false,
		},
		{
			name:        "case insensitive match",
			line:        "2024-03-10T12:00:00Z request_id=REQ123 success message",
			searchValue: "req123",
			keys:        []string{"request_id"},
			ignoreCase:  true,
			expectOK:    true,
		},
		{
			name:        "invalid timestamp",
			line:        "invalid request_id=req123 message",
			searchValue: "req123",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "too few fields",
			line:        "onlyonefield",
			searchValue: "onlyonefield",
			keys:        []string{},
			expectOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ScanConfig{
				SearchValue: tt.searchValue,
				IgnoreCase:  tt.ignoreCase,
				Keys:        tt.keys,
				JSONMode:    false,
			}

			lp := NewLineProcessor(cfg, tp)

			entry, ok := lp.ProcessLine(tt.line, "svc")

			if ok != tt.expectOK {
				t.Fatalf("expected ok=%v, got %v", tt.expectOK, ok)
			}

			if ok {
				if entry.Service != "svc" {
					t.Fatalf("expected service=svc, got %s", entry.Service)
				}
			}
		})
	}
}

func TestLineProcessor_ProcessLine_JSONMode(t *testing.T) {
	tp := NewTimeParser()

	tests := []struct {
		name        string
		line        string
		searchValue string
		ignoreCase  bool
		keys        []string
		expectOK    bool
	}{
		{
			name:        "valid json match",
			line:        `{"request_id":"abc123","timestamp":"2024-03-10T12:00:00Z","msg":"ok"}`,
			searchValue: "abc123",
			keys:        []string{"request_id"},
			expectOK:    true,
		},
		{
			name:        "no match",
			line:        `{"request_id":"abc123","timestamp":"2024-03-10T12:00:00Z","msg":"ok"}`,
			searchValue: "xyz",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "no match (bypass fast prefilter)",
			line:        `{"request_id":"abc123","timestamp":"2024-03-10T12:00:00Z","msg":"ok"}`,
			searchValue: "abc123",
			keys:        []string{"trace_id"},
			expectOK:    false,
		},
		{
			name:        "no match (case insensitive)",
			line:        `{"request_id":"abc123","timestamp":"2024-03-10T12:00:00Z","msg":"ok"}`,
			searchValue: "xyz",
			keys:        []string{"request_id"},
			expectOK:    false,
			ignoreCase:  true,
		},
		{
			name:        "missing timestamp",
			line:        `{"request_id":"abc123"}`,
			searchValue: "abc123",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "invalid timestamp",
			line:        `{"timestamp":"invalid","request_id":"abc123"}`,
			searchValue: "abc123",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "invalid json",
			line:        `{"request_id":abc123`,
			searchValue: "abc123",
			keys:        []string{"request_id"},
			expectOK:    false,
		},
		{
			name:        "case insensitive match",
			line:        `{"request_id":"ABC123","timestamp":"2024-03-10T12:00:00Z","msg":"ok"}`,
			searchValue: "abc123",
			keys:        []string{"request_id"},
			ignoreCase:  true,
			expectOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ScanConfig{
				SearchValue: tt.searchValue,
				IgnoreCase:  tt.ignoreCase,
				Keys:        tt.keys,
				JSONMode:    true,
			}

			lp := NewLineProcessor(cfg, tp)

			entry, ok := lp.ProcessLine(tt.line, "svc")

			if ok != tt.expectOK {
				t.Fatalf("expected ok=%v, got %v", tt.expectOK, ok)
			}

			if ok {
				if entry.Service != "svc" {
					t.Fatalf("unexpected service")
				}
			}
		})
	}
}

func TestLineProcessor_ParseOnly(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *ScanConfig
		line      string
		service   string
		wantOK    bool
		wantMsg   string
		wantSvc   string
		wantIsNil bool
	}{
		{
			name:    "valid text log",
			cfg:     &ScanConfig{},
			line:    "2024-03-10T12:00:00Z user=999 status=ok",
			service: "auth",
			wantOK:  true,
			wantMsg: "user=999 status=ok",
			wantSvc: "auth",
		},
		{
			name:      "invalid timestamp",
			cfg:       &ScanConfig{},
			line:      "invalid user=123",
			service:   "auth",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name:      "missing key=value field",
			cfg:       &ScanConfig{},
			line:      "2024-03-10T12:00:00Z",
			service:   "auth",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name: "valid json log",
			cfg: &ScanConfig{
				JSONMode: true,
			},
			line:    `{"time":"2024-03-10T12:00:00Z","user":"999","status":"ok"}`,
			service: "api",
			wantOK:  true,
			wantSvc: "api",
		},
		{
			name: "invalid json log",
			cfg: &ScanConfig{
				JSONMode: true,
			},
			line:      `{"time":"2024-03-10T12:00:00Z"`,
			service:   "api",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name: "json missing timestamp",
			cfg: &ScanConfig{
				JSONMode: true,
			},
			line:      `{"user":"123"}`,
			service:   "api",
			wantOK:    false,
			wantIsNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewLineProcessor(tt.cfg, NewTimeParser())

			entry, ok := lp.ParseOnly(tt.line, tt.service)

			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}

			if tt.wantIsNil {
				if entry != nil {
					t.Fatalf("expected nil entry, got %+v", entry)
				}
				return
			}

			if entry == nil {
				t.Fatal("expected non-nil entry")
			}

			if tt.wantMsg != "" && entry.Message != tt.wantMsg {
				t.Fatalf(
					"expected message %q, got %q",
					tt.wantMsg,
					entry.Message,
				)
			}

			if entry.Service != tt.wantSvc {
				t.Fatalf(
					"expected service %q, got %q",
					tt.wantSvc,
					entry.Service,
				)
			}

			if entry.Timestamp.IsZero() {
				t.Fatal("expected non-zero timestamp")
			}
		})
	}
}

func TestLineProcessor_JSONTimestampKeyCaching(t *testing.T) {
	tp := NewTimeParser()

	cfg := &ScanConfig{
		SearchValue: "abc",
		Keys:        []string{"request_id"},
		JSONMode:    true,
	}

	lp := NewLineProcessor(cfg, tp)

	// First line: timestamp key = "timestamp"
	line1 := `{"request_id":"abc","timestamp":"2024-03-10T12:00:00Z"}`
	_, ok := lp.ProcessLine(line1, "svc")
	if !ok {
		t.Fatalf("expected first parse success")
	}

	// Second line: timestamp moved → should FAIL due to cached key
	line2 := `{"request_id":"abc","time":"2024-03-10T12:00:00Z"}`
	_, ok = lp.ProcessLine(line2, "svc")
	if ok {
		t.Fatalf("expected failure due to cached timestamp key")
	}
}

func TestExtractJSONField(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		keys     []string
		expected string
		ok       bool
	}{
		{
			name:     "single key match",
			json:     `{"request_id":"abc123"}`,
			keys:     []string{"request_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "first matching key wins",
			json:     `{"id":"1","request_id":"abc123"}`,
			keys:     []string{"request_id", "id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "fallback to second key",
			json:     `{"id":"1"}`,
			keys:     []string{"request_id", "id"},
			expected: "1",
			ok:       true,
		},
		{
			name:     "nested key",
			json:     `{"meta":{"request_id":"abc123"}}`,
			keys:     []string{"meta.request_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name: "no match",
			json: `{"foo":"bar"}`,
			keys: []string{"request_id"},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := gjson.Parse(tt.json)

			val, ok := extractJSONField(obj, tt.keys)

			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}

			if val != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, val)
			}
		})
	}
}

func TestExtractTextField(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		keys     []string
		expected string
		ok       bool
	}{
		{
			name:     "basic match",
			line:     "ts req_id=abc123 status=ok",
			keys:     []string{"req_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "skip first field (timestamp)",
			line:     "2024-03-10T12:00:00Z req_id=abc123",
			keys:     []string{"req_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "multiple keys first wins",
			line:     "ts id=1 req_id=abc123",
			keys:     []string{"req_id", "id"},
			expected: "1",
			ok:       true,
		},
		{
			name:     "quoted value double quotes",
			line:     `ts req_id="abc123"`,
			keys:     []string{"req_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "quoted value single quotes",
			line:     `ts req_id='abc123'`,
			keys:     []string{"req_id"},
			expected: "abc123",
			ok:       true,
		},
		{
			name:     "value with equals inside",
			line:     "ts req_id=abc=123",
			keys:     []string{"req_id"},
			expected: "abc=123",
			ok:       true,
		},
		{
			name: "invalid kv pair ignored",
			line: "ts req_id status=ok",
			keys: []string{"req_id"},
			ok:   false,
		},
		{
			name: "no match",
			line: "ts foo=bar",
			keys: []string{"req_id"},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Fields(tt.line)

			val, ok := extractTextField(parts, tt.keys)

			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}

			if val != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, val)
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

func TestBuildJSONMessage(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		tsKey  string
		expect string
	}{
		{
			name:   "exclude timestamp",
			json:   `{"timestamp":"1","a":"x","b":"y"}`,
			tsKey:  "timestamp",
			expect: "a=x b=y",
		},
		{
			name:   "sorted output",
			json:   `{"b":"y","a":"x"}`,
			tsKey:  "timestamp",
			expect: "a=x b=y",
		},
		{
			name:   "non-string values use raw",
			json:   `{"a":1,"b":true}`,
			tsKey:  "timestamp",
			expect: "a=1 b=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := gjson.Parse(tt.json)

			msg := buildJSONMessage(obj, tt.tsKey)

			if msg != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, msg)
			}
		})
	}
}

func TestFormatJSONField(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		key      string
		expected string
	}{
		{
			name:     "string value",
			json:     `{"a":"x"}`,
			key:      "a",
			expected: "a=x",
		},
		{
			name:     "number value",
			json:     `{"a":1}`,
			key:      "a",
			expected: "a=1",
		},
		{
			name:     "bool value",
			json:     `{"a":true}`,
			key:      "a",
			expected: "a=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := gjson.Parse(tt.json)
			val := obj.Get(tt.key)

			got := formatJSONField(tt.key, val)

			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestExtractTextMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ts message here", "message here"},
		{"ts    message here", "message here"},
		{"singlefield", ""},
		{"ts ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractTextMessage(tt.input)

			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
