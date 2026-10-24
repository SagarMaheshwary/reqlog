package diagnostics

import (
	"sync"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

const service = "reqlog"

type Diagnostics struct {
	mu    sync.Mutex
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

	d.add(msg, fields)
}

func (d *Diagnostics) Warn(msg string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any, 1)
	}
	fields["level"] = "warn"

	d.add(msg, fields)
}

func (d *Diagnostics) Entries() []domain.LogEntry {
	d.mu.Lock()
	defer d.mu.Unlock()

	return append([]domain.LogEntry(nil), d.items...)
}

func (d *Diagnostics) add(msg string, fields map[string]any) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.items = append(d.items, domain.LogEntry{
		Service:       service,
		Message:       msg,
		Fields:        fields,
		Timestamp:     time.Now().UTC(),
		IsDiagnostics: true,
	})
}
