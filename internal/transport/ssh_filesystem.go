package transport

import (
	"context"
	"path/filepath"

	"github.com/pkg/sftp"
)

type SSHFileSystem struct {
	client *sftp.Client
}

func (fs *SSHFileSystem) Open(ctx context.Context, path string) (File, error) {
	return fs.client.Open(path)
}

func (fs *SSHFileSystem) ListFiles(ctx context.Context, dir string, opts ListOptions) ([]string, error) {
	matcher := buildServiceMatcher(opts.Services)

	if opts.Recursive {
		return fs.listRecursive(dir, matcher, opts)
	}

	return fs.listFlat(dir, matcher)
}

func (fs *SSHFileSystem) listRecursive(
	dir string,
	matchesService func(string) bool,
	opts ListOptions,
) ([]string, error) {
	files := make([]string, 0, 16)

	var walk func(string) error

	walk = func(current string) error {
		entries, err := fs.client.ReadDir(current)
		if err != nil {
			if opts.OnError != nil {
				opts.OnError(current, err)
			}
			return nil // continue
		}

		for _, entry := range entries {
			path := filepath.Join(current, entry.Name())

			if entry.IsDir() {
				if err := walk(path); err != nil {
					return err
				}
				continue
			}

			name := entry.Name()

			if !isLogFile(name) {
				continue
			}

			if !matchesService(name) {
				continue
			}

			files = append(files, path)
		}

		return nil
	}

	err := walk(dir)

	return files, err
}

func (fs *SSHFileSystem) listFlat(
	dir string,
	matchesService func(string) bool,
) ([]string, error) {
	entries, err := fs.client.ReadDir(dir)
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
