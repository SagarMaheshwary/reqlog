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
	offsets       map[string]int64
	lineProcessor *LineProcessor
}

func NewFileScanner(lp *LineProcessor) *FileScanner {
	return &FileScanner{
		offsets:       make(map[string]int64),
		lineProcessor: lp,
	}
}

func (fs *FileScanner) Scan(files []string) []domain.LogEntry {
	var h entryHeap
	var results []domain.LogEntry
	cfg := fs.lineProcessor.config
	sinceTime := parseSince(cfg.Since)

	if cfg.Limit > 0 {
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

					entry, ok := fs.lineProcessor.ProcessLine(line, service)
					if ok && passesSince(entry, sinceTime) {
						fs.lineProcessor.AddEntry(*entry, &results, &h)
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

	if cfg.Limit > 0 {
		results = make([]domain.LogEntry, 0, h.Len())
		for h.Len() > 0 {
			results = append(results, heap.Pop(&h).(domain.LogEntry))
		}
	}

	return results
}

func (fs *FileScanner) Follow(files []string) {
	f := formatter.NewFormatter([]domain.LogEntry{})

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

						entry, ok := fs.lineProcessor.ProcessLine(line, service)
						if !ok {
							continue
						}

						fmt.Println(f.Format(*entry))
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

func (fs *FileScanner) ListSources() ([]string, error) {
	cfg := fs.lineProcessor.config
	exact := map[string]struct{}{}
	prefixes := []string{}

	for _, s := range cfg.Services {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		if strings.HasSuffix(s, "*") {
			prefixes = append(prefixes, strings.TrimSuffix(s, "*"))
		} else {
			exact[s] = struct{}{}
		}
	}

	matchesService := func(name string) bool {
		if len(exact) == 0 && len(prefixes) == 0 {
			return true
		}

		name = strings.TrimSuffix(name, ".log")

		if _, ok := exact[name]; ok {
			return true
		}

		for _, p := range prefixes {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}

	if cfg.Recursive {
		files := make([]string, 0, 16)

		err := filepath.WalkDir(cfg.Dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				logFileScanError(path, err)
				return nil // continue walking
			}

			if d.IsDir() {
				return nil
			}

			name := d.Name()

			if !strings.HasSuffix(name, ".log") {
				return nil
			}

			if !matchesService(name) {
				return nil
			}

			files = append(files, path)
			return nil
		})
		return files, err
	}

	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 16)

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

		files = append(files, filepath.Join(cfg.Dir, name))
	}

	return files, nil
}
