package docker

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

type mockExecutor struct {
	lastCmd  string
	lastArgs []string

	runFunc    func(cmd string, args ...string) (io.ReadCloser, error)
	outputFunc func(cmd string, args ...string) ([]byte, error)
}

func (m *mockExecutor) Run(cmd string, args ...string) (io.ReadCloser, error) {
	m.lastCmd = cmd
	m.lastArgs = args

	if m.runFunc != nil {
		return m.runFunc(cmd, args...)
	}
	return io.NopCloser(bytes.NewBufferString("mock")), nil
}

func (m *mockExecutor) RunCombined(cmd string, args ...string) (io.ReadCloser, error) {
	return m.Run(cmd, args...)
}

func (m *mockExecutor) Output(cmd string, args ...string) ([]byte, error) {
	m.lastCmd = cmd
	m.lastArgs = args

	if m.outputFunc != nil {
		return m.outputFunc(cmd, args...)
	}
	return []byte("c1\nc2\n"), nil
}

func TestDockerCLIClient_Logs(t *testing.T) {
	tests := []struct {
		name      string
		container string
		follow    bool
		since     string
		wantArgs  []string
	}{
		{
			name:      "basic logs",
			container: "api",
			follow:    false,
			since:     "",
			wantArgs:  []string{"logs", "api"},
		},
		{
			name:      "with since",
			container: "api",
			follow:    false,
			since:     "1h",
			wantArgs:  []string{"logs", "--since", "1h", "api"},
		},
		{
			name:      "follow enabled",
			container: "api",
			follow:    true,
			since:     "",
			wantArgs:  []string{"logs", "--follow", "--tail", "0", "api"},
		},
		{
			name:      "follow + since",
			container: "api",
			follow:    true,
			since:     "10m",
			wantArgs:  []string{"logs", "--since", "10m", "--follow", "--tail", "0", "api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{}

			client := NewDockerCLIClient(exec)

			_, err := client.Logs(tt.container, tt.follow, tt.since)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exec.lastCmd != "docker" {
				t.Fatalf("expected docker, got %s", exec.lastCmd)
			}

			if !reflect.DeepEqual(exec.lastArgs, tt.wantArgs) {
				t.Fatalf("args mismatch\nwant=%v\ngot=%v", tt.wantArgs, exec.lastArgs)
			}
		})
	}
}

func TestDockerCLIClient_ListContainers(t *testing.T) {
	t.Run("parses output", func(t *testing.T) {
		exec := &mockExecutor{
			outputFunc: func(cmd string, args ...string) ([]byte, error) {
				return []byte("api\nworker\n\n"), nil
			},
		}

		client := NewDockerCLIClient(exec)

		got, err := client.ListContainers()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []string{"api", "worker"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected %v got %v", want, got)
		}
	})

	t.Run("trims spaces and ignores empty lines", func(t *testing.T) {
		exec := &mockExecutor{
			outputFunc: func(cmd string, args ...string) ([]byte, error) {
				return []byte(" api \n  worker  \n\n"), nil
			},
		}

		client := NewDockerCLIClient(exec)

		got, err := client.ListContainers()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []string{"api", "worker"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected %v got %v", want, got)
		}
	})

	t.Run("handles empty output", func(t *testing.T) {
		exec := &mockExecutor{
			outputFunc: func(cmd string, args ...string) ([]byte, error) {
				return []byte(""), nil
			},
		}

		client := NewDockerCLIClient(exec)

		got, err := client.ListContainers()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 0 {
			t.Fatalf("expected empty slice got %v", got)
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		exec := &mockExecutor{
			outputFunc: func(cmd string, args ...string) ([]byte, error) {
				return nil, io.ErrUnexpectedEOF
			},
		}

		client := NewDockerCLIClient(exec)

		_, err := client.ListContainers()
		if err == nil {
			t.Fatalf("expected error but got nil")
		}
	})
}
