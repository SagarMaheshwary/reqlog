package parser

import "testing"

func TestNewParser(t *testing.T) {
	tests := []struct {
		name    string
		input   ParserType
		wantErr bool
		check   func(t *testing.T, p LogParser)
	}{
		{
			name:  "text parser",
			input: TypeText,
			check: func(t *testing.T, p LogParser) {
				if _, ok := p.(TextParser); !ok {
					t.Fatalf("expected TextParser, got %T", p)
				}
			},
		},
		{
			name:  "json parser",
			input: TypeJSON,
			check: func(t *testing.T, p LogParser) {
				if _, ok := p.(JSONParser); !ok {
					t.Fatalf("expected JSONParser, got %T", p)
				}
			},
		},
		{
			name:    "unknown parser",
			input:   ParserType("xml"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if p != nil {
					t.Fatalf("expected nil parser, got %T", p)
				}
				if err.Error() != "unknown parser type" {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p == nil {
				t.Fatal("expected parser, got nil")
			}

			if tt.check != nil {
				tt.check(t, p)
			}
		})
	}
}
