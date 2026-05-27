package transport

import (
	"context"
	"io"
	"io/fs"
)

type FileSystem interface {
	Open(ctx context.Context, path string) (File, error)
	ListFiles(ctx context.Context, dir string, opts ListOptions) ([]string, error)
}

type File interface {
	io.Reader
	io.Seeker
	io.Closer
	Stat() (fs.FileInfo, error)
}

type ListOptions struct {
	Recursive bool
	Services  []string
	OnError   func(path string, err error)
}
