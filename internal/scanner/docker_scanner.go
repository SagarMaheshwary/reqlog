package scanner

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type DockerScanner struct {
	lineProcessor *LineProcessor
}

func NewDockerScanner(lp *LineProcessor) *DockerScanner {
	return &DockerScanner{lineProcessor: lp}
}

func (ds *DockerScanner) Scan(containers []string) []domain.LogEntry {
	return nil
}

func (ds *DockerScanner) Follow(containers []string) {
	//
}

func (ds *DockerScanner) ListSources() ([]string, error) {
	return nil, nil
}
