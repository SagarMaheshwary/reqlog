package transport

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestNewLogFileReader(t *testing.T) {
	t.Run("returns local LogFileReader when client is nil", func(t *testing.T) {
		r := NewLogFileReader(nil)

		if _, ok := r.(*LocalLogFileReader); !ok {
			t.Fatalf("expected *LocalLogFileReader got %T", r)
		}
	})

	t.Run("returns ssh LogFileReader when client provided", func(t *testing.T) {
		executor := &SSHExecutor{client: &ssh.Client{}}
		r := NewLogFileReader(executor)

		if _, ok := r.(*SSHLogFileReader); !ok {
			t.Fatalf("expected *SSHLogFileReader got %T", r)
		}
	})
}

func TestNewExecutor(t *testing.T) {
	t.Run("returns local executor when client is nil", func(t *testing.T) {
		exec := NewExecutor(nil)

		if _, ok := exec.(*LocalExecutor); !ok {
			t.Fatalf("expected *LocalExecutor got %T", exec)
		}
	})

	t.Run("returns ssh executor when client provided", func(t *testing.T) {
		client := &ssh.Client{}

		exec := NewExecutor(client)

		if _, ok := exec.(*SSHExecutor); !ok {
			t.Fatalf("expected *SSHExecutor got %T", exec)
		}
	})
}
