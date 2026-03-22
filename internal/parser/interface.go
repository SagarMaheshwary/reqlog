package parser

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type LogParser interface {
	Parse(line string, service string) (domain.LogEntry, error)
	ExtractField(line string, keys []string) (string, bool)
}
