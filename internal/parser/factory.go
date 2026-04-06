package parser

import "fmt"

type ParserType string

const (
	TypeText ParserType = "text"
	TypeJSON ParserType = "json"
)

func New(t ParserType) (LogParser, error) {
	switch t {
	case TypeText:
		return TextParser{}, nil
	case TypeJSON:
		return JSONParser{}, nil
	default:
		return nil, fmt.Errorf("unknown parser type")
	}
}
