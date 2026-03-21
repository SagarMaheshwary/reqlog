package formatter

import (
	"fmt"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func Print(entries []domain.LogEntry, limit int) {
	count := 0
	colorizer := NewColorizer()

	for _, e := range entries {
		if limit > 0 && count >= limit {
			break
		}

		fmt.Println(Format(e, colorizer))

		count++
	}
}

func Format(entry domain.LogEntry, c *Colorizer) string {
	serviceColor := c.Color(entry.Service)

	return fmt.Sprintf(
		"%s%s[%s]%s %s%s%s | %s%s%s",
		dim, tsColor,
		entry.Timestamp.Format("15:04:05"),
		reset,

		serviceColor,
		entry.Service,
		reset,

		msgColor,
		entry.Message,
		reset,
	)
}
