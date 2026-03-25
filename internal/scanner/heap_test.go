package scanner

import (
	"container/heap"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestEntryHeap_BasicMethods(t *testing.T) {
	now := time.Now()

	h := entryHeap{
		{Timestamp: now.Add(2 * time.Minute)},
		{Timestamp: now},
	}

	if h.Len() != 2 {
		t.Fatalf("expected len=2, got %d", h.Len())
	}

	if !h.Less(1, 0) {
		t.Fatalf("expected h[1] < h[0]")
	}

	h.Swap(0, 1)

	if !h[0].Timestamp.Equal(now) {
		t.Fatalf("swap failed")
	}
}

func TestEntryHeap_MinHeapOrdering(t *testing.T) {
	now := time.Now()

	entries := []domain.LogEntry{
		{Timestamp: now.Add(5 * time.Minute)},
		{Timestamp: now.Add(1 * time.Minute)},
		{Timestamp: now.Add(3 * time.Minute)},
		{Timestamp: now},
	}

	h := &entryHeap{}
	heap.Init(h)

	for _, e := range entries {
		heap.Push(h, e)
	}

	// Expect sorted order (oldest first)
	prev := heap.Pop(h).(domain.LogEntry)

	for h.Len() > 0 {
		curr := heap.Pop(h).(domain.LogEntry)

		if curr.Timestamp.Before(prev.Timestamp) {
			t.Fatalf("heap order violated")
		}

		prev = curr
	}
}

func TestEntryHeap_PushPop(t *testing.T) {
	now := time.Now()

	h := &entryHeap{}
	heap.Init(h)

	entry := domain.LogEntry{Timestamp: now}

	heap.Push(h, entry)

	if h.Len() != 1 {
		t.Fatalf("expected len=1, got %d", h.Len())
	}

	popped := heap.Pop(h).(domain.LogEntry)

	if !popped.Timestamp.Equal(now) {
		t.Fatalf("unexpected popped value")
	}

	if h.Len() != 0 {
		t.Fatalf("expected empty heap")
	}
}

func TestEntryHeap_MultiplePushPop(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		offsets []time.Duration
	}{
		{
			name:    "random order",
			offsets: []time.Duration{5, 1, 3, 2, 4},
		},
		{
			name:    "already sorted",
			offsets: []time.Duration{1, 2, 3, 4, 5},
		},
		{
			name:    "reverse order",
			offsets: []time.Duration{5, 4, 3, 2, 1},
		},
		{
			name:    "duplicates",
			offsets: []time.Duration{1, 1, 2, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &entryHeap{}
			heap.Init(h)

			for _, d := range tt.offsets {
				heap.Push(h, domain.LogEntry{
					Timestamp: now.Add(d * time.Minute),
				})
			}

			var prev time.Time

			for h.Len() > 0 {
				curr := heap.Pop(h).(domain.LogEntry)

				if !prev.IsZero() && curr.Timestamp.Before(prev) {
					t.Fatalf("heap order violated")
				}

				prev = curr.Timestamp
			}
		})
	}
}

func TestEntryHeap_PopEmptyPanics(t *testing.T) {
	h := &entryHeap{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when popping empty heap")
		}
	}()

	heap.Pop(h)
}
