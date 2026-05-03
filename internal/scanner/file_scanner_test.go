package scanner

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func newTestFileScanner(cfg *ScanConfig) *FileScanner {
	lp := NewLineProcessor(cfg, NewTimeParser())
	return NewFileScanner(lp, time.Second, io.Discard, io.Discard)
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
	lp := NewLineProcessor(cfg, NewTimeParser())
	fs := NewFileScanner(lp, time.Second, io.Discard, io.Discard)

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
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(lp, time.Second, io.Discard, io.Discard)
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
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(lp, time.Second, io.Discard, io.Discard)
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
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(lp, time.Second, io.Discard, io.Discard)
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

	writeFile(t, file1, []byte("2024-03-10T12:00:00Z id=123\nid=123\n"))
	writeFile(t, file2, []byte("2024-03-10T12:00:00Z id=123\nid=123\n"))

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
	}
	fs := newTestFileScanner(cfg)

	results := fs.Scan([]string{file1, file2})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScan_SkipsFileErrors(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.log")
	invalid := filepath.Join(dir, "missing.log")

	writeFile(t, valid, []byte("2024-03-10T12:00:00Z id=123\n"))

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg)

	results := fs.Scan([]string{valid, invalid})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestScan_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "a.log")
	writeFile(t, file, []byte("2024-03-10T12:00:00Z id=123")) // no newline

	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg)

	results := fs.Scan([]string{file})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFileScanner_Scan_ErrorLogging(t *testing.T) {
	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	// pass a missing file to trigger error
	files := []string{"/tmp/nonexistent.log"}

	var out bytes.Buffer
	var errOut bytes.Buffer
	fs := NewFileScanner(lp, time.Second, &out, &errOut)

	results := fs.Scan(files)

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if !strings.Contains(errOut.String(), "/tmp/nonexistent.log") {
		t.Errorf("expected error log, got %q", errOut.String())
	}
}

func TestFileScanner_Scan_JSON(t *testing.T) {
	tests := []struct {
		name        string
		logLines    []string
		expectedLen int
	}{
		{
			name: "valid json logs",
			logLines: []string{
				`{"time":"2024-03-10T12:00:00Z","user":"123","status":"ok"}`,
				`{"time":"2024-03-10T12:00:00Z","user":"456","status":"fail"}`,
			},
			expectedLen: 1,
		},
		{
			name: "invalid json lines are skipped",
			logLines: []string{
				`{"time":"2024-03-10T12:00:00Z","user":"123"}`,
				`invalid json`,
				`{"time":"2024-03-10T12:00:00Z","user":"123"}`,
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			content := strings.Join(tt.logLines, "\n")
			writeFile(t, filepath.Join(dir, "svc.log"), []byte(content))

			cfg := &ScanConfig{
				Dir:         dir,
				SearchValue: "123",
				Keys:        []string{"user"},
				JSONMode:    true,
			}

			lp := NewLineProcessor(cfg, NewTimeParser())
			fs := NewFileScanner(lp, time.Second, io.Discard, io.Discard)

			files, err := fs.ListSources()
			if err != nil {
				t.Fatal(err)
			}

			results := fs.Scan(files)

			if len(results) != tt.expectedLen {
				t.Fatalf("expected %d results, got %d", tt.expectedLen, len(results))
			}
		})
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

			fs := newTestFileScanner(tt.cfg)

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

func TestFileScanner_Follow(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name   string
		files  map[string]string
		want   []string
		append func() // optional append after first read
	}{
		{
			name: "single file logs",
			files: map[string]string{
				"auth.log": "2024-03-10T12:00:00Z user=123 status=ok\n",
			},
			want: []string{"2024-03-10T12:00:00Z [auth] user=123 status=ok"},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"auth.log": "2024-03-10T12:00:00Z user=123\n",
				"db.log":   "2024-03-10T12:00:00Z user=123\n",
			},
			want: []string{"2024-03-10T12:00:00Z [auth] user=123", "2024-03-10T12:00:00Z [db] user=123"},
		},
		{
			name: "ignore lines that don't match search",
			files: map[string]string{
				"svc.log": "2024-03-10T12:00:00Z user=123\nother=xyz\n",
			},
			want: []string{"2024-03-10T12:00:00Z [svc] user=123"},
		},
		{
			name: "new lines appended",
			files: map[string]string{
				"append.log": "2024-03-10T12:00:00Z user=123 line1\n",
			},
			want: []string{"2024-03-10T12:00:00Z [append] user=123 line1", "2024-03-10T12:00:00Z [append] user=123 line2"},
			append: func() {
				fpath := filepath.Join(dir, "append.log")
				f, _ := os.OpenFile(fpath, os.O_APPEND|os.O_WRONLY, 0644)
				defer f.Close()
				f.WriteString("2024-03-10T12:00:00Z user=123 line2\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []string{}
			for fname, content := range tt.files {
				path := filepath.Join(dir, fname)
				writeFile(t, path, []byte(content))
				files = append(files, path)
			}

			var out bytes.Buffer

			cfg := &ScanConfig{SearchValue: "123", Keys: []string{"user"}}
			lp := NewLineProcessor(cfg, NewTimeParser())
			fs := NewFileScanner(lp, 10*time.Millisecond, &out, io.Discard)

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			// Optionally append new lines after start
			if tt.append != nil {
				go func() {
					time.Sleep(100 * time.Millisecond)
					tt.append()
				}()
			}

			fs.Follow(ctx, files, &mockFormatter{})

			lines := strings.FieldsFunc(out.String(), func(r rune) bool { return r == '\n' || r == '\r' })

			sort.Strings(lines)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)

			if len(lines) != len(want) {
				t.Fatalf("expected %v lines, got %v", len(want), len(lines))
			}
			for i := range lines {
				if lines[i] != want[i] {
					t.Errorf("line %d: expected %q, got %q", i, want[i], lines[i])
				}
			}
		})
	}
}

func TestFileScanner_Follow_Errors(t *testing.T) {
	cfg := &ScanConfig{
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	var out bytes.Buffer
	var errOut bytes.Buffer
	fs := NewFileScanner(lp, 10*time.Millisecond, &out, &errOut)

	// pass a missing file to trigger error
	files := []string{"/tmp/nonexistent.log"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	fs.Follow(ctx, files, &mockFormatter{})

	if !strings.Contains(errOut.String(), "error scanning /tmp/nonexistent.log") {
		t.Errorf("expected error log, got %q", errOut.String())
	}
}
