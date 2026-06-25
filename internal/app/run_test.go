package app

import (
	"context"
	"errors"
	"testing"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
)

type fakeScanner struct {
	sources []string
	err     error
}

func (f *fakeScanner) ListSources(context.Context) ([]string, error) {
	return f.sources, f.err
}

func (f *fakeScanner) Scan(context.Context, []string) ([]domain.LogEntry, error) {
	return nil, nil
}

func (f *fakeScanner) Follow(context.Context, []string, formatter.LogFormatter) {
}

func TestResolveSource(t *testing.T) {
	tests := []struct {
		name    string
		scanner scanner.Scanner
		want    []string
		wantErr bool
	}{
		{
			name: "success",
			scanner: &fakeScanner{
				sources: []string{"api.log"},
			},
			want: []string{"api.log"},
		},
		{
			name: "scanner error",
			scanner: &fakeScanner{
				err: errors.New("failed"),
			},
			wantErr: true,
		},
		{
			name: "no sources",
			scanner: &fakeScanner{
				sources: []string{},
			},
			wantErr: true,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveSource(ctx, tt.scanner)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("expected %v got %v", tt.want, got)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("expected %v got %v", tt.want, got)
				}
			}
		})
	}
}

func TestScannersForConfig_Local(t *testing.T) {
	cfg := &config.Config{
		Source: config.SourceFile,
		Host:   "",
	}

	lp := newLineProcessor(cfg)

	got, err := scannersForConfig(cfg, lp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 scanner got %d", len(got))
	}

	if got[0].scanner == nil {
		t.Fatal("expected scanner")
	}
}

func TestScannersForConfig_InvalidHost(t *testing.T) {
	cfg := &config.Config{
		Host: "missing",
		Config: &config.SSH{
			Hosts: map[string]config.Host{},
		},
	}

	lp := newLineProcessor(cfg)

	_, err := scannersForConfig(cfg, lp, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewLineProcessor(t *testing.T) {
	cfg := &config.Config{
		Dir:         "/tmp/logs",
		SearchValue: "error",
		IgnoreCase:  true,
		Keys:        []string{"msg"},
		Limit:       100,
		Recursive:   true,
		Services:    []string{"api"},
		Latest:      true,
		Context:     2,
		Format:      "json",
	}

	lp := newLineProcessor(cfg)

	if lp == nil {
		t.Fatal("expected line processor, got nil")
	}
}
