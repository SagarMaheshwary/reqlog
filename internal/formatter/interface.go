package formatter

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type LogFormatter interface {
	Format(entry domain.LogEntry) string
}
