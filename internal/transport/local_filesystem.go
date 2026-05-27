package transport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type LocalFileSystem struct{}

func (fs *LocalFileSystem) Open(ctx context.Context, path string) (File, error) {
	return os.Open(path)
}

func (fs *LocalFileSystem) ListFiles(ctx context.Context, dir string, opts ListOptions) ([]string, error) {
	matcher := buildServiceMatcher(opts.Services)

	if opts.Recursive {
		return fs.listRecursive(dir, matcher, opts)
	}

	return fs.listFlat(dir, matcher)
}

func (fs *LocalFileSystem) listRecursive(
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

func (fs *LocalFileSystem) listFlat(
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
