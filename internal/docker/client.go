package docker

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type DockerCLIClient struct{}

func NewDockerCLIClient() *DockerCLIClient {
	return &DockerCLIClient{}
}

func (c *DockerCLIClient) Logs(container string, follow bool, since string) (io.ReadCloser, error) {
	args := []string{"logs"}

	if follow {
		args = append(args, "--follow", "--tail", "0")
	}

	if since != "" {
		args = append(args, "--since", since)
	}

	args = append(args, container)

	cmd := exec.Command("docker", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Docker logs go to stderr by default
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &CMDReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

func (c *DockerCLIClient) ListContainers() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var containers []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			containers = append(containers, l)
		}
	}

	return containers, nil
}
