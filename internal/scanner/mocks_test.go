package scanner

import (
	"io"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type mockDockerClient struct {
	logsFn func(container string, follow bool, since string) (io.ReadCloser, error)
	listFn func() ([]string, error)
}

func (m *mockDockerClient) Logs(container string, follow bool, since string) (io.ReadCloser, error) {
	return m.logsFn(container, follow, since)
}

func (m *mockDockerClient) ListContainers() ([]string, error) {
	return m.listFn()
}

type mockFormatter struct{}

func (f *mockFormatter) Format(entry domain.LogEntry) string {
	return entry.Timestamp.Format(time.RFC3339) + " [" + entry.Service + "] " + entry.Message
}
