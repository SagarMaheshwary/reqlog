package domain

import "time"

type LogEntry struct {
	Timestamp time.Time
	Service   string
	Message   string
	IsContext bool
	Fields    map[string]any
	Raw       string
}
