package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type TextParser struct{}

func (p TextParser) Parse(line string, service string) (domain.LogEntry, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return domain.LogEntry{}, fmt.Errorf("invalid log format")
	}

	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return domain.LogEntry{}, err
	}

	message := strings.Join(parts[1:], " ")

	return domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   message,
	}, nil
}

func (p TextParser) ExtractField(line string, keys []string) (string, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", false
	}

	for i := 1; i < len(parts); i++ {
		part := parts[i]

		if !strings.Contains(part, "=") {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		for _, key := range keys {
			if kv[0] == key {
				return stripQuotes(kv[1]), true
			}
		}
	}

	return "", false
}

func stripQuotes(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return val
}
