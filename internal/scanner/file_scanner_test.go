package scanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/diagnostics"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

func newTestFileScanner(cfg *Config) *FileScanner {
	lp := NewLineProcessor(cfg, NewTimeParser())
	return NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})
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

	cfg := &Config{
		Dir:         dir,
		SearchValue: "123",
		IgnoreCase:  false,
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())
	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})

	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	results, err := fs.Scan(t.Context(), files)
	if err != nil {
		t.Fatal(err)
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
	writeFile(t, filepath.Join(dir, "svc.log"), []byte(logContent))

	cfg := &Config{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
		Since:       "5m", // should only include recent one
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})
	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	results, err := fs.Scan(t.Context(), files)
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
	writeFile(t, filepath.Join(dir, "svc.log"), []byte(logContent))

	cfg := &Config{
		Dir:         dir,
		SearchValue: "abc",
		Keys:        []string{"user"},
		IgnoreCase:  true,
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})
	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	results, err := fs.Scan(t.Context(), files)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFileScanner_Scan_Recursive(t *testing.T) {
	dir := t.TempDir()

	subDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	logContent := `2024-03-10T12:00:00Z user=abc status=ok`
	writeFile(
		t,
		filepath.Join(subDir, "auth.log"),
		[]byte(logContent),
	)

	cfg := &Config{
		Dir:         dir,
		SearchValue: "abc",
		Keys:        []string{"user"},
		Recursive:   true,
	}

	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})

	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d (%v)", len(files), files)
	}

	if files[0] != filepath.Join(subDir, "auth.log") {
		t.Fatalf("unexpected file: %v", files[0])
	}

	results, err := fs.Scan(t.Context(), files)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Service != "auth" {
		t.Fatalf("expected service auth, got %q", results[0].Service)
	}
}

func TestFileScanner_Scan_IgnoresNonLogFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "file.txt"), []byte("test"))
	writeFile(t, filepath.Join(dir, "app.log"), []byte("invalid line"))

	cfg := &Config{
		Dir:         dir,
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor:  lp,
		FollowInterval: time.Second,
		Out:            io.Discard,
		Now:            time.Now(),
		LogFileReader:  transport.NewLogFileReader(nil),
		Diagnostics:    diagnostics.NewDiagnostics(),
	})
	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	results, err := fs.Scan(t.Context(), files)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestScan_MultiFile_Limit(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.log")
	file2 := filepath.Join(dir, "b.log")

	writeFile(t, file1, []byte(`
2024-03-10T12:00:01Z id=123
2024-03-10T12:00:03Z id=123
2024-03-10T12:00:05Z id=123
`))

	writeFile(t, file2, []byte(`
2024-03-10T12:00:02Z id=123
2024-03-10T12:00:04Z id=123
2024-03-10T12:00:06Z id=123
`))

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
	}
	fs := newTestFileScanner(cfg)

	results, err := fs.Scan(t.Context(), []string{file1, file2})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestScan_MultiFile_Latest(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.log")
	file2 := filepath.Join(dir, "b.log")

	writeFile(t, file1, []byte(`
2024-03-10T12:00:01Z id=123
2024-03-10T12:00:03Z id=123
2024-03-10T12:00:05Z id=123
`))

	writeFile(t, file2, []byte(`
2024-03-10T12:00:02Z id=123
2024-03-10T12:00:04Z id=123
2024-03-10T12:00:06Z id=123
`))

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
		Latest:      true,
	}

	fs := newTestFileScanner(cfg)

	results, err := fs.Scan(t.Context(), []string{file1, file2})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	expected := []string{
		"2024-03-10T12:00:05Z",
		"2024-03-10T12:00:06Z",
	}

	for i, ts := range expected {
		expectedTime, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			t.Fatal(err)
		}

		if !results[i].Timestamp.Equal(expectedTime) {
			t.Fatalf(
				"result[%d]: expected %v, got %v",
				i,
				expectedTime,
				results[i].Timestamp,
			)
		}
	}
}

func TestScan_SkipsFileErrors(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.log")
	invalid := filepath.Join(dir, "missing.log")

	writeFile(t, valid, []byte("2024-03-10T12:00:00Z id=123\n"))

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg)

	results, err := fs.Scan(t.Context(), []string{valid, invalid})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestScan_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "a.log")
	writeFile(t, file, []byte("2024-03-10T12:00:00Z id=123")) // no newline

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"id"},
	}
	fs := newTestFileScanner(cfg)

	results, err := fs.Scan(t.Context(), []string{file})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFileScanner_Scan_InvalidSince(t *testing.T) {
	dir := t.TempDir()

	logContent := `2024-03-10T12:00:00Z user=ABC status=ok`
	writeFile(t, filepath.Join(dir, "svc.log"), []byte(logContent))

	cfg := &Config{
		Dir:         dir,
		SearchValue: "abc",
		Keys:        []string{"user"},
		IgnoreCase:  true,
		Since:       "invalid",
	}

	fs := newTestFileScanner(cfg)

	_, err := fs.Scan(t.Context(), []string{filepath.Join(dir, "svc.log")})
	if err == nil {
		t.Fatalf("expected error, got %v", err)
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

			cfg := &Config{
				Dir:         dir,
				SearchValue: "123",
				Keys:        []string{"user"},
				Format:      config.FormatJSON,
			}

			lp := NewLineProcessor(cfg, NewTimeParser())
			fs := NewFileScanner(&FileScannerOpts{
				LineProcessor:  lp,
				FollowInterval: time.Second,
				Out:            io.Discard,
				Now:            time.Now(),
				LogFileReader:  transport.NewLogFileReader(nil),
				Diagnostics:    diagnostics.NewDiagnostics(),
			})
			files, err := fs.ListSources(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			results, err := fs.Scan(t.Context(), files)
			if err != nil {
				t.Fatal(err)
			}

			if len(results) != tt.expectedLen {
				t.Fatalf("expected %d results, got %d", tt.expectedLen, len(results))
			}
		})
	}
}

func TestFileScanner_Scan_ContextWindow_BeforeAndAfter(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "svc.log")

	// Structure:
	// lines 1-2 -> context before match
	// line 3    -> match (triggers window)
	// line 4-5  -> after-context (must be included)
	// line 6    -> should NOT be processed (stop after context)
	writeFile(t, file, []byte(
		"2024-03-10T12:00:00Z some message user=ignore\n"+ // before
			"2024-03-10T12:00:01Z some message user=123\n"+ // before
			"2024-03-10T12:00:02Z some message user=123\n"+ // match
			"2024-03-10T12:00:03Z some message user=ignore\n"+ // after-context
			"2024-03-10T12:00:04Z some message user=ignore\n"+ // after-context
			"2024-03-10T12:00:05Z some message user=123\n", // must NOT be processed
	))

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
		Context:     2,
		Limit:       1,
	}

	fs := newTestFileScanner(cfg)

	results, err := fs.Scan(t.Context(), []string{file})
	if err != nil {
		t.Fatal(err)
	}

	got := make([]domain.LogEntry, 0, len(results))
	for _, r := range results {
		got = append(got, r)
	}

	expected := []domain.LogEntry{
		{
			Message:   "some message",
			Service:   "svc",
			IsContext: true,
			Fields: map[string]any{
				"user": "ignore",
			},
		},
		{
			Message: "some message",
			Fields: map[string]any{
				"user": "123",
			},
			IsContext: false,
		},
		{
			Message: "some message",
			Fields: map[string]any{
				"user": "123",
			},
			IsContext: true,
		},
		{
			Message:   "some message",
			IsContext: true,
			Fields: map[string]any{
				"user": "ignore",
			},
		},
	}
	for i, exp := range expected {
		if got[i].Message != exp.Message {
			t.Fatalf("msg[%d]: expected %q got %q", i, exp.Message, got[i].Message)
		}

		if got[i].IsContext != exp.IsContext {
			t.Fatalf("ctx[%d]: expected %v got %v", i, exp.IsContext, got[i].IsContext)
		}

		if !equalFields(got[i].Fields, exp.Fields) {
			t.Fatalf("fields[%d]: expected %v got %v", i, exp.Fields, got[i].Fields)
		}
	}
}

func TestListSources(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		cfg       *Config
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
			cfg: &Config{
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
			cfg: &Config{
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
			cfg: &Config{
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
			cfg: &Config{
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

			files, err := fs.ListSources(t.Context())
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

func TestListSources_OnErrorCallback(t *testing.T) {
	fs := newTestFileScanner(&Config{})

	fs.logFileReader = &mockLogFileReader{
		listFilesFn: func(
			ctx context.Context,
			dir string,
			opts transport.ListOptions,
		) ([]string, error) {
			if opts.OnError != nil {
				opts.OnError("/tmp/bad.log", errors.New("permission denied"))
			}

			return []string{"auth.log"}, nil
		},
	}

	files, err := fs.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(files, []string{"auth.log"}) {
		t.Fatalf("expected files returned, got %v", files)
	}

	entries := fs.diagnostics.Entries()

	if len(entries) != 1 {
		t.Fatalf("expected 1 diagnostic entry, got %d", len(entries))
	}

	entry := entries[0]

	if entry.Fields["level"] != "error" {
		t.Fatalf("expected error level, got %v", entry.Fields["level"])
	}

	if !strings.Contains(entry.Message, "Error listing file /tmp/bad.log") {
		t.Fatalf("unexpected message: %q", entry.Message)
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
				"auth.log": "2024-03-10T12:00:00Z request arrived status=ok user=123\n",
			},
			want: []string{"2024-03-10T12:00:00Z [auth] request arrived status=ok user=123"},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"auth.log": "2024-03-10T12:00:00Z request arrived user=123\n",
				"db.log":   "2024-03-10T12:00:00Z request arrived user=123\n",
			},
			want: []string{"2024-03-10T12:00:00Z [auth] request arrived user=123", "2024-03-10T12:00:00Z [db] request arrived user=123"},
		},
		{
			name: "ignore lines that don't match search",
			files: map[string]string{
				"svc.log": "2024-03-10T12:00:00Z request arrived user=123\nother=xyz\n",
			},
			want: []string{"2024-03-10T12:00:00Z [svc] request arrived user=123"},
		},
		{
			name: "new lines appended",
			files: map[string]string{
				"append.log": "2024-03-10T12:00:00Z user=123 line1\n",
			},
			want: []string{"2024-03-10T12:00:00Z [append] line1 user=123", "2024-03-10T12:00:00Z [append] line2 user=123"},
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

			cfg := &Config{SearchValue: "123", Keys: []string{"user"}}
			lp := NewLineProcessor(cfg, NewTimeParser())
			fs := NewFileScanner(&FileScannerOpts{
				LineProcessor:  lp,
				FollowInterval: 10 * time.Millisecond,
				Out:            &out,
				Now:            time.Now(),
				LogFileReader:  transport.NewLogFileReader(nil),
				Diagnostics:    diagnostics.NewDiagnostics(),
			})

			ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
			defer cancel()

			// Optionally append new lines after start
			if tt.append != nil {
				go func() {
					time.Sleep(100 * time.Millisecond)
					tt.append()
				}()
			}

			fs.Follow(ctx, files, &testFormatter{})

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

func TestFileScanner_Scan_OpenErrorCreatesDiagnostic(t *testing.T) {
	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
	}

	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor: lp,
		Out:           io.Discard,
		Now:           time.Now(),
		Diagnostics:   diagnostics.NewDiagnostics(),
	})

	fs.logFileReader = &mockLogFileReader{
		openFn: func(ctx context.Context, path string) (io.ReadCloser, error) {
			return nil, errors.New("open failed")
		},
	}

	entries, err := fs.Scan(t.Context(), []string{"app.log"})
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}

	diags := fs.diagnostics.Entries()

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	want := "Error opening file app.log: open failed"

	if diags[0].Message != want {
		t.Fatalf("expected %q got %q", want, diags[0].Message)
	}
}

func TestFileScanner_ScanFromOffset_SizeErrorCreatesDiagnostic(t *testing.T) {
	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
	}

	lp := NewLineProcessor(cfg, NewTimeParser())

	fs := NewFileScanner(&FileScannerOpts{
		LineProcessor: lp,
		Out:           io.Discard,
		Now:           time.Now(),
		Diagnostics:   diagnostics.NewDiagnostics(),
	})

	fs.offsets["app.log"] = 0

	fs.logFileReader = &mockLogFileReader{
		sizeFn: func(ctx context.Context, path string) (int64, error) {
			return 0, errors.New("size failed")
		},
	}

	f := formatter.NewFormatter(&formatter.Opts{
		Output: config.OutputPretty,
	})

	fs.scanFromOffset(t.Context(), "app.log", f)

	diags := fs.diagnostics.Entries()

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	want := "Error reading file app.log: size failed"

	if diags[0].Message != want {
		t.Fatalf("expected %q got %q", want, diags[0].Message)
	}
}

func equalFields(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}

		if fmt.Sprint(av) != fmt.Sprint(bv) {
			return false
		}
	}

	return true
}
