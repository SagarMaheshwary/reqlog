package parser

import "testing"

func TestNewParser_Text(t *testing.T) {
	p, err := NewParser(TypeText)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p == nil {
		t.Fatal("expected parser, got nil")
	}

	if _, ok := p.(TextParser); !ok {
		t.Fatalf("expected TextParser, got %T", p)
	}
}

func TestNewParser_JSON(t *testing.T) {
	p, err := NewParser(TypeJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p == nil {
		t.Fatal("expected parser, got nil")
	}

	if _, ok := p.(JSONParser); !ok {
		t.Fatalf("expected JSONParser, got %T", p)
	}
}

func TestNewParser_UnknownType(t *testing.T) {
	p, err := NewParser(ParserType("xml"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if p != nil {
		t.Fatalf("expected nil parser, got %T", p)
	}

	if err.Error() != "unknown parser type" {
		t.Fatalf("unexpected error: %v", err)
	}
}
