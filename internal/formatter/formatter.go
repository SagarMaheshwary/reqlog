package formatter

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

type Formatter struct {
	colorizer    *Colorizer
	serviceWidth int
	searchKeys   []string
}

func NewFormatter(entries []domain.LogEntry, searchKeys []string) *Formatter {
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
	}
}

func (f *Formatter) padAfter(service string) string {
	if len(service) >= f.serviceWidth {
		return ""
	}
	return strings.Repeat(" ", f.serviceWidth-len(service))
}

func (f *Formatter) Format(entry domain.LogEntry) string {
	serviceColor := f.colorizer.Color(entry.Service)
	padding := f.padAfter(entry.Service)

	return fmt.Sprintf(
		"%s%s%s%s %s[%s]%s%s | %s%s%s",
		dim, tsColor,
		entry.Timestamp.Format(time.RFC3339),
		reset,

		serviceColor,
		entry.Service,
		reset,
		padding,

		msgColor,
		f.formatMessage(entry.Message),
		reset,
	)
}

func (f *Formatter) colorLevel(val string) string {
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

func (f *Formatter) formatMessage(msg string) string {
	mainMsg, pairs := parseMessage(msg)

	var kvParts []string

	for _, p := range pairs {
		key := p.key
		val := p.value

		if key == "level" {
			val = f.colorLevel(val)
		}

		key = f.highlightKey(key)

		kvParts = append(kvParts,
			fmt.Sprintf("%s=%s",
				f.colorizer.Cyan(key),
				val,
			),
		)
	}

	if len(kvParts) == 0 {
		return mainMsg
	}

	if mainMsg == "" {
		return fmt.Sprintf("%s", strings.Join(kvParts, ""))
	}

	return fmt.Sprintf("%s %s", mainMsg, strings.Join(kvParts, " "))
}

type kv struct {
	key   string
	value string
}

func parseMessage(msg string) (string, []kv) {
	parts := strings.Fields(msg)

	var pairs []kv
	var messageParts []string

	i := 0
	for i < len(parts) {
		part := parts[i]

		if strings.Contains(part, "=") {
			split := strings.SplitN(part, "=", 2)
			key := split[0]
			value := split[1]

			j := i + 1
			for j < len(parts) && !strings.Contains(parts[j], "=") {
				value += " " + parts[j]
				j++
			}

			pairs = append(pairs, kv{key, value})
			i = j
			continue
		}

		messageParts = append(messageParts, part)
		i++
	}

	var mainMsg string
	filtered := make([]kv, 0, len(pairs))

	for _, p := range pairs {
		if p.key == "message" || p.key == "msg" {
			mainMsg = p.value
			continue
		}
		filtered = append(filtered, p)
	}

	if mainMsg == "" {
		mainMsg = strings.Join(messageParts, " ")
	}

	filtered = sortKVByPriority(filtered)

	return mainMsg, filtered
}

var keyPriority = map[string]int{
	"level":      1,
	"request_id": 2,
}

func sortKVByPriority(pairs []kv) []kv {
	priority := func(key string) int {
		if val, ok := keyPriority[key]; ok {
			return val
		}
		return 99
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
