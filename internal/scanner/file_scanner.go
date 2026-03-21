package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

type ScanConfig struct {
	Dir        string
	RequestID  string
	IgnoreCase bool
	Key        string
	Since      string
}

type FileScanner struct {
	parser parser.LogParser
}

func NewFileScanner(p parser.LogParser) *FileScanner {
	return &FileScanner{parser: p}
}

func (fs *FileScanner) Scan(cfg ScanConfig) ([]domain.LogEntry, error) {
	var results []domain.LogEntry

	keys := parser.DefaultKeys
	if cfg.Key != "" {
		keys = []string{cfg.Key}
	}

	sinceTime := parseSince(cfg.Since)

	err := filepath.Walk(cfg.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".log") {
			return nil
		}

		service := strings.TrimSuffix(filepath.Base(path), ".log")

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.Contains(line, cfg.RequestID) {
				continue
			}

			entry, fields, err := fs.parser.Parse(line, service)
			if err != nil {
				continue
			}

			reqID, ok := extractRequestID(fields, keys)
			if !ok {
				continue
			}

			if !entryMatches(entry, reqID, cfg.RequestID, cfg.IgnoreCase, sinceTime) {
				continue
			}

			results = append(results, entry)
		}

		return scanner.Err()
	})

	return results, err
}

func (fs *FileScanner) Follow(cfg ScanConfig) error {
	keys := parser.DefaultKeys
	if cfg.Key != "" {
		keys = []string{cfg.Key}
	}

	files := make(map[string]int64)
	colorizor := formatter.NewColorizer()

	for {
		filepath.Walk(cfg.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".log") {
				return nil
			}

			offset := files[path]

			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()

			file.Seek(offset, 0)

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				line := scanner.Text()

				if !strings.Contains(line, cfg.RequestID) {
					continue
				}

				service := strings.TrimSuffix(filepath.Base(path), ".log")

				entry, fields, err := fs.parser.Parse(line, service)
				if err != nil {
					continue
				}

				reqID, ok := extractRequestID(fields, keys)
				if !ok {
					continue
				}

				if match(reqID, cfg.RequestID, cfg.IgnoreCase) {
					fmt.Println(formatter.Format(entry, colorizor))
				}
			}

			pos, _ := file.Seek(0, 1)
			files[path] = pos

			return nil
		})

		time.Sleep(1 * time.Second)
	}
}

func match(foundID, requestID string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(foundID, requestID)
	}
	return foundID == requestID
}

func entryMatches(
	entry domain.LogEntry,
	foundID string,
	requestID string,
	ignoreCase bool,
	sinceTime time.Time,
) bool {
	return match(foundID, requestID, ignoreCase) && passesSince(entry, sinceTime)
}

func passesSince(entry domain.LogEntry, sinceTime time.Time) bool {
	if sinceTime.IsZero() {
		return true
	}
	return !entry.Timestamp.Before(sinceTime)
}

func parseSince(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}
	}

	return time.Now().Add(-d)
}

func extractRequestID(fields map[string]string, keys []string) (string, bool) {
	for _, key := range keys {
		if value := fields[key]; value != "" {
			return value, true
		}
	}
	return "", false
}
