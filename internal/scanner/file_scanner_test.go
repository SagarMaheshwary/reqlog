package scanner

import (
	"container/heap"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

func TestFileScanner_Scan(t *testing.T) {
	dir := t.TempDir()

	logContent := `2024-03-10T12:00:00Z user=123 status=ok
2024-03-10T12:01:00Z user=456 status=fail
2024-03-10T12:02:00Z user=123 status=ok
`

	filePath := filepath.Join(dir, "auth.log")
	err := os.WriteFile(filePath, []byte(logContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		IgnoreCase:  false,
		Keys:        []string{"user"},
	}

	fs := NewFileScanner(cfg, parser.TextParser{})
	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}
	results := fs.Scan(files)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Service != "auth" {
			t.Errorf("expected service auth, got %s", r.Service)
		}
	}
}

func TestFileScanner_Scan_WithSince(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC()

	oldTime := now.Add(-10 * time.Minute).Format(time.RFC3339)
	newTime := now.Add(-1 * time.Minute).Format(time.RFC3339)

	logContent := oldTime + " user=123 status=ok\n" +
		newTime + " user=123 status=ok\n"

	filePath := filepath.Join(dir, "svc.log")
	if err := os.WriteFile(filePath, []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
		Since:       "5m", // should only include recent one
	}

	fs := NewFileScanner(cfg, parser.TextParser{})
	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}
	results := fs.Scan(files)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFileScanner_Scan_IgnoreCase(t *testing.T) {
	dir := t.TempDir()

	logContent := `2024-03-10T12:00:00Z user=ABC status=ok`

	filePath := filepath.Join(dir, "svc.log")
	if err := os.WriteFile(filePath, []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ScanConfig{
		Dir:         dir,
		SearchValue: "abc",
		Keys:        []string{"user"},
		IgnoreCase:  true,
	}

	fs := NewFileScanner(cfg, parser.TextParser{})
	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}
	results := fs.Scan(files)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFileScanner_Scan_IgnoresNonLogFiles(t *testing.T) {
	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "app.log"), []byte("invalid line"), 0644)

	cfg := ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
	}

	fs := NewFileScanner(cfg, parser.TextParser{})
	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}
	results := fs.Scan(files)

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestScan_MultiFile_GlobalLimit(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.log")
	file2 := filepath.Join(dir, "b.log")

	os.WriteFile(file1, []byte("id=123\nid=123\n"), 0644)
	os.WriteFile(file2, []byte("id=123\nid=123\n"), 0644)

	fs := NewFileScanner(ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
	}, mockParser{extractOK: true})

	results := fs.Scan([]string{file1, file2})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScan_SkipsFileErrors(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.log")
	invalid := filepath.Join(dir, "missing.log")

	os.WriteFile(valid, []byte("id=123\n"), 0644)

	fs := NewFileScanner(ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}, mockParser{extractOK: true})

	results := fs.Scan([]string{valid, invalid})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestScan_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "a.log")
	os.WriteFile(file, []byte("id=123"), 0644) // no newline

	fs := NewFileScanner(ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}, mockParser{extractOK: true})

	results := fs.Scan([]string{file})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestListLogFiles_FilterByService(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "auth.log"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "db.log"), []byte(""), 0644)

	fs := NewFileScanner(ScanConfig{
		Dir:      dir,
		Services: []string{"auth"},
	}, mockParser{})

	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestListLogFiles_RecursiveService(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "svc.log"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "svc-1.log"), []byte(""), 0644)

	fs := NewFileScanner(ScanConfig{
		Dir:      dir,
		Services: []string{"svc*"},
	}, mockParser{})

	files, err := fs.ListLogFiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
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

func TestProcessLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		search      string
		ignoreCase  bool
		extractOK   bool
		parseErr    error
		expectMatch bool
	}{
		{
			name:        "match success",
			line:        "id=123 hello",
			search:      "123",
			extractOK:   true,
			expectMatch: true,
		},
		{
			name:        "no contains match",
			line:        "hello world",
			search:      "123",
			extractOK:   true,
			expectMatch: false,
		},
		{
			name:        "extract fails",
			line:        "id=123",
			search:      "123",
			extractOK:   false,
			expectMatch: false,
		},
		{
			name:        "parse fails",
			line:        "id=123",
			search:      "123",
			extractOK:   true,
			parseErr:    errors.New("parse error"),
			expectMatch: false,
		},
		{
			name:        "ignore case match",
			line:        "ID=123",
			search:      "123",
			ignoreCase:  true,
			extractOK:   true,
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFileScanner(ScanConfig{
				SearchValue: tt.search,
				IgnoreCase:  tt.ignoreCase,
				Keys:        []string{"id"},
			}, mockParser{
				extractOK: tt.extractOK,
				parseErr:  tt.parseErr,
			})

			_, ok := fs.processLine(tt.line, "svc")

			if ok != tt.expectMatch {
				t.Fatalf("expected %v, got %v", tt.expectMatch, ok)
			}
		})
	}
}
func TestAddEntry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		limit    int
		input    []time.Duration
		expected int
	}{
		{
			name:     "no limit",
			limit:    0,
			input:    []time.Duration{1, 2, 3},
			expected: 3,
		},
		{
			name:     "limit enforced",
			limit:    2,
			input:    []time.Duration{1, 2, 3},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFileScanner(ScanConfig{Limit: tt.limit}, mockParser{})
			var results []domain.LogEntry
			var h entryHeap

			if tt.limit > 0 {
				heap.Init(&h)
			}

			for _, d := range tt.input {
				entry := domain.LogEntry{
					Timestamp: now.Add(d * time.Minute),
				}
				fs.addEntry(entry, &results, &h)
			}

			if tt.limit <= 0 {
				if len(results) != tt.expected {
					t.Fatalf("expected %d results, got %d", tt.expected, len(results))
				}
			} else {
				if h.Len() != tt.expected {
					t.Fatalf("expected heap size %d, got %d", tt.expected, h.Len())
				}
			}
		})
	}
}
