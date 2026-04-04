package scanner

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type Scanner interface {
	Scan(sources []string) []domain.LogEntry
	Follow(sources []string)
	ListSources() ([]string, error)
}
