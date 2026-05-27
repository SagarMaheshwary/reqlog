package scanner

import (
	"testing"

	"github.com/sagarmaheshwary/reqlog/internal/config"
)

func TestNew(t *testing.T) {
	lp := &LineProcessor{}

	tests := []struct {
		name      string
		source    config.Source
		wantType  any
		wantError bool
	}{
		{
			name:     "file scanner",
			source:   config.SourceFile,
			wantType: &FileScanner{},
		},
		{
			name:     "docker scanner",
			source:   config.SourceDocker,
			wantType: &DockerScanner{},
		},
		{
			name:      "unknown source",
			source:    config.Source("invalid"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := New(&FactoryOpts{
				Source:        tt.source,
				LineProcessor: lp,
			})

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.wantType.(type) {
			case *FileScanner:
				if _, ok := scanner.(*FileScanner); !ok {
					t.Fatalf("expected *FileScanner got %T", scanner)
				}

			case *DockerScanner:
				if _, ok := scanner.(*DockerScanner); !ok {
					t.Fatalf("expected *DockerScanner got %T", scanner)
				}
			}
		})
	}
}
