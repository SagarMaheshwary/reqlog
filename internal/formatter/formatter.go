package formatter

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

var tsFormat = "2006-01-02T15:04:05.000Z07:00"

type Formatter struct {
	colorizer    *Colorizer
	serviceWidth int
	searchKeys   []string
	output       config.OutputFormat
	context      int
	fields       []string
}

type Opts struct {
	Entries    []domain.LogEntry
	SearchKeys []string
	Output     config.OutputFormat
	Context    int
	Fields     []string
}

func NewFormatter(opts *Opts) *Formatter {
	max := 0
	for _, e := range opts.Entries {
		if len(e.Service) > max {
			max = len(e.Service)
		}
	}

	return &Formatter{
		colorizer:    NewColorizer(),
		serviceWidth: max,
		searchKeys:   opts.SearchKeys,
		output:       opts.Output,
		context:      opts.Context,
		fields:       opts.Fields,
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
	case config.OutputJSON:
		return f.outputJSON(entry)

	case config.OutputPretty:
		fallthrough

	default:
		return f.outputPretty(entry)
	}
}

func (f *Formatter) outputJSON(entry domain.LogEntry) string {
	if len(f.fields) > 0 {
		return f.outputJSONFields(entry)
	}

	out := make(map[string]any, len(entry.Fields)+6)

	out["timestamp"] = entry.Timestamp.Format(time.RFC3339Nano)
	out["service"] = entry.Service
	out["message"] = entry.Message

	if f.context > 0 {
		out["context"] = entry.IsContext
	}
	if entry.Host != "" {
		out["host"] = entry.Host
	}

	for k, v := range entry.Fields {
		// avoid accidental overwrite of core keys
		switch k {
		case "timestamp", "service", "message", "context", "host":
			out["fields."+k] = v
		default:
			out[k] = v
		}
	}

	return marshal(out, entry)
}

func (f *Formatter) outputJSONFields(
	entry domain.LogEntry,
) string {
	out := make(map[string]any, len(f.fields))

	for _, k := range f.fields {
		switch k {
		case "timestamp":
			out[k] = entry.Timestamp.Format(time.RFC3339Nano)

		case "service":
			out[k] = entry.Service

		case "message":
			out[k] = entry.Message

		case "context":
			if f.context > 0 {
				out[k] = entry.IsContext
			}

		case "host":
			if entry.Host != "" {
				out[k] = entry.Host
			}

		default:
			fieldKey := k

			// allow "fields." prefix to explicitly access fields
			// when there's a name conflict with core keys
			if after, ok := strings.CutPrefix(k, "fields."); ok {
				fieldKey = after
			}
			if val, ok := entry.Fields[fieldKey]; ok {
				out[k] = val
			}
		}
	}

	return marshal(out, entry)
}

func (f *Formatter) outputPretty(entry domain.LogEntry) string {
	serviceColor := f.colorizer.Color(entry.Service)
	padding := f.padAfter(entry.Service)

	msg := f.renderPrettyMessage(entry)

	service := entry.Service

	if entry.Host != "" {
		service = entry.Host + ":" + entry.Service
	}

	return fmt.Sprintf(
		"%s%s%s%s %s[%s]%s%s | %s%s%s",
		dim,
		tsColor,
		entry.Timestamp.Format(tsFormat),
		reset,

		serviceColor,
		service,
		reset,
		padding,

		msgColor,
		msg,
		reset,
	)
}

func (f *Formatter) renderPrettyMessage(entry domain.LogEntry) string {
	level, fields := f.renderPrettyFields(entry.Fields, entry.IsContext, f.fields)

	if entry.Message == "" {
		return fields
	}

	message := entry.Message
	if entry.IsContext {
		message = dim + entry.Message + reset
	} else {
		message = f.colorizer.Bold(entry.Message) + reset
	}

	return strings.TrimSpace(level + " " + message + " " + fields)
}

func (f *Formatter) renderPrettyFields(
	fields map[string]any,
	isContext bool,
	fieldsToInclude []string,
) (string, string) {
	var (
		kvParts []string
		level   string
	)

	if val, ok := fields["level"]; ok {
		level = f.colorLevel(val)

		if isContext {
			level = dim + level + reset
		}
	}

	kvParts = make([]string, 0, len(fields))

	// Respect --fields order
	if len(fieldsToInclude) > 0 {
		for _, key := range fieldsToInclude {
			val, ok := fields[key]
			if !ok || key == "level" {
				continue
			}

			kvParts = append(
				kvParts,
				f.renderPrettyField(key, val, isContext),
			)
		}

		return level, strings.Join(kvParts, " ")
	}

	keys := make([]string, 0, len(fields))

	for key := range fields {
		if key == "level" {
			continue
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		kvParts = append(
			kvParts,
			f.renderPrettyField(key, fields[key], isContext),
		)
	}

	return level, strings.Join(kvParts, " ")
}

func (f *Formatter) renderPrettyField(
	key string,
	val any,
	isContext bool,
) string {
	renderedVal := stringifyValue(val)

	if isContext {
		return fmt.Sprintf(
			"%s%s%s=%s%s",
			dim,
			f.colorizer.Cyan(key),
			dim,
			dim,
			renderedVal,
		)
	}

	key = f.highlightKey(key)

	return fmt.Sprintf(
		"%s=%s",
		f.colorizer.Cyan(key),
		renderedVal,
	)
}

func (f *Formatter) colorLevel(v any) string {
	val := strings.ToUpper(stringifyValue(v))
	switch val {
	case "INFO":
		return f.colorizer.Green(val)
	case "ERROR", "ERR":
		return f.colorizer.Red(val)
	case "WARN", "WARNING":
		return f.colorizer.Yellow(val)
	case "DEBUG":
		return f.colorizer.Blue(val)
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

func stringifyValue(v any) string {
	switch val := v.(type) {
	case string:
		return val

	case nil:
		return "null"

	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func marshal(out any, entry domain.LogEntry) string {
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
