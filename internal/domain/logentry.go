package domain

import "time"

type LogEntry struct {
	Timestamp time.Time
	Service   string
	Message   string
}
