package formatter

import (
	"fmt"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func Format(entry domain.LogEntry, c *Colorizer) string {
	serviceColor := c.Color(entry.Service)

	return fmt.Sprintf(
		"%s%s%s%s %s[%s]%s | %s%s%s",
		dim, tsColor,
		entry.Timestamp.Format(time.RFC3339),
		reset,

		serviceColor,
		entry.Service,
		reset,

		msgColor,
		entry.Message,
		reset,
	)
}
