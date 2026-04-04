package scanner

import "testing"

func TestLineProcessor_ProcessLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		search     string
		ignoreCase bool

		parser mockParser
		wantOK bool
	}{
		{
			name:       "match normal case",
			line:       "id=123 status=ok",
			search:     "123",
			ignoreCase: false,
			parser:     mockParser{extractOK: true},
			wantOK:     true,
		},
		{
			name:       "no match in line (prefilter)",
			line:       "id=456 status=ok",
			search:     "123",
			ignoreCase: false,
			parser:     mockParser{extractOK: true},
			wantOK:     false,
		},
		{
			name:       "ignore case match",
			line:       "ID=ABC status=ok",
			search:     "abc",
			ignoreCase: true,
			parser:     mockParser{extractOK: true, extractedValue: "ABC"},
			wantOK:     true,
		},
		{
			name:       "ignore case prefilter fail",
			line:       "id=xyz",
			search:     "ABC",
			ignoreCase: true,
			parser:     mockParser{extractOK: true},
			wantOK:     false,
		},
		{
			name:       "extract field fails",
			line:       "id=123",
			search:     "123",
			ignoreCase: false,
			parser:     mockParser{extractOK: false},
			wantOK:     false,
		},
		{
			name:       "match fails after extract",
			line:       "id=999",
			search:     "123",
			ignoreCase: false,
			parser: mockParser{
				extractOK:      true,
				extractedValue: "999",
			},
			wantOK: false,
		},
		{
			name:       "parse fails",
			line:       "id=123",
			search:     "123",
			ignoreCase: false,
			parser: mockParser{
				extractOK: true,
				parseErr:  true,
			},
			wantOK: false,
		},
		{
			name:       "trims newline",
			line:       "id=123\n",
			search:     "123",
			ignoreCase: false,
			parser:     mockParser{extractOK: true},
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ScanConfig{
				SearchValue: tt.search,
				IgnoreCase:  tt.ignoreCase,
				Keys:        []string{"id"},
			}

			lp := NewLineProcessor(cfg, tt.parser)

			entry, ok := lp.ProcessLine(tt.line, "svc")

			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}

			if tt.wantOK && entry == nil {
				t.Fatal("expected entry, got nil")
			}

			if !tt.wantOK && entry != nil {
				t.Fatal("expected nil entry")
			}
		})
	}
}
