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
	Recursive   bool
	Services    []string
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

func (fs *FileScanner) Scan(files []string) []domain.LogEntry {
	var h entryHeap
	var results []domain.LogEntry
	sinceTime := parseSince(fs.config.Since)

	if fs.config.Limit > 0 {
		heap.Init(&h)
	}

	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			logFileScanError(path, err)
			continue
		}

		offset, err := func() (int64, error) {
			defer file.Close()

			service := strings.TrimSuffix(filepath.Base(path), ".log")
			reader := bufio.NewReader(file)
			var offset int64 = 0

			for {
				line, err := reader.ReadString('\n')

				if len(line) > 0 {
					offset += int64(len(line))

					entry, ok := fs.processLine(line, service)
					if ok && passesSince(entry, sinceTime) {
						fs.addEntry(*entry, &results, &h)
					}
				}

				if err != nil {
					if err == io.EOF {
						break
					}
					return 0, err
				}
			}

			return offset, nil
		}()

		if err != nil {
			logFileScanError(path, err)
		}

		fs.offsets[path] = offset
	}

	if fs.config.Limit > 0 {
		results = make([]domain.LogEntry, 0, h.Len())
		for h.Len() > 0 {
			results = append(results, heap.Pop(&h).(domain.LogEntry))
		}
	}

	return results
}

func (fs *FileScanner) Follow(files []string) {
	colorizer := formatter.NewColorizer()

	for {
		for _, path := range files {
			file, err := os.Open(path)
			if err != nil {
				logFileScanError(path, err)
				continue
			}

			offset, err := func() (int64, error) {
				defer file.Close()

				service := strings.TrimSuffix(filepath.Base(path), ".log")

				offset := fs.offsets[path]
				_, err := file.Seek(offset, io.SeekStart)
				if err != nil {
					return 0, err
				}

				reader := bufio.NewReader(file)

				for {
					line, err := reader.ReadString('\n')
					if len(line) > 0 {
						offset += int64(len(line))

						entry, ok := fs.processLine(line, service)
						if !ok {
							continue
						}

						fmt.Println(formatter.Format(*entry, colorizer))
					}

					if err != nil {
						if err == io.EOF {
							break
						}
						return 0, err
					}
				}

				return offset, nil
			}()

			if err != nil {
				logFileScanError(path, err)
			}

			fs.offsets[path] = offset
		}

		time.Sleep(1 * time.Second)
	}
}

func (fs *FileScanner) ListLogFiles() ([]string, error) {
	var files []string

	serviceSet := make(map[string]struct{})
	for _, s := range fs.config.Services {
		s = strings.TrimSpace(s)
		if s != "" {
			serviceSet[s] = struct{}{}
		}
	}

	matchesService := func(name string) bool {
		if len(serviceSet) == 0 {
			return true
		}
		service := strings.TrimSuffix(name, ".log")
		_, ok := serviceSet[service]
		return ok
	}

	if fs.config.Recursive {
		err := filepath.Walk(fs.config.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(info.Name(), ".log") {
				return nil
			}

			if !matchesService(info.Name()) {
				return nil
			}

			files = append(files, path)
			return nil
		})
		return files, err
	}

	// non-recursive
	entries, err := os.ReadDir(fs.config.Dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.HasSuffix(name, ".log") {
			continue
		}

		if !matchesService(name) {
			continue
		}

		files = append(files, filepath.Join(fs.config.Dir, name))
	}

	return files, nil
}

func (fs *FileScanner) processLine(line, service string) (*domain.LogEntry, bool) {
	// fast pre-filter
	if fs.config.IgnoreCase {
		if !containsFoldASCII(line, fs.config.SearchValue) {
			return nil, false
		}
	} else {
		if !strings.Contains(line, fs.config.SearchValue) {
			return nil, false
		}
	}

	line = strings.TrimRight(line, "\r\n")

	foundID, ok := fs.parser.ExtractField(line, fs.config.Keys)
	if !ok || !match(foundID, fs.config.SearchValue, fs.config.IgnoreCase) {
		return nil, false
	}

	entry, err := fs.parser.Parse(line, service)
	if err != nil {
		return nil, false
	}

	return &entry, true
}

func (fs *FileScanner) addEntry(
	entry domain.LogEntry,
	results *[]domain.LogEntry,
	h *entryHeap,
) {
	if fs.config.Limit <= 0 {
		*results = append(*results, entry)
		return
	}

	if h.Len() < fs.config.Limit {
		heap.Push(h, entry)
		return
	}

	if entry.Timestamp.After((*h)[0].Timestamp) {
		heap.Pop(h)
		heap.Push(h, entry)
	}
}

func match(foundID, SearchValue string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(foundID, SearchValue)
	}
	return foundID == SearchValue
}

func passesSince(entry *domain.LogEntry, sinceTime time.Time) bool {
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

func logFileScanError(path string, err error) {
	fmt.Fprintf(os.Stderr, "error scanning %s: %v\n", path, err)
}
