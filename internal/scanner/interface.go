package scanner

import (
	"io"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type Scanner interface {
	Scan(sources []string) []domain.LogEntry
	Follow(sources []string)
	ListSources() ([]string, error)
}

type CLIDockerClient interface {
	Logs(container string, follow bool, since string) (io.ReadCloser, error)
	ListContainers() ([]string, error)
}
