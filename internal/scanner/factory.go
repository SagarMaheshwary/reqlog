package scanner

import "fmt"

func New(source string, lp *LineProcessor) (Scanner, error) {
	switch source {
	case "file":
		return NewFileScanner(lp), nil
	case "docker":
		return NewDockerScanner(lp), nil
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}
