package docker

import (
	"io"
	"os/exec"
	"testing"
)

func TestCMDReadCloser_Close(t *testing.T) {
	cmd := exec.Command("echo", "test")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	rc := &CMDReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}

	// drain stdout to avoid broken pipe
	_, _ = io.ReadAll(rc)

	if err := rc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
