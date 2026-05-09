package scanner

import (
	"fmt"
	"os"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/docker"
)

func New(source string, lp *LineProcessor) (Scanner, error) {
	switch source {
	case "file":
		return NewFileScanner(lp, time.Second, os.Stdout, os.Stderr, time.Now()), nil
	case "docker":
		return NewDockerScanner(lp, docker.NewDockerCLIClient(), os.Stdout, os.Stderr, time.Now()), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
