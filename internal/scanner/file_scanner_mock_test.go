package scanner

import (
	"fmt"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

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
