package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type TextParser struct{}

func (p TextParser) Parse(line string, service string) (domain.LogEntry, map[string]string, error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return domain.LogEntry{}, nil, fmt.Errorf("invalid log format")
	}

	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return domain.LogEntry{}, nil, err
	}

	fields := extractFields(line)

	return domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   parts[2],
	}, fields, nil
}

func extractFields(line string) map[string]string {
	fields := make(map[string]string)

	parts := strings.SplitSeq(line, " ")
	for part := range parts {
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			fields[kv[0]] = kv[1]
		}
	}

	return fields
}
