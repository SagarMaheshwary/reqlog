package scanner

import (
	"fmt"

	"github.com/sagarmaheshwary/reqlog/internal/docker"
)

func New(source string, lp *LineProcessor) (Scanner, error) {
	switch source {
	case "file":
		return NewFileScanner(lp), nil
	case "docker":
		return NewDockerScanner(lp, docker.NewCLIDockerClient()), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
