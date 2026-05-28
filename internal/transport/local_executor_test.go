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

func TestLocalExecutor_RunCombined(t *testing.T) {
	exec := &LocalExecutor{}

	rc1, err := exec.Run("printf", "a")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	defer rc1.Close()

	rc2, err := exec.RunCombined("printf", "a")
	if err != nil {
		t.Fatalf("RunCombined failed: %v", err)
	}
	defer rc2.Close()

	b1 := make([]byte, 10)
	b2 := make([]byte, 10)

	n1, _ := rc1.Read(b1)
	n2, _ := rc2.Read(b2)

	if string(b1[:n1]) != string(b2[:n2]) {
		t.Fatalf("Run and RunCombined differ")
	}
}

func TestLocalExecutor_InvalidCommand(t *testing.T) {
	exec := &LocalExecutor{}

	_, err := exec.Run("nonexistent_command_xyz")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
