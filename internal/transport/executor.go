package transport

import (
	"io"
)

type Executor interface {
	Run(name string, args ...string) (io.ReadCloser, error)
	RunCombined(name string, args ...string) (io.ReadCloser, error)

	Output(name string, args ...string) ([]byte, error)
}
