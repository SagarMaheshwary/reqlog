package app

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
)

type fakeScanner struct {
	sources []string
	err     error

	entries  []domain.LogEntry
	seekErr  error
	followed []string // sources returned by SeekToEnd; defaults to all

	scanned []string
	seeked  []string
}

func (f *fakeScanner) ListSources(context.Context) ([]string, error) {
	return f.sources, f.err
}

func (f *fakeScanner) Scan(_ context.Context, sources []string) ([]domain.LogEntry, error) {
	f.scanned = sources
	return f.entries, nil
}

func (f *fakeScanner) SeekToEnd(_ context.Context, sources []string) ([]string, error) {
	f.seeked = sources
	if f.followed != nil {
		return f.followed, f.seekErr
	}
	return sources, f.seekErr
}

func (f *fakeScanner) ServiceName(source string) string {
	return strings.TrimSuffix(source, ".log")
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

func TestNewLineProcessor_LiveOnlyMatchesAllLines(t *testing.T) {
	line := "2024-03-10T12:00:00Z request arrived status=ok"

	tests := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{
			name: "follow without search value matches any line",
			cfg:  &config.Config{Follow: true, Keys: config.DefaultKeys, Format: config.FormatAuto},
			want: true,
		},
		{
			name: "follow with search value still filters",
			cfg:  &config.Config{Follow: true, SearchValue: "abc123", Keys: config.DefaultKeys, Format: config.FormatAuto},
			want: false,
		},
		{
			name: "search without follow still filters",
			cfg:  &config.Config{SearchValue: "abc123", Keys: config.DefaultKeys, Format: config.FormatAuto},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := newLineProcessor(tt.cfg)

			_, ok := lp.ProcessLine(line, "svc", "")
			if ok != tt.want {
				t.Fatalf("expected ok=%v got %v", tt.want, ok)
			}
		})
	}
}

func TestScanSources(t *testing.T) {
	entries := []domain.LogEntry{{Service: "api", Message: "existing"}}

	t.Run("search mode scans existing logs", func(t *testing.T) {
		fs := &fakeScanner{sources: []string{"api.log"}, entries: entries}
		scannerSources := []scannerSource{{scanner: fs}}

		got, err := scanSources(context.Background(), &config.Config{SearchValue: "abc123", Follow: true}, scannerSources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(got, entries) {
			t.Fatalf("expected %v got %v", entries, got)
		}

		if !reflect.DeepEqual(fs.scanned, []string{"api.log"}) {
			t.Fatalf("expected Scan to be called with %v got %v", []string{"api.log"}, fs.scanned)
		}

		if fs.seeked != nil {
			t.Fatalf("expected SeekToEnd not to be called, got %v", fs.seeked)
		}

		if !reflect.DeepEqual(scannerSources[0].sources, []string{"api.log"}) {
			t.Fatalf("expected resolved sources %v got %v", []string{"api.log"}, scannerSources[0].sources)
		}
	})

	t.Run("live-only mode skips existing logs", func(t *testing.T) {
		fs1 := &fakeScanner{sources: []string{"api.log"}, entries: entries}
		fs2 := &fakeScanner{sources: []string{"db.log"}, entries: entries}
		scannerSources := []scannerSource{{scanner: fs1}, {scanner: fs2}}

		got, err := scanSources(context.Background(), &config.Config{Follow: true}, scannerSources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 0 {
			t.Fatalf("expected no entries got %v", got)
		}

		for i, fs := range []*fakeScanner{fs1, fs2} {
			if fs.scanned != nil {
				t.Fatalf("scanner %d: expected Scan not to be called, got %v", i, fs.scanned)
			}

			if !reflect.DeepEqual(fs.seeked, fs.sources) {
				t.Fatalf("scanner %d: expected SeekToEnd to be called with %v got %v", i, fs.sources, fs.seeked)
			}

			if !reflect.DeepEqual(scannerSources[i].sources, fs.sources) {
				t.Fatalf("scanner %d: expected resolved sources %v got %v", i, fs.sources, scannerSources[i].sources)
			}
		}
	})

	t.Run("live-only mode follows only seeked sources", func(t *testing.T) {
		fs := &fakeScanner{
			sources:  []string{"api.log", "broken.log"},
			followed: []string{"api.log"},
		}
		scannerSources := []scannerSource{{scanner: fs}}

		if _, err := scanSources(context.Background(), &config.Config{Follow: true}, scannerSources); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(scannerSources[0].sources, []string{"api.log"}) {
			t.Fatalf("expected followed sources %v got %v", []string{"api.log"}, scannerSources[0].sources)
		}
	})

	t.Run("live-only mode returns seek error", func(t *testing.T) {
		fs := &fakeScanner{sources: []string{"api.log"}, seekErr: errors.New("seek failed")}

		_, err := scanSources(context.Background(), &config.Config{Follow: true}, []scannerSource{{scanner: fs}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns resolve error", func(t *testing.T) {
		fs := &fakeScanner{sources: []string{}}

		_, err := scanSources(context.Background(), &config.Config{Follow: true}, []scannerSource{{scanner: fs}})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestFormatterEntries(t *testing.T) {
	entries := []domain.LogEntry{{Service: "reqlog", Message: "diagnostic"}}

	scannerSources := []scannerSource{
		{scanner: &fakeScanner{}, sources: []string{"api.log", "inventory-service.log"}},
		{scanner: &fakeScanner{}, sources: []string{"db.log"}, host: "srv1"},
	}

	t.Run("search mode uses only entries", func(t *testing.T) {
		got := formatterEntries(&config.Config{SearchValue: "abc123", Follow: true}, entries, scannerSources)

		if !reflect.DeepEqual(got, entries) {
			t.Fatalf("expected %v got %v", entries, got)
		}
	})

	t.Run("live-only mode adds a placeholder per followed source", func(t *testing.T) {
		got := formatterEntries(&config.Config{Follow: true}, entries, scannerSources)

		want := []domain.LogEntry{
			{Service: "reqlog", Message: "diagnostic"},
			{Service: "api"},
			{Service: "inventory-service"},
			{Service: "db", Host: "srv1"},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected %v got %v", want, got)
		}

		if len(entries) != 1 {
			t.Fatalf("expected input entries to be left unchanged, got %v", entries)
		}
	})

	t.Run("live-only pretty output is aligned across sources", func(t *testing.T) {
		f := formatter.NewFormatter(&formatter.Opts{
			Entries: formatterEntries(&config.Config{Follow: true}, nil, scannerSources),
			Output:  config.OutputPretty,
		})

		ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		ts := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)

		var want int
		for i, e := range []domain.LogEntry{
			{Timestamp: ts, Service: "api", Message: "a"},
			{Timestamp: ts, Service: "inventory-service", Message: "b"},
			{Timestamp: ts, Service: "db", Host: "srv1", Message: "c"},
		} {
			line := ansi.ReplaceAllString(f.Format(e), "")
			got := strings.Index(line, " | ")

			if i == 0 {
				want = got
				continue
			}

			if got != want {
				t.Fatalf("expected separator at column %d, got %d in %q", want, got, line)
			}
		}
	})
}
