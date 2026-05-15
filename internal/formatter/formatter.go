package formatter

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

var tsFormat = "2006-01-02T15:04:05.000Z07:00"

type OutputFormat string

const (
	OutputPretty OutputFormat = "pretty"
	OutputJSON   OutputFormat = "json"
)

type Formatter struct {
	colorizer    *Colorizer
	serviceWidth int
	searchKeys   []string
	output       OutputFormat
}

func NewFormatter(entries []domain.LogEntry, searchKeys []string, output OutputFormat) *Formatter {
	max := 0
	for _, e := range entries {
		if len(e.Service) > max {
			max = len(e.Service)
		}
	}

	return &Formatter{
		colorizer:    NewColorizer(),
		serviceWidth: max,
		searchKeys:   searchKeys,
		output:       output,
	}
}

func (f *Formatter) padAfter(service string) string {
	if len(service) >= f.serviceWidth {
		return ""
	}
	return strings.Repeat(" ", f.serviceWidth-len(service))
}

func (f *Formatter) Format(entry domain.LogEntry) string {
	switch f.output {
	case OutputJSON:
		return f.outputJSON(entry)

	case OutputPretty:
		fallthrough

	default:
		return f.outputPretty(entry)
	}
}

func (f *Formatter) outputJSON(entry domain.LogEntry) string {
	out := make(map[string]any, len(entry.Fields)+5)

	out["timestamp"] = entry.Timestamp.Format(time.RFC3339Nano)
	out["service"] = entry.Service
	out["message"] = entry.Message
	out["context"] = entry.IsContext

	for k, v := range entry.Fields {
		// avoid accidental overwrite of core keys
		switch k {
		case "timestamp", "service", "message", "context":
			out["fields."+k] = v
		default:
			out[k] = v
		}
	}

	b, err := json.Marshal(out)
	if err == nil {
		return string(b)
	}

	errorOut := map[string]any{
		"error":   "failed to marshal log entry",
		"details": err.Error(),
		"raw":     entry.Raw,
	}

	fallback, _ := json.Marshal(errorOut)
	return string(fallback)
}

func (f *Formatter) outputPretty(entry domain.LogEntry) string {
	serviceColor := f.colorizer.Color(entry.Service)
	padding := f.padAfter(entry.Service)

	msg := f.renderPrettyMessage(entry)

	if entry.IsContext {
		msg = dim + msg + reset
	}

	return fmt.Sprintf(
		"%s%s%s%s %s[%s]%s%s | %s%s%s",
		dim,
		tsColor,
		entry.Timestamp.Format(tsFormat),
		reset,

		serviceColor,
		entry.Service,
		reset,
		padding,

		msgColor,
		msg,
		reset,
	)
}

func (f *Formatter) renderPrettyMessage(entry domain.LogEntry) string {
	fields := f.renderPrettyFields(entry.Fields)

	if entry.Message == "" {
		return fields
	}

	if fields == "" {
		return entry.Message
	}

	return entry.Message + " " + fields
}

func (f *Formatter) renderPrettyFields(fields map[string]any) string {
	var kvParts []string

	for _, pair := range sortKVByPriority(fields) {
		key := pair.key
		val := pair.value

		renderedVal := stringifyValue(val)

		if key == "level" {
			renderedVal = f.colorLevel(val)
		}

		key = f.highlightKey(key)

		kvParts = append(
			kvParts,
			fmt.Sprintf(
				"%s=%s",
				f.colorizer.Cyan(key),
				renderedVal,
			),
		)
	}

	return strings.Join(kvParts, " ")
}

func (f *Formatter) colorLevel(v any) string {
	val := stringifyValue(v)
	switch strings.ToLower(val) {
	case "error":
		return f.colorizer.Red(val)
	case "warn", "warning":
		return f.colorizer.Yellow(val)
	default:
		return val
	}
}

func (f *Formatter) highlightKey(key string) string {
	if slices.Contains(f.searchKeys, key) {
		return f.colorizer.Bold(key)
	}
	return key
}

type kv struct {
	key   string
	value any
}

func sortKVByPriority(fields map[string]any) []kv {
	pairs := make([]kv, 0, len(fields))

	for k, v := range fields {
		pairs = append(pairs, kv{
			key:   k,
			value: v,
		})
	}

	priority := func(key string) int {
		switch key {
		case "level":
			return 1
		case "request_id":
			return 2
		default:
			return 99
		}
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		pi := priority(pairs[i].key)
		pj := priority(pairs[j].key)

		if pi != pj {
			return pi < pj
		}

		return pairs[i].key < pairs[j].key
	})

	return pairs
}

func stringifyValue(v any) string {
	switch val := v.(type) {
	case string:
		return val

	case nil:
		return "null"

	default:
		return fmt.Sprintf("%v", val)
	}
}
