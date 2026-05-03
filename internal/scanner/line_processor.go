package scanner

import (
	"container/heap"
	"sort"
	"strings"
	"sync"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/tidwall/gjson"
)

type LineProcessor struct {
	config        *ScanConfig
	timeParser    TimeParser
	mu            sync.RWMutex
	timestampKeys map[string]string
}

func NewLineProcessor(cfg *ScanConfig, tp TimeParser) *LineProcessor {
	return &LineProcessor{
		config:        cfg,
		timeParser:    tp,
		mu:            sync.RWMutex{},
		timestampKeys: make(map[string]string),
	}
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

	if lp.config.JSONMode {
		return lp.processJSONLine(line, service)
	}

	line = strings.TrimRight(line, "\r\n")
	return lp.processTextLine(line, service)
}

func (lp *LineProcessor) processJSONLine(line string, service string) (*domain.LogEntry, bool) {
	if !gjson.Valid(line) {
		return nil, false
	}
	obj := gjson.Parse(line)

	foundID, ok := extractJSONField(obj, lp.config.Keys)
	if !ok || !match(foundID, lp.config.SearchValue, lp.config.IgnoreCase) {

		return nil, false
	}

	tsKey, tsVal, ok := lp.extractJSONTimestampValue(obj, service)
	if !ok {
		return nil, false
	}
	ts, ok := lp.timeParser.Parse(tsVal, service)
	if !ok {
		return nil, false
	}

	return &domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   buildJSONMessage(obj, tsKey),
	}, true
}

func (lp *LineProcessor) processTextLine(line string, service string) (*domain.LogEntry, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, false
	}

	foundID, ok := extractTextField(parts, lp.config.Keys)
	if !ok || !match(foundID, lp.config.SearchValue, lp.config.IgnoreCase) {
		return nil, false
	}

	ts, ok := lp.timeParser.Parse(parts[0], service)
	if !ok {
		return nil, false
	}

	return &domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Message:   extractTextMessage(line),
	}, true
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

func (lp *LineProcessor) extractJSONTimestampValue(
	obj gjson.Result,
	service string,
) (key string, value string, ok bool) {
	lp.mu.RLock()
	knownKey := lp.timestampKeys[service]
	lp.mu.RUnlock()

	if knownKey != "" {
		v := obj.Get(knownKey)
		if v.Exists() {
			return knownKey, v.String(), true
		}
		return "", "", false
	}

	for _, key := range TimestampKeys {
		v := obj.Get(key)
		if v.Exists() {
			lp.mu.Lock()
			lp.timestampKeys[service] = key
			lp.mu.Unlock()

			return key, v.String(), true
		}
	}

	return "", "", false
}

func extractJSONField(obj gjson.Result, keys []string) (string, bool) {
	for _, key := range keys {
		v := obj.Get(key)
		if v.Exists() {
			return v.String(), true
		}
	}
	return "", false
}

func extractTextField(parts []string, keys []string) (string, bool) {
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		if !strings.Contains(part, "=") {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		for _, key := range keys {
			if kv[0] == key {
				return stripQuotes(kv[1]), true
			}
		}
	}

	return "", false
}

func stripQuotes(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return val
}

func buildJSONMessage(obj gjson.Result, tsKey string) string {
	parts := make([]string, 0, 8)

	obj.ForEach(func(key, value gjson.Result) bool {
		k := key.String()
		if tsKey == k {
			return true
		}

		parts = append(parts, formatJSONField(k, value))
		return true
	})

	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func formatJSONField(key string, value gjson.Result) string {
	switch value.Type {
	case gjson.String:
		return key + "=" + value.String()
	default:
		return key + "=" + value.Raw
	}
}

func extractTextMessage(line string) string {
	i := strings.IndexByte(line, ' ')
	if i == -1 {
		return ""
	}
	return strings.TrimLeft(line[i+1:], " ")
}
