package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

func newTestFileScanner(cfg *ScanConfig, p parser.LogParser) *FileScanner {
	lp := NewLineProcessor(cfg, p)
	return NewFileScanner(lp)
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func TestFileScanner_Scan(t *testing.T) {
	dir := t.TempDir()

	logContent := `2024-03-10T12:00:00Z user=123 status=ok
2024-03-10T12:01:00Z user=456 status=fail
2024-03-10T12:02:00Z user=123 status=ok
`
	writeFile(t, filepath.Join(dir, "auth.log"), []byte(logContent))

	cfg := &ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		IgnoreCase:  false,
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, &parser.TextParser{})
	fs := NewFileScanner(lp)

	files, err := fs.ListSources()
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
	writeFile(t, filepath.Join(dir, "svc.log"), []byte(logContent))

	cfg := &ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
		Since:       "5m", // should only include recent one
	}
	lp := NewLineProcessor(cfg, parser.TextParser{})

	fs := NewFileScanner(lp)
	files, err := fs.ListSources()
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
	writeFile(t, filepath.Join(dir, "svc.log"), []byte(logContent))

	cfg := &ScanConfig{
		Dir:         dir,
		SearchValue: "abc",
		Keys:        []string{"user"},
		IgnoreCase:  true,
	}
	lp := NewLineProcessor(cfg, parser.TextParser{})

	fs := NewFileScanner(lp)
	files, err := fs.ListSources()
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

	writeFile(t, filepath.Join(dir, "file.txt"), []byte("test"))
	writeFile(t, filepath.Join(dir, "app.log"), []byte("invalid line"))

	cfg := &ScanConfig{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, parser.TextParser{})

	fs := NewFileScanner(lp)
	files, err := fs.ListSources()
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

	writeFile(t, file1, []byte("id=123\nid=123\n"))
	writeFile(t, file2, []byte("id=123\nid=123\n"))

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
	}
	fs := newTestFileScanner(cfg, mockParser{extractOK: true})

	results := fs.Scan([]string{file1, file2})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScan_SkipsFileErrors(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.log")
	invalid := filepath.Join(dir, "missing.log")

	writeFile(t, valid, []byte("id=123\n"))

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg, mockParser{extractOK: true})

	results := fs.Scan([]string{valid, invalid})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestScan_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "a.log")
	writeFile(t, file, []byte("id=123")) // no newline

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg, mockParser{extractOK: true})

	results := fs.Scan([]string{file})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestListSources(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		cfg       *ScanConfig
		wantFiles func(dir string) []string
	}{
		{
			name: "skip non-log and subdirectories",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "auth.log"), []byte(""))
				writeFile(t, filepath.Join(dir, "non-log-file"), []byte(""))

				os.Mkdir(filepath.Join(dir, "sub-dir"), 0755)
				writeFile(t, filepath.Join(dir, "sub-dir", "svc.log"), []byte(""))
			},
			cfg: &ScanConfig{
				Services: []string{},
			},
			wantFiles: func(dir string) []string {
				return []string{filepath.Join(dir, "auth.log")}
			},
		},
		{
			name: "recursive skip non-log and include subdir logs",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "auth.log"), []byte(""))
				writeFile(t, filepath.Join(dir, "non-log-file"), []byte(""))

				os.Mkdir(filepath.Join(dir, "sub-dir"), 0755)
				writeFile(t, filepath.Join(dir, "sub-dir", "svc.log"), []byte(""))
				writeFile(t, filepath.Join(dir, "sub-dir", "non-log-file"), []byte(""))
			},
			cfg: &ScanConfig{
				Services:  []string{"svc"},
				Recursive: true,
			},
			wantFiles: func(dir string) []string {
				return []string{filepath.Join(dir, "sub-dir/svc.log")}
			},
		},
		{
			name: "filter by service",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "auth.log"), []byte(""))
				writeFile(t, filepath.Join(dir, "db.log"), []byte(""))
			},
			cfg: &ScanConfig{
				Services: []string{"auth", " "}, //skip empty strings
			},
			wantFiles: func(dir string) []string {
				return []string{filepath.Join(dir, "auth.log")}
			},
		},
		{
			name: "wildcard service",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "svc.log"), []byte(""))
				writeFile(t, filepath.Join(dir, "svc-1.log"), []byte(""))
			},
			cfg: &ScanConfig{
				Services:  []string{"svc*"},
				Recursive: true,
			},
			wantFiles: func(dir string) []string {
				return []string{filepath.Join(dir, "svc.log"), filepath.Join(dir, "svc-1.log")}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			tt.setup(dir)
			tt.cfg.Dir = dir

			fs := newTestFileScanner(tt.cfg, mockParser{})

			files, err := fs.ListSources()
			if err != nil {
				t.Fatal(err)
			}

			wantFiles := tt.wantFiles(dir)

			sort.Strings(files)
			sort.Strings(wantFiles)

			if !reflect.DeepEqual(files, wantFiles) {
				t.Fatalf("expected %v, got %v", wantFiles, files)
			}
		})
	}
}
