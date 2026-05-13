package scanner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func rbEntry(id int) domain.LogEntry {
	return domain.LogEntry{
		Message: fmt.Sprintf("msg-%d", id),
	}
}

func rbMessages(entries []domain.LogEntry) []string {
	out := make([]string, 0, len(entries))

	for _, e := range entries {
		out = append(out, e.Message)
	}

	return out
}

func TestRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		pushes   []int
		expected []string
	}{
		{
			name:     "zero capacity",
			capacity: 0,
			pushes:   []int{1, 2, 3},
			expected: []string{},
		},
		{
			name:     "single entry",
			capacity: 3,
			pushes:   []int{1},
			expected: []string{"msg-1"},
		},
		{
			name:     "under capacity",
			capacity: 3,
			pushes:   []int{1, 2},
			expected: []string{"msg-1", "msg-2"},
		},
		{
			name:     "exact capacity",
			capacity: 3,
			pushes:   []int{1, 2, 3},
			expected: []string{"msg-1", "msg-2", "msg-3"},
		},
		{
			name:     "overwrite oldest once full",
			capacity: 3,
			pushes:   []int{1, 2, 3, 4},
			expected: []string{"msg-2", "msg-3", "msg-4"},
		},
		{
			name:     "multiple wraparounds",
			capacity: 3,
			pushes:   []int{1, 2, 3, 4, 5, 6},
			expected: []string{"msg-4", "msg-5", "msg-6"},
		},
		{
			name:     "capacity one",
			capacity: 1,
			pushes:   []int{1, 2, 3},
			expected: []string{"msg-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRingBuffer(tt.capacity)

			for _, id := range tt.pushes {
				rb.Push(rbEntry(id))
			}

			got := rbMessages(rb.Drain())

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestRingBuffer_DrainResetsBuffer(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Push(rbEntry(1))
	rb.Push(rbEntry(2))

	first := rbMessages(rb.Drain())

	expectedFirst := []string{"msg-1", "msg-2"}

	if !reflect.DeepEqual(first, expectedFirst) {
		t.Fatalf("expected %v, got %v", expectedFirst, first)
	}

	second := rb.Drain()

	if second != nil {
		t.Fatalf("expected nil after draining empty buffer, got %v", second)
	}
}

func TestRingBuffer_MultipleDrainCycles(t *testing.T) {
	rb := NewRingBuffer(2)

	rb.Push(rbEntry(1))
	rb.Push(rbEntry(2))

	got1 := rbMessages(rb.Drain())

	expected1 := []string{"msg-1", "msg-2"}

	if !reflect.DeepEqual(got1, expected1) {
		t.Fatalf("expected %v, got %v", expected1, got1)
	}

	rb.Push(rbEntry(3))

	got2 := rbMessages(rb.Drain())

	expected2 := []string{"msg-3"}

	if !reflect.DeepEqual(got2, expected2) {
		t.Fatalf("expected %v, got %v", expected2, got2)
	}
}
