package scanner

import (
	"fmt"
	"os"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/docker"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type FactoryOpts struct {
	Source        config.Source
	LineProcessor *LineProcessor
	FS            transport.FileSystem
	Executor      transport.Executor
	Host          string
}

func New(opts *FactoryOpts) (Scanner, error) {
	switch opts.Source {
	case config.SourceFile:
		return NewFileScanner(&FileScannerOpts{
			LineProcessor:  opts.LineProcessor,
			FollowInterval: time.Second,
			Out:            os.Stdout,
			ErrOut:         os.Stderr,
			Now:            time.Now(),
			FS:             opts.FS,
			Host:           opts.Host,
		}), nil
	case config.SourceDocker:
		return NewDockerScanner(&DockerScannerOpts{
			LineProcessor: opts.LineProcessor,
			dockerClient:  docker.NewDockerCLIClient(opts.Executor),
			Out:           os.Stdout,
			ErrOut:        os.Stderr,
			Now:           time.Now(),
			Host:          opts.Host,
		}), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
