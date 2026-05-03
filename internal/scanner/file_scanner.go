package scanner

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"io"
	"log"
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
	JSONMode    bool
}

type FileScanner struct {
	offsets        map[string]int64
	lineProcessor  *LineProcessor
	followInterval time.Duration
	out            io.Writer
	errOut         io.Writer
}

func NewFileScanner(
	lp *LineProcessor,
	followInterval time.Duration,
	out io.Writer,
	errOut io.Writer,
) *FileScanner {
	return &FileScanner{
		offsets:        make(map[string]int64),
		lineProcessor:  lp,
		followInterval: followInterval, // default
		out:            out,
		errOut:         errOut,
	}
}

func (fs *FileScanner) Scan(files []string) []domain.LogEntry {
	var h entryHeap
	var results []domain.LogEntry
	cfg := fs.lineProcessor.config
	sinceTime, err := parseSince(cfg.Since)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Limit > 0 {
		heap.Init(&h)
	}

	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			logScanError(fs.errOut, path, err)
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
			logScanError(fs.errOut, path, err)
		}

		fs.offsets[path] = offset
	}

	if cfg.Limit > 0 {
		results = drainHeap(&h)
	}

	return results
}

func (fs *FileScanner) Follow(ctx context.Context, files []string, f formatter.LogFormatter) {
	ticker := time.NewTicker(fs.followInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			for _, path := range files {
				fs.processFile(path, f)
			}
		}
	}
}

func (fs *FileScanner) processFile(path string, f formatter.LogFormatter) {
	file, err := os.Open(path)
	if err != nil {
		logScanError(fs.errOut, path, err)
		return
	}
	defer file.Close()

	service := strings.TrimSuffix(filepath.Base(path), ".log")

	offset := fs.offsets[path]

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		logScanError(fs.errOut, path, err)
		return
	}

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			offset += int64(len(line))

			entry, ok := fs.lineProcessor.ProcessLine(line, service)
			if ok {
				fmt.Fprintln(fs.out, f.Format(*entry))
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			logScanError(fs.errOut, path, err)
			return
		}
	}

	fs.offsets[path] = offset
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

		if before, ok := strings.CutSuffix(s, "*"); ok {
			prefixes = append(prefixes, before)
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
				logScanError(fs.errOut, path, err)
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
