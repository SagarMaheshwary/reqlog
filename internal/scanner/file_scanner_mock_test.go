package scanner

import (
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type mockParser struct {
	extractOK bool
	parseErr  error
}

func (m mockParser) ExtractField(line string, keys []string) (string, bool) {
	if !m.extractOK {
		return "", false
	}
	return "123", true
}

func (m mockParser) Parse(line, service string) (domain.LogEntry, error) {
	if m.parseErr != nil {
		return domain.LogEntry{}, m.parseErr
	}
	return domain.LogEntry{
		Timestamp: time.Now(),
		Service:   service,
		Message:   line,
	}, nil
}
