package scanner

import (
	"context"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type Scanner interface {
	Scan(sources []string) []domain.LogEntry
	Follow(ctx context.Context, sources []string, f formatter.LogFormatter)
	ListSources() ([]string, error)
}
