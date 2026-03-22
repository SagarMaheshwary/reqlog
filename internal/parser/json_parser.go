package parser

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/tidwall/gjson"
)

type JSONParser struct{}

func (p JSONParser) ExtractField(line string, keys []string) (string, bool) {
	for _, key := range keys {
		result := gjson.Get(line, key)
		if result.Exists() {
			return result.String(), true
		}
	}
	return "", false
}

func (p JSONParser) Parse(line string, service string) (domain.LogEntry, error) {
	ts, ok := extractJSONTimestamp(line)
	if !ok {
		return domain.LogEntry{}, fmt.Errorf("invalid timestamp")
	}

	return domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   buildJSONMessage(line),
	}, nil
}

func extractJSONTimestamp(line string) (time.Time, bool) {
	for _, key := range []string{"time", "timestamp", "ts"} {
		value := gjson.Get(line, key)
		if !value.Exists() {
			continue
		}

		ts, err := time.Parse(time.RFC3339, value.String())
		if err == nil {
			return ts, true
		}
	}

	return time.Time{}, false
}

func buildJSONMessage(line string) string {
	result := gjson.Parse(line)
	if !result.IsObject() {
		return line
	}

	parts := make([]string, 0, 8)

	result.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		if isTimestampKey(k) {
			return true
		}

		parts = append(parts, formatJSONField(k, value))
		return true
	})

	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func isTimestampKey(key string) bool {
	switch key {
	case "time", "timestamp", "ts":
		return true
	default:
		return false
	}
}

func formatJSONField(key string, value gjson.Result) string {
	switch value.Type {
	case gjson.String:
		return key + "=" + value.String()
	default:
		return key + "=" + value.Raw
	}
}
