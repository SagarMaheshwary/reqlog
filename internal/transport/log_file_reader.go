package transport

import (
	"context"
	"io"
)

type LogFileReader interface {
	Open(
		ctx context.Context,
		path string,
	) (io.ReadCloser, error)

	OpenFromOffset(
		ctx context.Context,
		path string,
		offset int64,
	) (io.ReadCloser, error)

	Size(
		ctx context.Context,
		path string,
	) (int64, error)

	ListFiles(
		ctx context.Context,
		dir string,
		opts ListOptions,
	) ([]string, error)
}

type ListOptions struct {
	Recursive bool
	Services  []string
	OnError   func(path string, err error)
}
