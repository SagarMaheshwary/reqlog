package scanner

import (
	"container/heap"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

type LineProcessor struct {
	config *ScanConfig
	parser parser.LogParser
}

func NewLineProcessor(cfg *ScanConfig, p parser.LogParser) *LineProcessor {
	return &LineProcessor{config: cfg, parser: p}
}

func (lp *LineProcessor) ProcessLine(line, service string) (*domain.LogEntry, bool) {
	// fast pre-filter (return if searchValue is not present in the line string)
	if lp.config.IgnoreCase {
		if !containsFoldASCII(line, lp.config.SearchValue) {
			return nil, false
		}
	} else {
		if !strings.Contains(line, lp.config.SearchValue) {
			return nil, false
		}
	}

	line = strings.TrimRight(line, "\r\n")

	foundID, ok := lp.parser.ExtractField(line, lp.config.Keys)
	if !ok || !match(foundID, lp.config.SearchValue, lp.config.IgnoreCase) {
		return nil, false
	}

	entry, err := lp.parser.Parse(line, service)
	if err != nil {
		return nil, false
	}

	return &entry, true
}

func (lp *LineProcessor) AddEntry(
	entry domain.LogEntry,
	results *[]domain.LogEntry,
	h *entryHeap,
) {
	if lp.config.Limit <= 0 {
		*results = append(*results, entry)
		return
	}

	if h.Len() < lp.config.Limit {
		heap.Push(h, entry)
		return
	}

	if entry.Timestamp.After((*h)[0].Timestamp) {
		heap.Pop(h)
		heap.Push(h, entry)
	}
}
