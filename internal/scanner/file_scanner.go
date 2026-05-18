package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type FileScanner struct {
	offsets        map[string]int64
	lineProcessor  *LineProcessor
	followInterval time.Duration
	out            io.Writer
	errOut         io.Writer
	now            time.Time
}

func NewFileScanner(
	lp *LineProcessor,
	followInterval time.Duration,
	out io.Writer,
	errOut io.Writer,
	now time.Time,
) *FileScanner {
	return &FileScanner{
		offsets:        make(map[string]int64),
		lineProcessor:  lp,
		followInterval: followInterval,
		out:            out,
		errOut:         errOut,
		now:            now,
	}
}

func (fs *FileScanner) Scan(files []string) ([]domain.LogEntry, error) {
	cfg := fs.lineProcessor.config

	collector, err := NewEntryCollector(cfg, fs.now)
	if err != nil {
		return nil, err
	}

	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			logScanError(fs.errOut, path, err)
			continue
		}

		offset, err := func() (int64, error) {
			defer file.Close()
			collector.StartSource()

			service := strings.TrimSuffix(filepath.Base(path), ".log")
			reader := bufio.NewReader(file)
			var offset int64 = 0

			engine := NewContextEngine(fs.lineProcessor, collector, cfg.Context)

			for {
				line, err := reader.ReadString('\n')

				if len(line) > 0 {
					offset += int64(len(line))

					entry, ok := fs.lineProcessor.ProcessLine(line, service)
					continueReading := engine.Handle(ContextLine{
						Line:    line,
						Service: service,
						Entry:   entry,
						IsMatch: ok,
					})
					if !continueReading {
						break
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

	return collector.Results(), nil
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

	matcher := buildServiceMatcher(cfg.Services)

	if cfg.Recursive {
		return fs.listRecursive(cfg.Dir, matcher)
	}

	return fs.listFlat(cfg.Dir, matcher)
}

func (fs *FileScanner) listRecursive(
	dir string,
	matchesService func(string) bool,
) ([]string, error) {
	files := make([]string, 0, 16)

	err := filepath.WalkDir(
		dir,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				logScanError(fs.errOut, path, err)
				return nil // continue walking
			}

			if d.IsDir() {
				return nil
			}

			name := d.Name()

			if !isLogFile(name) {
				return nil
			}

			if !matchesService(name) {
				return nil
			}

			files = append(files, path)

			return nil
		},
	)

	return files, err
}

func (fs *FileScanner) listFlat(
	dir string,
	matchesService func(string) bool,
) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 16)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !isLogFile(name) {
			continue
		}

		if !matchesService(name) {
			continue
		}

		files = append(files, filepath.Join(dir, name))
	}

	return files, nil
}

func buildServiceMatcher(services []string) func(string) bool {
	exact := map[string]struct{}{}
	prefixes := []string{}

	for _, s := range services {
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

	return func(name string) bool {
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
}

func isLogFile(name string) bool {
	return strings.HasSuffix(name, ".log")
}
