package scanner

import (
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

	results, err := fs.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	results, err := fs.Scan()
	if err != nil {
		t.Fatal(err)
	}

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

	results, err := fs.Scan()
	if err != nil {
		t.Fatal(err)
	}

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

	results, err := fs.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
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
			result := passesSince(entry, tt.since)

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
