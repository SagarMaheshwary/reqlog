package scanner

import "github.com/sagarmaheshwary/reqlog/internal/domain"

type entryHeap []domain.LogEntry

func (h entryHeap) Len() int { return len(h) }

func (h entryHeap) Less(i, j int) bool {
	// min-heap by timestamp: oldest entry at top
	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h entryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *entryHeap) Push(x any) {
	*h = append(*h, x.(domain.LogEntry))
}

func (h *entryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
