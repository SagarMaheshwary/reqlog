package transport

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalLogFileReader struct{}

func (r *LocalLogFileReader) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (r *LocalLogFileReader) OpenFromOffset(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

func (r *LocalLogFileReader) Size(ctx context.Context, path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (r *LocalLogFileReader) ListFiles(ctx context.Context, dir string, opts ListOptions) ([]string, error) {
	matcher := buildServiceMatcher(opts.Services)

	if opts.Recursive {
		return r.listRecursive(dir, matcher, opts)
	}

	return r.listFlat(dir, matcher)
}

func (r *LocalLogFileReader) listRecursive(
	dir string,
	matchesService func(string) bool,
	opts ListOptions,
) ([]string, error) {
	files := make([]string, 0, 16)

	err := filepath.WalkDir(
		dir,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if opts.OnError != nil {
					opts.OnError(path, err)
				}
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

func (r *LocalLogFileReader) listFlat(
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
