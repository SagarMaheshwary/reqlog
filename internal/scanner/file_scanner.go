package scanner

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

type ScanConfig struct {
	Dir         string
	SearchValue string
	IgnoreCase  bool
	Keys        []string
	Since       string
	Limit       int
}

type FileScanner struct {
	parser  parser.LogParser
	offsets map[string]int64
	config  ScanConfig
}

func NewFileScanner(cfg ScanConfig, p parser.LogParser) *FileScanner {
	return &FileScanner{
		parser:  p,
		offsets: make(map[string]int64),
		config:  cfg,
	}
}

func (fs *FileScanner) Scan() ([]domain.LogEntry, error) {
	var h entryHeap
	var results []domain.LogEntry
	sinceTime := parseSince(fs.config.Since)

	if fs.config.Limit > 0 {
		heap.Init(&h)
	}

	err := filepath.Walk(fs.config.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".log") {
			return nil
		}

		service := strings.TrimSuffix(filepath.Base(path), ".log")

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		var offset int64 = 0

		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return err
			}

			if len(line) > 0 {
				offset += int64(len(line))

				if fs.config.IgnoreCase {
					if !containsFoldASCII(line, fs.config.SearchValue) {
						continue
					}
				} else {
					if !strings.Contains(line, fs.config.SearchValue) {
						continue
					}
				}

				line = strings.TrimRight(line, "\r\n")

				foundID, ok := fs.parser.ExtractField(line, fs.config.Keys)
				if ok && match(foundID, fs.config.SearchValue, fs.config.IgnoreCase) {
					entry, parseErr := fs.parser.Parse(line, service)
					if parseErr == nil && passesSince(entry, sinceTime) {
						if fs.config.Limit <= 0 {
							results = append(results, entry)
						} else {
							if h.Len() < fs.config.Limit {
								heap.Push(&h, entry)
							} else if entry.Timestamp.After(h[0].Timestamp) {
								heap.Pop(&h)
								heap.Push(&h, entry)
							}
						}
					}
				}
			}

			if err == io.EOF {
				break
			}
		}

		fs.offsets[path] = offset
		return nil
	})

	if fs.config.Limit > 0 {
		results = make([]domain.LogEntry, 0, h.Len())
		for h.Len() > 0 {
			results = append(results, heap.Pop(&h).(domain.LogEntry))
		}
	}

	return results, err
}

func (fs *FileScanner) Follow() error {
	colorizer := formatter.NewColorizer()

	for {
		filepath.Walk(fs.config.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".log") {
				return nil
			}
			service := strings.TrimSuffix(filepath.Base(path), ".log")
			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()

			offset := fs.offsets[path]
			file.Seek(offset, io.SeekStart)
			reader := bufio.NewReader(file)
			currentOffset := offset

			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					currentOffset += int64(len(line))

					if fs.config.IgnoreCase {
						if !containsFoldASCII(line, fs.config.SearchValue) {
							continue
						}
					} else {
						if !strings.Contains(line, fs.config.SearchValue) {
							continue
						}
					}

					line = strings.TrimRight(line, "\r\n")

					foundID, ok := fs.parser.ExtractField(line, fs.config.Keys)
					if ok && match(foundID, fs.config.SearchValue, fs.config.IgnoreCase) {
						entry, parseErr := fs.parser.Parse(line, service)
						if parseErr == nil {
							fmt.Println(formatter.Format(entry, colorizer))
						}
					}
				}

				if err != nil {
					if err == io.EOF {
						break
					}
					return nil
				}
			}

			fs.offsets[path] = currentOffset
			return nil
		})

		time.Sleep(1 * time.Second)
	}
}

func match(foundID, SearchValue string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(foundID, SearchValue)
	}
	return foundID == SearchValue
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

func asciiLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func containsFoldASCII(s, substr string) bool {
	n := len(substr)
	if n == 0 {
		return true
	}
	if n > len(s) {
		return false
	}

	first := asciiLower(substr[0])

	for i := 0; i <= len(s)-n; i++ {
		if asciiLower(s[i]) != first {
			continue
		}

		ok := true
		for j := 1; j < n; j++ {
			if asciiLower(s[i+j]) != asciiLower(substr[j]) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}

	return false
}
