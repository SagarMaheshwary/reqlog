package transport

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLocalLogFileReader_Open(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := &LocalLogFileReader{}

	f, err := fs.Open(t.Context(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()
}

func TestLocalLogFileReader_OpenFromOffset(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "test.log")

	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &LocalLogFileReader{}

	rc, err := r.OpenFromOffset(t.Context(), path, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if string(data) != "world" {
		t.Fatalf("expected %q got %q", "world", string(data))
	}
}

func TestLocalLogFileReader_OpenFromOffset_FileNotFound(t *testing.T) {
	r := &LocalLogFileReader{}

	_, err := r.OpenFromOffset(
		t.Context(),
		filepath.Join(t.TempDir(), "missing.log"),
		0,
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalLogFileReader_Size(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "test.log")

	content := []byte("hello world")

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := &LocalLogFileReader{}

	size, err := r.Size(t.Context(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if size != int64(len(content)) {
		t.Fatalf("expected size %d got %d", len(content), size)
	}
}

func TestLocalLogFileReader_Size_FileNotFound(t *testing.T) {
	r := &LocalLogFileReader{}

	_, err := r.Size(
		t.Context(),
		filepath.Join(t.TempDir(), "missing.log"),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalLogFileReader_ListFiles_Flat(t *testing.T) {
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

	fs := &LocalLogFileReader{}

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

func TestLocalLogFileReader_ListFiles_Recursive(t *testing.T) {
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

	fs := &LocalLogFileReader{}

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

func TestLocalLogFileReader_ListFiles_Recursive_OnError(t *testing.T) {
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

	fs := &LocalLogFileReader{}

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
