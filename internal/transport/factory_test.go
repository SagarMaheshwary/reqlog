package transport

import (
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func TestNewFileSystem(t *testing.T) {
	t.Run("returns local filesystem when client is nil", func(t *testing.T) {
		fs := NewFileSystem(nil)

		if _, ok := fs.(*LocalFileSystem); !ok {
			t.Fatalf("expected *LocalFileSystem got %T", fs)
		}
	})

	t.Run("returns ssh filesystem when client provided", func(t *testing.T) {
		client := &sftp.Client{}

		fs := NewFileSystem(client)

		if _, ok := fs.(*SSHFileSystem); !ok {
			t.Fatalf("expected *SSHFileSystem got %T", fs)
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
