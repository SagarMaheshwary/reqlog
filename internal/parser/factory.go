package parser

import "fmt"

type ParserType string

const (
	TypeText   ParserType = "text"
	TypeJSON   ParserType = "json"
	TypeDocker ParserType = "docker"
)

func NewParser(t ParserType) (LogParser, error) {
	switch t {
	case TypeText:
		return TextParser{}, nil
	case TypeJSON:
		return JSONParser{}, nil
	case TypeDocker:
		// future
		return nil, fmt.Errorf("docker parser not implemented yet")
	default:
		return nil, fmt.Errorf("unknown parser type")
	}
}
