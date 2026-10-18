package scanner

import (
	"context"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type Scanner interface {
	Scan(ctx context.Context, sources []string) ([]domain.LogEntry, error)
	// SeekToEnd skips existing content so a subsequent Follow only emits
	// newly appended lines. It returns the sources that can be followed.
	SeekToEnd(ctx context.Context, sources []string) ([]string, error)
	// ServiceName returns the service label used for entries from source.
	ServiceName(source string) string
	Follow(ctx context.Context, sources []string, f formatter.LogFormatter)
	ListSources(ctx context.Context) ([]string, error)
}
