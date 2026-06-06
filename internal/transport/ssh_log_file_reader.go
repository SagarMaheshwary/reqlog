package transport

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type SSHLogFileReader struct {
	excutor Executor
}

func (r *SSHLogFileReader) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	// "cat" streams the file content, so it can be used to read line
	// by line without loading the entire file into memory.
	return r.excutor.Run("cat", quoteShellArg(path))
}

func (r *SSHLogFileReader) OpenFromOffset(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	start := offset + 1

	// "tail" -c +<offset> reads the file content starting from the specified byte offset.
	return r.excutor.Run("tail", "-c", fmt.Sprintf("+%d", start), quoteShellArg(path))
}

func (r *SSHLogFileReader) Size(ctx context.Context, path string) (int64, error) {
	out, err := r.excutor.Output("wc", "-c", "<", quoteShellArg(path))
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}

func (r *SSHLogFileReader) ListFiles(ctx context.Context, dir string, opts ListOptions) ([]string, error) {
	matcher := buildServiceMatcher(opts.Services)

	if opts.Recursive {
		return r.listRecursive(dir, matcher)
	}

	return r.listFlat(dir, matcher)
}

func (r *SSHLogFileReader) listRecursive(
	dir string,
	matchesService func(string) bool,
) ([]string, error) {
	out, err := r.excutor.Output("find", dir, "-type", "f")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var files []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if !isLogFile(l) {
			continue
		}

		if l != "" {
			if matchesService(l) {
				files = append(files, l)
			}
		}
	}

	return files, nil
}

func (r *SSHLogFileReader) listFlat(
	dir string,
	matchesService func(string) bool,
) ([]string, error) {
	out, err := r.excutor.Output("find", dir, "-maxdepth", "1", "-type", "f")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var files []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if !isLogFile(l) {
			continue
		}

		if l != "" {
			if matchesService(l) {
				files = append(files, l)
			}
		}
	}

	return files, nil
}
