package diagnostics

import (
	"testing"
	"time"
)

func TestDiagnostics_LogEntries(t *testing.T) {
	tests := []struct {
		name      string
		logFn     func(*Diagnostics)
		wantLevel string
	}{
		{
			name: "error",
			logFn: func(d *Diagnostics) {
				d.Error("something failed", map[string]any{
					"path": "/tmp/app.log",
				})
			},
			wantLevel: "error",
		},
		{
			name: "warn",
			logFn: func(d *Diagnostics) {
				d.Warn("something suspicious", map[string]any{
					"path": "/tmp/app.log",
				})
			},
			wantLevel: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiagnostics()

			before := time.Now().UTC()

			tt.logFn(d)

			after := time.Now().UTC()

			entries := d.Entries()

			if len(entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(entries))
			}

			entry := entries[0]

			if entry.Service != service {
				t.Fatalf("expected service %q got %q", service, entry.Service)
			}

			if !entry.IsDiagnostics {
				t.Fatal("expected IsDiagnostics=true")
			}

			if entry.Fields["level"] != tt.wantLevel {
				t.Fatalf(
					"expected level %q got %v",
					tt.wantLevel,
					entry.Fields["level"],
				)
			}

			if entry.Fields["path"] != "/tmp/app.log" {
				t.Fatalf(
					"expected path field, got %v",
					entry.Fields["path"],
				)
			}

			if entry.Timestamp.Before(before) ||
				entry.Timestamp.After(after) {
				t.Fatalf(
					"timestamp %v outside expected range [%v, %v]",
					entry.Timestamp,
					before,
					after,
				)
			}
		})
	}
}

func TestDiagnostics_NilFields(t *testing.T) {
	tests := []struct {
		name      string
		logFn     func(*Diagnostics)
		wantLevel string
	}{
		{
			name: "error",
			logFn: func(d *Diagnostics) {
				d.Error("failed", nil)
			},
			wantLevel: "error",
		},
		{
			name: "warn",
			logFn: func(d *Diagnostics) {
				d.Warn("warning", nil)
			},
			wantLevel: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiagnostics()

			tt.logFn(d)

			entry := d.Entries()[0]

			if entry.Fields == nil {
				t.Fatal("fields should not be nil")
			}

			if entry.Fields["level"] != tt.wantLevel {
				t.Fatalf(
					"expected level %q got %v",
					tt.wantLevel,
					entry.Fields["level"],
				)
			}
		})
	}
}
