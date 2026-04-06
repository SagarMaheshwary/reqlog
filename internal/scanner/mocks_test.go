package scanner

import (
	"fmt"
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

type mockParser struct {
	extractOK      bool
	parseErr       bool
	extractedValue string
}

func (m mockParser) ExtractField(line string, keys []string) (string, bool) {
	if !m.extractOK {
		return "", false
	}
	if m.extractedValue != "" {
		return m.extractedValue, true
	}
	return "123", true
}

func (m mockParser) Parse(line, service string) (domain.LogEntry, error) {
	if m.parseErr {
		return domain.LogEntry{}, fmt.Errorf("parse error")
	}
	return domain.LogEntry{
		Timestamp: time.Now(),
		Service:   service,
		Message:   line,
	}, nil
}
