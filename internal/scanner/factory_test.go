package scanner

import "testing"

func TestNewScanner(t *testing.T) {
	cfg := &ScanConfig{}
	lp := NewLineProcessor(cfg, NewTimeParser())

	tests := []struct {
		name    string
		source  string
		wantErr bool
		check   func(t *testing.T, s Scanner)
	}{
		{
			name:   "file source",
			source: "file",
			check: func(t *testing.T, s Scanner) {
				if _, ok := s.(*FileScanner); !ok {
					t.Fatalf("expected *FileScanner")
				}
			},
		},
		{
			name:   "docker source",
			source: "docker",
			check: func(t *testing.T, s Scanner) {
				if _, ok := s.(*DockerScanner); !ok {
					t.Fatalf("expected *DockerScanner")
				}
			},
		},
		{
			name:    "unknown source",
			source:  "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.source, lp)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s == nil {
				t.Fatalf("expected scanner, got nil")
			}

			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}
