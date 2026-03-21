package parser

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type JSONLog struct {
	Time          string `json:"time"`
	Timestamp     string `json:"timestamp"`
	TS            string `json:"ts"`
	RequestID     string `json:"request_id"`
	ReqID         string `json:"req_id"`
	TraceID       string `json:"trace_id"`
	CorrelationID string `json:"correlation_id"`
	Message       string `json:"message"`
	Msg           string `json:"msg"`
}

type JSONParser struct{}

func (p JSONParser) Parse(line string, service string) (domain.LogEntry, map[string]string, error) {
	var log JSONLog

	if err := json.Unmarshal([]byte(line), &log); err != nil {
		return domain.LogEntry{}, nil, err
	}

	ts, ok := log.GetTimestamp()
	if !ok {
		return domain.LogEntry{}, nil, fmt.Errorf("invalid timestamp")
	}

	fields := log.ToFields()

	return domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   log.GetMessage(),
	}, fields, nil
}

func (l JSONLog) ToFields() map[string]string {
	return map[string]string{
		"request_id":     l.RequestID,
		"req_id":         l.ReqID,
		"trace_id":       l.TraceID,
		"correlation_id": l.CorrelationID,
	}
}

func (l JSONLog) GetTimestamp() (time.Time, bool) {
	candidates := []string{
		l.Time,
		l.Timestamp,
		l.TS,
	}

	for _, value := range candidates {
		if value == "" {
			continue
		}

		ts, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return ts, true
		}
	}

	return time.Time{}, false
}

func (l JSONLog) GetMessage() string {
	if l.Message != "" {
		return l.Message
	}
	return l.Msg
}
