package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type Formatter struct {
	colorizer    *Colorizer
	serviceWidth int
}

func NewFormatter(entries []domain.LogEntry) *Formatter {
	max := 0
	for _, e := range entries {
		if len(e.Service) > max {
			max = len(e.Service)
		}
	}

	return &Formatter{
		colorizer:    NewColorizer(),
		serviceWidth: max,
	}
}

func (f *Formatter) padAfter(service string) string {
	if len(service) >= f.serviceWidth {
		return ""
	}
	return strings.Repeat(" ", f.serviceWidth-len(service))
}

func (f *Formatter) Format(entry domain.LogEntry) string {
	serviceColor := f.colorizer.Color(entry.Service)
	padding := f.padAfter(entry.Service)

	return fmt.Sprintf(
		"%s%s%s%s %s[%s]%s%s | %s%s%s",
		dim, tsColor,
		entry.Timestamp.Format(time.RFC3339),
		reset,

		serviceColor,
		entry.Service,
		reset,
		padding,

		msgColor,
		entry.Message,
		reset,
	)
}
