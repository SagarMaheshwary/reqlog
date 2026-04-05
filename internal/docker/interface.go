package docker

import "io"

type CLIDockerClient interface {
	Logs(container string, follow bool, since string) (io.ReadCloser, error)
	ListContainers() ([]string, error)
}
