package scanner

import (
	"reflect"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func newEntry(ts int64, service string) *domain.LogEntry {
	return &domain.LogEntry{
		Timestamp: time.Unix(ts, 0),
		Service:   service,
		Message:   "msg",
	}
}

func timestamps(entries []domain.LogEntry) []int64 {
	out := make([]int64, 0, len(entries))

	for _, e := range entries {
		out = append(out, e.Timestamp.Unix())
	}

	return out
}

func TestNewEntryCollector(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		cfg       *ScanConfig
		expectErr bool
	}{
		{
			name: "valid without latest",
			cfg: &ScanConfig{
				Limit: 10,
			},
		},
		{
			name: "invalid since",
			cfg: &ScanConfig{
				Since: "invalid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEntryCollector(tt.cfg, now)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEntryCollector_Add_NoLimit(t *testing.T) {
	cfg := &ScanConfig{
		Limit: 0,
	}

	c, err := NewEntryCollector(cfg, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := []*domain.LogEntry{
		newEntry(1, "svc"),
		newEntry(2, "svc"),
		newEntry(3, "svc"),
	}

	for _, e := range entries {
		ok := c.Add(e)

		if !ok {
			t.Fatalf("expected continue=true")
		}
	}

	got := timestamps(c.Results())
	expected := []int64{1, 2, 3}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}

func TestEntryCollector_Add_DefaultLimit(t *testing.T) {
	cfg := &ScanConfig{
		Limit: 2,
	}

	c, err := NewEntryCollector(cfg, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c.StartSource()

	tests := []struct {
		entry          *domain.LogEntry
		expectContinue bool
	}{
		{
			entry:          newEntry(1, "svc"),
			expectContinue: true,
		},
		{
			entry:          newEntry(2, "svc"),
			expectContinue: false,
		},
		{
			entry:          newEntry(3, "svc"),
			expectContinue: false,
		},
	}

	for _, tt := range tests {
		got := c.Add(tt.entry)

		if got != tt.expectContinue {
			t.Fatalf("expected continue=%v, got %v",
				tt.expectContinue, got)
		}
	}

	got := timestamps(c.Results())
	expected := []int64{1, 2}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}

func TestEntryCollector_Add_Latest(t *testing.T) {
	cfg := &ScanConfig{
		Limit:  2,
		Latest: true,
	}

	c, err := NewEntryCollector(cfg, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := []*domain.LogEntry{
		newEntry(1, "svc"),
		newEntry(3, "svc"),
		newEntry(2, "svc"),
	}

	for _, e := range entries {
		ok := c.Add(e)

		if !ok {
			t.Fatalf("expected continue=true")
		}
	}

	got := timestamps(c.Results())
	expected := []int64{2, 3}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}

func TestEntryCollector_Add_SinceFiltering(t *testing.T) {
	now := time.Unix(100, 0)

	cfg := &ScanConfig{
		Since: "10s",
	}

	c, err := NewEntryCollector(cfg, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c.Add(newEntry(80, "svc")) // filtered out
	c.Add(newEntry(95, "svc")) // included

	got := timestamps(c.Results())
	expected := []int64{95}

	if len(got) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	if got[0] != expected[0] {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestEntryCollector_Add_NilEntry(t *testing.T) {
	c, err := NewEntryCollector(&ScanConfig{}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ok := c.Add(nil)

	if !ok {
		t.Fatalf("expected continue=true")
	}

	if len(c.Results()) != 0 {
		t.Fatalf("expected no results")
	}
}

func TestEntryCollector_Results_Sorted(t *testing.T) {
	c, err := NewEntryCollector(&ScanConfig{}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c.Add(newEntry(3, "svc"))
	c.Add(newEntry(1, "svc"))
	c.Add(newEntry(2, "svc"))

	got := timestamps(c.Results())
	expected := []int64{1, 2, 3}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}

func TestEntryCollector_Results_WithContextAndLimit(t *testing.T) {
	cfg := &ScanConfig{
		Limit: 2,
	}

	c, err := NewEntryCollector(cfg, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := []domain.LogEntry{
		{
			Timestamp: time.Unix(1, 0),
			Message:   "context-before-1",
			IsContext: true,
		},
		{
			Timestamp: time.Unix(2, 0),
			Message:   "match-1",
		},
		{
			Timestamp: time.Unix(3, 0),
			Message:   "context-between",
			IsContext: true,
		},
		{
			Timestamp: time.Unix(4, 0),
			Message:   "match-2",
		},
		{
			Timestamp: time.Unix(5, 0),
			Message:   "context-after-limit",
			IsContext: true,
		},
	}

	for _, e := range entries {
		if e.IsContext {
			c.AddContext(&e)
		} else {
			c.Add(&e)
		}
	}

	results := c.Results()

	got := make([]string, 0, len(results))

	for _, r := range results {
		got = append(got, r.Message)
	}

	expected := []string{
		"context-before-1",
		"match-1",
		"context-between",
		"match-2",
		"context-after-limit",
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestEntryCollector_AddContext(t *testing.T) {
	c, err := NewEntryCollector(&ScanConfig{}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := domain.LogEntry{
		Timestamp: time.Unix(1, 0),
		Message:   "context",
		IsContext: true,
	}

	c.AddContext(&entry)

	results := c.Results()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].IsContext {
		t.Fatalf("expected context entry")
	}

	if results[0].Message != "context" {
		t.Fatalf("unexpected message: %q", results[0].Message)
	}
}
