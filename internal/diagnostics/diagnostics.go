package diagnostics

import (
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

const service = "reqlog"

type Diagnostics struct {
	items []domain.LogEntry
}

func NewDiagnostics() *Diagnostics {
	return &Diagnostics{
		items: make([]domain.LogEntry, 0),
	}
}

func (d *Diagnostics) Error(msg string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any, 1)
	}
	fields["level"] = "error"

	d.items = append(d.items, domain.LogEntry{
		Service:       service,
		Message:       msg,
		Fields:        fields,
		Timestamp:     time.Now().UTC(),
		IsDiagnostics: true,
	})
}

func (d *Diagnostics) Warn(msg string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any, 1)
	}
	fields["level"] = "warn"

	d.items = append(d.items, domain.LogEntry{
		Service:       service,
		Message:       msg,
		Fields:        fields,
		Timestamp:     time.Now().UTC(),
		IsDiagnostics: true,
	})
}

func (d *Diagnostics) Entries() []domain.LogEntry {
	return d.items
}
