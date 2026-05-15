package scanner

import (
	"encoding/json"
	"reflect"
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
			cfg := &Config{
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
			cfg := &Config{
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
		cfg       *Config
		line      string
		service   string
		wantOK    bool
		wantMsg   string
		wantSvc   string
		wantIsNil bool
	}{
		{
			name:    "valid text log",
			cfg:     &Config{},
			line:    "2024-03-10T12:00:00Z some message user=999 status=ok",
			service: "auth",
			wantOK:  true,
			wantMsg: "some message",
			wantSvc: "auth",
		},
		{
			name:      "invalid timestamp",
			cfg:       &Config{},
			line:      "invalid user=123",
			service:   "auth",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name:      "missing key=value field",
			cfg:       &Config{},
			line:      "2024-03-10T12:00:00Z",
			service:   "auth",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name: "valid json log",
			cfg: &Config{
				JSONMode: true,
			},
			line:    `{"time":"2024-03-10T12:00:00Z","user":"999","status":"ok"}`,
			service: "api",
			wantOK:  true,
			wantSvc: "api",
		},
		{
			name: "invalid json log",
			cfg: &Config{
				JSONMode: true,
			},
			line:      `{"time":"2024-03-10T12:00:00Z"`,
			service:   "api",
			wantOK:    false,
			wantIsNil: true,
		},
		{
			name: "json missing timestamp",
			cfg: &Config{
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

	cfg := &Config{
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

func TestExtractJSONFields(t *testing.T) {
	MessageKeys = map[string]struct{}{
		"msg":     {},
		"message": {},
	}

	tests := []struct {
		name        string
		input       string
		tsKey       string
		wantMessage string
		wantFields  map[string]any
	}{
		{
			name: "basic fields with message",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"level": "info",
				"msg": "hello world",
				"user": "123"
			}`,
			tsKey:       "ts",
			wantMessage: "hello world",
			wantFields: map[string]any{
				"level": "info",
				"user":  "123",
			},
		},
		{
			name: "number and boolean parsing",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"count": 42,
				"ok": true,
				"msg": "done"
			}`,
			tsKey:       "ts",
			wantMessage: "done",
			wantFields: map[string]any{
				"count": float64(42),
				"ok":    true,
			},
		},
		{
			name: "null and nested json",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"data": {"a": 1},
				"extra": null
			}`,
			tsKey:       "ts",
			wantMessage: "",
			wantFields: map[string]any{
				"data": map[string]any{
					"a": float64(1),
				},
				"extra": nil,
			},
		},
		{
			name: "array parsing",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"tags": ["a", "b"]
			}`,
			tsKey: "ts",
			wantFields: map[string]any{
				"tags": []any{"a", "b"},
			},
		},
		{
			name: "timestamp key excluded",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"user": "123"
			}`,
			tsKey: "ts",
			wantFields: map[string]any{
				"user": "123",
			},
		},
		{
			name: "message only once (first match wins)",
			input: `{
				"ts": "2024-03-10T12:00:00Z",
				"msg": "first",
				"message": "second",
				"user": "123"
			}`,
			tsKey:       "ts",
			wantMessage: "first",
			wantFields: map[string]any{
				"user": "123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := gjson.Parse(tt.input)

			got := extractJSONFields(obj, tt.tsKey)

			if got.Message != tt.wantMessage {
				t.Fatalf("message: expected %q got %q", tt.wantMessage, got.Message)
			}

			if len(got.Fields) != len(tt.wantFields) {
				t.Fatalf("fields length: expected %d got %d", len(tt.wantFields), len(got.Fields))
			}

			for k, v := range tt.wantFields {
				gv, ok := got.Fields[k]
				if !ok {
					t.Fatalf("missing field %q", k)
				}

				// deep compare JSON-safe values
				gotB, _ := json.Marshal(gv)
				wantB, _ := json.Marshal(v)

				if string(gotB) != string(wantB) {
					t.Fatalf("field %q: expected %v got %v", k, v, gv)
				}
			}
		})
	}
}

func TestExtractTextFields(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  ParsedFields
	}{
		{
			name: "fields first then message",
			input: []string{
				"level=info",
				"request_id=abc123",
				"start",
				"request",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"level":      "info",
					"request_id": "abc123",
				},
				Message: "start request",
			},
		},
		{
			name: "message first then fields",
			input: []string{
				"start",
				"request",
				"level=info",
				"request_id=abc123",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"level":      "info",
					"request_id": "abc123",
				},
				Message: "start request",
			},
		},
		{
			name: "mixed ordering",
			input: []string{
				"start",
				"level=info",
				"request",
				"request_id=abc123",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"level":      "info",
					"request_id": "abc123",
				},
				Message: "start request",
			},
		},
		{
			name: "typed values",
			input: []string{
				"ok=true",
				"count=10",
				"rate=1.5",
				"done",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"ok":    true,
					"count": int64(10),
					"rate":  1.5,
				},
				Message: "done",
			},
		},
		{
			name: "no fields only message",
			input: []string{
				"hello",
				"world",
			},
			want: ParsedFields{
				Fields:  map[string]any{},
				Message: "hello world",
			},
		},
		{
			name: "no message only fields",
			input: []string{
				"level=info",
				"request_id=abc123",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"level":      "info",
					"request_id": "abc123",
				},
				Message: "",
			},
		},
		{
			name:  "empty input",
			input: []string{},
			want: ParsedFields{
				Fields:  map[string]any{},
				Message: "",
			},
		},
		{
			name: "value with equals ignored split safety",
			input: []string{
				"query=a=b=c",
				"run",
			},
			want: ParsedFields{
				Fields: map[string]any{
					"query": "a=b=c",
				},
				Message: "run",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextFields(tt.input)

			if !reflect.DeepEqual(got.Fields, tt.want.Fields) {
				t.Fatalf("fields mismatch\nwant: %+v\ngot:  %+v", tt.want.Fields, got.Fields)
			}

			if got.Message != tt.want.Message {
				t.Fatalf("message mismatch\nwant: %q\ngot:  %q", tt.want.Message, got.Message)
			}
		})
	}
}
