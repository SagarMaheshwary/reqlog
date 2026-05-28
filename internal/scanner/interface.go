package scanner

import (
	"context"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type Scanner interface {
	Scan(ctx context.Context, sources []string) ([]domain.LogEntry, error)
	Follow(ctx context.Context, sources []string, f formatter.LogFormatter)
	ListSources(ctx context.Context) ([]string, error)
}
