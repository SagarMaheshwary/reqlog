package scanner

import (
	"container/heap"
	"sort"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type EntryCollector struct {
	cfg       *ScanConfig
	sinceTime time.Time
	results   []domain.LogEntry
	heap      entryHeap
	//per-source count for default --limit mode
	sourceCount int
}

func NewEntryCollector(cfg *ScanConfig, now time.Time) (*EntryCollector, error) {
	sinceTime, err := parseSince(cfg.Since, now)
	if err != nil {
		return nil, err
	}

	c := &EntryCollector{
		cfg:       cfg,
		sinceTime: sinceTime,
	}

	if cfg.Latest {
		heap.Init(&c.heap)
	}

	return c, nil
}

func (c *EntryCollector) StartSource() {
	c.sourceCount = 0
}

func (c *EntryCollector) Add(entry *domain.LogEntry) bool {
	if entry == nil {
		return true
	}

	if !passesSince(entry, c.sinceTime) {
		return true
	}

	// no limit
	if c.cfg.Limit == 0 {
		c.results = append(c.results, *entry)
		return true
	}

	// --latest => global heap, scan everything
	if c.cfg.Latest {
		if c.heap.Len() < c.cfg.Limit {
			heap.Push(&c.heap, *entry)
		} else if entry.Timestamp.After(c.heap[0].Timestamp) {
			heap.Pop(&c.heap)
			heap.Push(&c.heap, *entry)
		}

		return true
	}

	//default --limit => early exit per source
	if c.sourceCount < c.cfg.Limit {
		c.results = append(c.results, *entry)
		c.sourceCount++
	}

	//stop scanning this source once enough matches found
	return c.sourceCount < c.cfg.Limit
}

func (c *EntryCollector) Results() []domain.LogEntry {
	if c.cfg.Limit > 0 && c.cfg.Latest {
		c.results = drainHeap(&c.heap)
	}

	sort.Slice(c.results, func(i, j int) bool {
		return c.results[i].Timestamp.Before(c.results[j].Timestamp)
	})

	if c.cfg.Limit > 0 && !c.cfg.Latest && len(c.results) > c.cfg.Limit {
		c.results = c.results[len(c.results)-c.cfg.Limit:]
	}

	return c.results
}
