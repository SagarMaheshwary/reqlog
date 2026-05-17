package scanner

import (
	"fmt"
	"os"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/docker"
)

func New(source config.Source, lp *LineProcessor) (Scanner, error) {
	switch source {
	case config.SourceFile:
		return NewFileScanner(lp, time.Second, os.Stdout, os.Stderr, time.Now()), nil
	case config.SourceDocker:
		return NewDockerScanner(lp, docker.NewDockerCLIClient(), os.Stdout, os.Stderr, time.Now()), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
