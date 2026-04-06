package docker

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func fakeExecCommand(output string, err error) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
	cmd.Env = []string{
		"GO_WANT_HELPER_PROCESS=1",
		"OUTPUT=" + output,
		"ERR=" + fmt.Sprint(err != nil),
	}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if os.Getenv("ERR") == "true" {
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, os.Getenv("OUTPUT"))
	os.Exit(0)
}

func TestDockerCLIClient_ListContainers(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		err       error
		want      []string
		expectErr bool
	}{
		{
			name:   "multiple containers",
			output: "auth\nsvc\nworker\n",
			want:   []string{"auth", "svc", "worker"},
		},
		{
			name:   "empty lines trimmed",
			output: "\nauth\n\nsvc\n",
			want:   []string{"auth", "svc"},
		},
		{
			name:      "command error",
			err:       fmt.Errorf("docker not found"),
			expectErr: true,
		},
	}

	orig := execCommand
	defer func() { execCommand = orig }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = func(name string, args ...string) *exec.Cmd {
				return fakeExecCommand(tt.output, tt.err)
			}

			c := NewDockerCLIClient()

			res, err := c.ListContainers()

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, res)
			}
		})
	}
}

func TestDockerCLIClient_Logs_Args(t *testing.T) {
	tests := []struct {
		name     string
		follow   bool
		since    string
		expected []string
	}{
		{
			name:   "basic logs",
			follow: false,
			since:  "",
			expected: []string{
				"logs", "auth",
			},
		},
		{
			name:   "follow enabled",
			follow: true,
			since:  "",
			expected: []string{
				"logs", "--follow", "--tail", "0", "auth",
			},
		},
		{
			name:   "since provided",
			follow: false,
			since:  "5m",
			expected: []string{
				"logs", "--since", "5m", "auth",
			},
		},
		{
			name:   "follow + since",
			follow: true,
			since:  "5m",
			expected: []string{
				"logs", "--follow", "--tail", "0", "--since", "5m", "auth",
			},
		},
	}

	orig := execCommand
	defer func() { execCommand = orig }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedArgs []string

			execCommand = func(name string, args ...string) *exec.Cmd {
				capturedArgs = args
				return fakeExecCommand("log line\n", nil)
			}

			c := NewDockerCLIClient()

			rc, err := c.Logs("auth", tt.follow, tt.since)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer rc.Close()

			if !reflect.DeepEqual(capturedArgs, tt.expected) {
				t.Fatalf("expected args %v, got %v", tt.expected, capturedArgs)
			}
		})
	}
}
