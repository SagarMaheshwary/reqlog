package formatter

import (
	"fmt"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

const (
	colorReset = "\033[0m"
	colorBlue  = "\033[34m"
	colorGray  = "\033[90m"
)

func Print(requestID string, entries []domain.LogEntry, limit int) {
	fmt.Printf("\nRequest: %s\n\n", requestID)
	// fmt.Printf("%-12s %-20s %s\n", "TIME", "SERVICE", "MESSAGE")

	count := 0

	for _, e := range entries {
		if limit > 0 && count >= limit {
			break
		}

		fmt.Printf(
			"%s[%s]%s %s%s%s → %s\n",
			colorBlue,
			e.Service,
			colorReset,
			colorGray,
			e.Timestamp.Format("2006-01-02 15:04:05"),
			colorReset,
			e.Message,
		)

		count++
	}
}
