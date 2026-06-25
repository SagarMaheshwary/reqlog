package scanner

import (
	"fmt"
	"os"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/diagnostics"
	"github.com/sagarmaheshwary/reqlog/internal/docker"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type FactoryOpts struct {
	Source        config.Source
	LineProcessor *LineProcessor
	Executor      transport.Executor
	LogFileReader transport.LogFileReader
	Host          string
	Diagnostics   *diagnostics.Diagnostics
}

func New(opts *FactoryOpts) (Scanner, error) {
	switch opts.Source {
	case config.SourceFile:
		return NewFileScanner(&FileScannerOpts{
			LineProcessor:  opts.LineProcessor,
			FollowInterval: time.Second,
			Out:            os.Stdout,
			Now:            time.Now(),
			LogFileReader:  opts.LogFileReader,
			Host:           opts.Host,
			Diagnostics:    opts.Diagnostics,
		}), nil
	case config.SourceDocker:
		return NewDockerScanner(&DockerScannerOpts{
			LineProcessor: opts.LineProcessor,
			DockerClient:  docker.NewDockerCLIClient(opts.Executor),
			Out:           os.Stdout,
			Now:           time.Now(),
			Host:          opts.Host,
			Diagnostics:   opts.Diagnostics,
		}), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
