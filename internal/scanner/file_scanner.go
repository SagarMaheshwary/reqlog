package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type FileScanner struct {
	offsets        map[string]int64
	lineProcessor  *LineProcessor
	followInterval time.Duration
	out            io.Writer
	errOut         io.Writer
	now            time.Time
	fsys           transport.FileSystem
	host           string
}

type FileScannerOpts struct {
	LineProcessor  *LineProcessor
	FollowInterval time.Duration
	Out            io.Writer
	ErrOut         io.Writer
	Now            time.Time
	FS             transport.FileSystem
	Host           string
}

func NewFileScanner(opts *FileScannerOpts) *FileScanner {
	return &FileScanner{
		offsets:        make(map[string]int64),
		lineProcessor:  opts.LineProcessor,
		followInterval: opts.FollowInterval,
		out:            opts.Out,
		errOut:         opts.ErrOut,
		now:            opts.Now,
		fsys:           opts.FS,
		host:           opts.Host,
	}
}

func (fs *FileScanner) Scan(ctx context.Context, files []string) ([]domain.LogEntry, error) {
	cfg := fs.lineProcessor.config

	collector, err := NewEntryCollector(cfg, fs.now)
	if err != nil {
		return nil, err
	}

	for _, path := range files {
		file, err := fs.fsys.Open(ctx, path)
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

			stat, err := file.Stat()
			if err == nil {
				offset = stat.Size()
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
				fs.processFile(ctx, path, f)
			}
		}
	}
}

func (fs *FileScanner) processFile(ctx context.Context, path string, f formatter.LogFormatter) {
	file, err := fs.fsys.Open(ctx, path)
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

func (fs *FileScanner) ListSources(ctx context.Context) ([]string, error) {
	cfg := fs.lineProcessor.config

	files, err := fs.fsys.ListFiles(ctx, cfg.Dir, transport.ListOptions{
		Recursive: cfg.Recursive,
		Services:  cfg.Services,
		OnError: func(path string, err error) {
			logScanError(fs.errOut, path, err)
		},
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
