package docker

import (
	"fmt"
	"io"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type DockerCLIClient struct {
	exec transport.Executor
}

func NewDockerCLIClient(exec transport.Executor) *DockerCLIClient {
	return &DockerCLIClient{
		exec: exec,
	}
}

func (c *DockerCLIClient) Logs(
	container string,
	follow bool,
	since string,
) (io.ReadCloser, error) {
	args := []string{"logs"}

	if since != "" {
		args = append(args, "--since", since)
	}

	if follow {
		args = append(args, "--follow", "--tail", "0")
	}

	args = append(args, container)

	return c.exec.Run("docker", args...)
}

func (c *DockerCLIClient) ListContainers() ([]string, error) {
	out, err := c.exec.Output("docker", "ps", "--format", "{{.Names}}")
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
