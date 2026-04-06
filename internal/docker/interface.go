package docker

import "io"

type DockerClient interface {
	Logs(container string, follow bool, since string) (io.ReadCloser, error)
	ListContainers() ([]string, error)
}
