package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/diagnostics"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type FileScanner struct {
	offsets        map[string]int64
	lineProcessor  *LineProcessor
	followInterval time.Duration
	out            io.Writer
	now            time.Time
	logFileReader  transport.LogFileReader
	host           string
	diagnostics    *diagnostics.Diagnostics
}

type FileScannerOpts struct {
	LineProcessor  *LineProcessor
	FollowInterval time.Duration
	Out            io.Writer
	Now            time.Time
	LogFileReader  transport.LogFileReader
	Host           string
	Diagnostics    *diagnostics.Diagnostics
}

func NewFileScanner(opts *FileScannerOpts) *FileScanner {
	return &FileScanner{
		offsets:        make(map[string]int64),
		lineProcessor:  opts.LineProcessor,
		followInterval: opts.FollowInterval,
		out:            opts.Out,
		now:            opts.Now,
		logFileReader:  opts.LogFileReader,
		host:           opts.Host,
		diagnostics:    opts.Diagnostics,
	}
}

func (fs *FileScanner) Scan(ctx context.Context, files []string) ([]domain.LogEntry, error) {
	cfg := fs.lineProcessor.config

	collector, err := NewEntryCollector(cfg, fs.now)
	if err != nil {
		return nil, err
	}

	for _, path := range files {
		file, err := fs.logFileReader.Open(ctx, path)
		if err != nil {
			fs.diagnostics.Error(fmt.Sprintf("Error opening file %s: %v", path, err), nil)
			continue
		}

		offset, err := func() (int64, error) {
			defer file.Close()
			collector.StartSource()

			service := strings.TrimSuffix(filepath.Base(path), ".log")
			reader := bufio.NewReader(file)
			engine := NewContextEngine(fs.lineProcessor, collector, cfg.Context)

			for {
				line, err := reader.ReadString('\n')

				if len(line) > 0 {
					entry, ok := fs.lineProcessor.ProcessLine(line, service, fs.host)
					continueReading := engine.Handle(ContextLine{
						Line:    line,
						Service: service,
						Host:    fs.host,
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

			size, err := fs.logFileReader.Size(ctx, path)
			if err != nil {
				return 0, err
			}
			return size, nil
		}()

		if err != nil {
			fs.diagnostics.Error(fmt.Sprintf("Error reading file %s: %v", path, err), nil)
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
				fs.scanFromOffset(ctx, path, f)
			}
		}
	}
}

func (fs *FileScanner) scanFromOffset(ctx context.Context, path string, f formatter.LogFormatter) {
	service := strings.TrimSuffix(filepath.Base(path), ".log")
	offset := fs.offsets[path]

	size, err := fs.logFileReader.Size(
		ctx,
		path,
	)
	if err != nil {
		fs.diagnostics.Error(fmt.Sprintf("Error reading file %s: %v", path, err), nil)
		return
	}

	// log rotation / truncation
	if size < offset {
		offset = 0
	}

	// no new content
	if size == offset {
		return
	}

	file, err := fs.logFileReader.OpenFromOffset(ctx, path, offset)
	if err != nil {
		fs.diagnostics.Error(fmt.Sprintf("Error opening file %s: %v", path, err), nil)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			offset += int64(len(line))

			entry, ok := fs.lineProcessor.ProcessLine(line, service, fs.host)
			if ok {
				fmt.Fprintln(fs.out, f.Format(*entry))
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			fs.diagnostics.Error(fmt.Sprintf("Error reading file %s: %v", path, err), nil)
			return
		}
	}

	fs.offsets[path] = offset
}

func (fs *FileScanner) ListSources(ctx context.Context) ([]string, error) {
	cfg := fs.lineProcessor.config

	files, err := fs.logFileReader.ListFiles(ctx, cfg.Dir, transport.ListOptions{
		Recursive: cfg.Recursive,
		Services:  cfg.Services,
		OnError: func(path string, err error) {
			fs.diagnostics.Error(fmt.Sprintf("Error listing file %s: %v", path, err), nil)
		},
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
