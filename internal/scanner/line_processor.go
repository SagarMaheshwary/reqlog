package scanner

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/tidwall/gjson"
)

type LineProcessor struct {
	config        *Config
	timeParser    TimeParser
	mu            sync.RWMutex
	timestampKeys map[string]string
}

func NewLineProcessor(cfg *Config, tp TimeParser) *LineProcessor {
	return &LineProcessor{
		config:        cfg,
		timeParser:    tp,
		mu:            sync.RWMutex{},
		timestampKeys: make(map[string]string),
	}
}

func (lp *LineProcessor) ProcessLine(line, service string, host string) (*domain.LogEntry, bool) {
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

	return lp.Parse(line, service, host, false)
}

func (lp *LineProcessor) Parse(line, service string, host string, skipMatch bool) (*domain.LogEntry, bool) {
	line = strings.TrimRight(line, "\r\n")

	switch lp.config.Format {
	case config.FormatJSON:
		return lp.processJSONLine(line, service, host, skipMatch)
	case config.FormatText:
		return lp.processTextLine(line, service, host, skipMatch)
	case config.FormatAuto:
		fallthrough
	default:
		if isJSONLine(line) {
			return lp.processJSONLine(line, service, host, skipMatch)
		}
		return lp.processTextLine(line, service, host, skipMatch)
	}
}

func (lp *LineProcessor) processJSONLine(line string, service string, host string, skipMatch bool) (*domain.LogEntry, bool) {
	if !gjson.Valid(line) {
		return nil, false
	}
	obj := gjson.Parse(line)

	if !skipMatch {
		foundID, ok := extractJSONField(obj, lp.config.Keys)
		if !ok || !match(foundID, lp.config.SearchValue, lp.config.IgnoreCase) {

			return nil, false
		}
	}

	tsKey, tsVal, ok := lp.extractJSONTimestampValue(obj, service)
	if !ok {
		return nil, false
	}
	ts, ok := lp.timeParser.Parse(tsVal, service)
	if !ok {
		return nil, false
	}

	parsedFields := extractJSONFields(obj, tsKey)

	return &domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Host:      host,
		Message:   parsedFields.Message,
		Fields:    parsedFields.Fields,
	}, true
}

func (lp *LineProcessor) processTextLine(line string, service string, host string, skipMatch bool) (*domain.LogEntry, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, false
	}

	if !skipMatch {
		foundID, ok := extractTextField(parts, lp.config.Keys)
		if !ok || !match(foundID, lp.config.SearchValue, lp.config.IgnoreCase) {
			return nil, false
		}
	}

	ts, ok := lp.timeParser.Parse(parts[0], service)
	if !ok {
		return nil, false
	}

	parsed := extractTextFields(parts[1:]) // skip timestamp

	return &domain.LogEntry{
		Timestamp: ts,
		Service:   service,
		Host:      host,
		Message:   parsed.Message,
		Fields:    parsed.Fields,
		Raw:       line,
	}, true
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

type ParsedFields struct {
	Fields  map[string]any
	Message string
}

func extractJSONFields(obj gjson.Result, tsKey string) ParsedFields {
	fields := make(map[string]any, 8)
	var message string

	obj.ForEach(func(key, value gjson.Result) bool {
		k := key.String()

		if k == tsKey {
			return true
		}

		v := extractJSONValue(value)

		if _, isMsg := MessageKeys[k]; isMsg {
			if message == "" {
				message = value.String()
			}
			return true
		}

		fields[k] = v
		return true
	})

	return ParsedFields{
		Fields:  fields,
		Message: message,
	}
}

func extractJSONValue(value gjson.Result) any {
	switch value.Type {
	case gjson.String:
		return value.String()

	case gjson.Number:
		return value.Value()

	case gjson.True:
		return true

	case gjson.False:
		return false

	case gjson.Null:
		return nil

	case gjson.JSON:
		var v any
		if err := json.Unmarshal([]byte(value.Raw), &v); err == nil {
			return v
		}

		return value.Raw

	default:
		return value.Raw
	}
}

func extractTextFields(parts []string) ParsedFields {
	fields := make(map[string]any, 8)
	messageParts := make([]string, 0, len(parts))

	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if ok && key != "" {
			fields[key] = parseTextValue(value)
			continue
		}

		messageParts = append(messageParts, part)
	}

	return ParsedFields{
		Fields:  fields,
		Message: strings.Join(messageParts, " "),
	}
}

func parseTextValue(v string) any {
	if strings.ContainsAny(v, ".eE") {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}

	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}

	switch strings.ToLower(v) {
	case "true":
		return true
	case "false":
		return false
	}

	return v
}

func isJSONLine(line string) bool {
	line = strings.TrimSpace(line)

	if len(line) < 2 {
		return false
	}

	return line[0] == '{'
}
