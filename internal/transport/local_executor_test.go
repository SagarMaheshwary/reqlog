package transport

import (
	"testing"
)

func TestLocalExecutor_Output(t *testing.T) {
	exec := &LocalExecutor{}

	out, err := exec.Output("printf", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(out)

	// echo adds newline
	want := "hello"

	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}

func TestLocalExecutor_Run(t *testing.T) {
	exec := &LocalExecutor{}

	rc, err := exec.Run("printf", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	buf := make([]byte, 64)

	n, err := rc.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("unexpected read error: %v", err)
	}

	got := string(buf[:n])

	want := "hello"

	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}

func TestLocalExecutor_InvalidCommand(t *testing.T) {
	exec := &LocalExecutor{}

	_, err := exec.Run("nonexistent_command_xyz")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
