package transport

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIsLogFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"log file", "app.log", true},
		{"text file", "app.txt", false},
		{"no extension", "app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLogFile(tt.file)

			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestBuildServiceMatcher(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		file     string
		want     bool
	}{
		{"no filters matches all", nil, "api.log", true},
		{"exact match + empty service", []string{"api", ""}, "api.log", true},
		{"exact no match", []string{"db"}, "api.log", false},
		{"prefix wildcard", []string{"api-*"}, "api-prod.log", true},
		{"prefix wildcard no match", []string{"api-*"}, "worker.log", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := buildServiceMatcher(tt.services)

			got := match(tt.file)

			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestLocalFileSystem_Open(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &LocalFileSystem{}

	f, err := fs.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()
}

func TestLocalFileSystem_ListFiles_Flat(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"api.log",
		"db.log",
		"notes.txt",
	}

	for _, name := range files {
		path := filepath.Join(dir, name)

		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// create nested directory which should be ignored by listFlat
	subDir := filepath.Join(dir, "archive")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(subDir, "api.log"),
		[]byte("nested"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	fs := &LocalFileSystem{}

	got, err := fs.ListFiles(t.Context(), dir, ListOptions{
		Services: []string{"api"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(dir, "api.log"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestLocalFileSystem_ListFiles_Recursive(t *testing.T) {
	dir := t.TempDir()

	nested := filepath.Join(dir, "services")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(nested, "api.log"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(nested, "db.log"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(nested, "notes.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &LocalFileSystem{}

	got, err := fs.ListFiles(t.Context(), dir, ListOptions{
		Recursive: true,
		Services:  []string{"api"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(nested, "api.log"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestLocalFileSystem_ListFiles_Recursive_OnError(t *testing.T) {
	dir := t.TempDir()

	protectedDir := filepath.Join(dir, "restricted")
	if err := os.Mkdir(protectedDir, 0o000); err != nil {
		t.Fatal(err)
	}

	// restore permissions so temp cleanup can remove it
	defer os.Chmod(protectedDir, 0o755)

	var (
		called  bool
		errPath string
		gotErr  error
	)

	fs := &LocalFileSystem{}

	files, err := fs.ListFiles(t.Context(), dir, ListOptions{
		Recursive: true,
		OnError: func(path string, err error) {
			called = true
			errPath = path
			gotErr = err
		},
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if files == nil {
		files = []string{}
	}

	if !called {
		t.Fatal("expected OnError to be called")
	}

	if errPath != protectedDir {
		t.Fatalf("expected path %q, got %q", protectedDir, errPath)
	}

	if gotErr == nil {
		t.Fatal("expected error passed to OnError")
	}
}

func TestBuildServiceMatcher_MixedExactAndWildcard(t *testing.T) {
	match := buildServiceMatcher([]string{"api-*", "worker"})

	tests := []struct {
		name string
		file string
		want bool
	}{
		{
			name: "matches wildcard prefix",
			file: "api-prod.log",
			want: true,
		},
		{
			name: "matches exact service",
			file: "worker.log",
			want: true,
		},
		{
			name: "does not match non matching service",
			file: "db.log",
			want: false,
		},
		{
			name: "does not match partial exact service",
			file: "worker-prod.log",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := match(tt.file)

			if got != tt.want {
				t.Fatalf("match(%q): expected %v got %v", tt.file, tt.want, got)
			}
		})
	}
}
