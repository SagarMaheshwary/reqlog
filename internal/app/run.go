package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/diagnostics"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
	"golang.org/x/crypto/ssh"
)

func Run(ctx context.Context, cfg *config.Config) error {
	lp := newLineProcessor(cfg)
	dg := diagnostics.NewDiagnostics()

	scannerSources, err := scannersForConfig(cfg, lp, dg)
	if err != nil {
		if cfg.Verbose {
			printDiagnostics(cfg, dg)
		}
		return err
	}

	defer func() {
		for _, ss := range scannerSources {
			ss.Close()
		}
	}()

	allEntries, err := scanSources(ctx, cfg, scannerSources)
	if err != nil {
		return err
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
	})

	if cfg.Limit > 0 && len(allEntries) > cfg.Limit {
		start := 0
		end := cfg.Limit

		if cfg.Latest {
			start = len(allEntries) - cfg.Limit
			end = len(allEntries)
		}

		allEntries = allEntries[start:end]
	}

	if cfg.Verbose {
		allEntries = append(allEntries, dg.Entries()...)
	}

	f := formatter.NewFormatter(&formatter.Opts{
		Entries:    formatterEntries(cfg, allEntries, scannerSources),
		SearchKeys: cfg.Keys,
		Output:     cfg.Output,
		Context:    cfg.Context,
		Fields:     cfg.Fields,
	})

	for _, e := range allEntries {
		fmt.Println(f.Format(e))
	}

	if cfg.Follow {
		for _, ss := range scannerSources {
			go ss.scanner.Follow(ctx, ss.sources, f)
		}
		<-ctx.Done()
	}

	return nil
}

func printDiagnostics(cfg *config.Config, dg *diagnostics.Diagnostics) {
	entries := dg.Entries()

	f := formatter.NewFormatter(&formatter.Opts{
		Entries: entries,
		Output:  cfg.Output,
		Fields:  cfg.Fields,
	})

	for _, e := range entries {
		fmt.Println(f.Format(e))
	}
}

func newLineProcessor(cfg *config.Config) *scanner.LineProcessor {
	return scanner.NewLineProcessor(&scanner.Config{
		Dir:         cfg.Dir,
		SearchValue: cfg.SearchValue,
		IgnoreCase:  cfg.IgnoreCase,
		Keys:        cfg.Keys,
		Since:       cfg.Since,
		Limit:       cfg.Limit,
		Recursive:   cfg.Recursive,
		Services:    cfg.Services,
		Latest:      cfg.Latest,
		Context:     cfg.Context,
		Format:      cfg.Format,
		MatchAll:    cfg.LiveOnly(),
	}, scanner.NewTimeParser())
}

// scanSources resolves the sources of every scanner and collects existing
// matches. In live-only mode existing content is skipped instead, so only
// newly appended lines are emitted once following starts.
func scanSources(
	ctx context.Context,
	cfg *config.Config,
	scannerSources []scannerSource,
) ([]domain.LogEntry, error) {
	var allEntries []domain.LogEntry

	for i, ss := range scannerSources {
		sources, err := resolveSource(ctx, ss.scanner)
		if err != nil {
			return nil, err
		}
		scannerSources[i].sources = append(ss.sources, sources...)

		if cfg.LiveOnly() {
			followable, err := ss.scanner.SeekToEnd(ctx, sources)
			if err != nil {
				return nil, err
			}
			scannerSources[i].sources = followable
			continue
		}

		entries, err := ss.scanner.Scan(ctx, sources)
		if err != nil {
			return nil, err
		}

		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// formatterEntries returns the entries used to size the formatter's service
// column. In live-only mode there are no initial results to measure, so a
// placeholder entry is added for every followed source.
func formatterEntries(
	cfg *config.Config,
	entries []domain.LogEntry,
	scannerSources []scannerSource,
) []domain.LogEntry {
	if !cfg.LiveOnly() {
		return entries
	}

	out := append([]domain.LogEntry(nil), entries...)
	for _, ss := range scannerSources {
		for _, source := range ss.sources {
			out = append(out, domain.LogEntry{
				Service: ss.scanner.ServiceName(source),
				Host:    ss.host,
			})
		}
	}

	return out
}

func resolveSource(
	ctx context.Context,
	scanner scanner.Scanner,
) ([]string, error) {
	sources, err := scanner.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no matching sources found")
	}

	return sources, nil
}

func scannersForConfig(
	cfg *config.Config,
	lp *scanner.LineProcessor,
	dg *diagnostics.Diagnostics,
) ([]scannerSource, error) {
	if cfg.Host == "" {
		scn, err := scanner.New(&scanner.FactoryOpts{
			Source:        cfg.Source,
			LineProcessor: lp,
			Executor:      transport.NewExecutor(nil),
			LogFileReader: transport.NewLogFileReader(nil),
			Diagnostics:   dg,
		})
		if err != nil {
			return nil, err
		}

		return []scannerSource{{scanner: scn}}, nil
	}

	hosts := strings.Split(cfg.Host, ",")
	scanners := make([]scannerSource, 0, len(hosts))

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, host := range hosts {
		h, ok := cfg.Config.Hosts[host]
		if !ok {
			keys := make([]string, 0, len(cfg.Config.Hosts))
			for k := range cfg.Config.Hosts {
				keys = append(keys, k)
			}

			return nil, fmt.Errorf("invalid value passed to --host: %s, available hosts are %v", host, keys)
		}

		wg.Add(1)

		go func(host string, h config.Host) {
			defer wg.Done()

			sshClient, err := transport.NewSSHClient(h, ssh.Dial)
			if err != nil {
				dg.Error(fmt.Sprintf("Error creating SSH client for host %s: %v", host, err), nil)
				return
			}

			executor := transport.NewExecutor(sshClient)
			scn, err := scanner.New(&scanner.FactoryOpts{
				Source:        cfg.Source,
				LineProcessor: lp,
				Executor:      executor,
				LogFileReader: transport.NewLogFileReader(executor),
				Host:          host,
				Diagnostics:   dg,
			})
			if err != nil {
				sshClient.Close()
				dg.Error(fmt.Sprintf("Error creating scanner for host %s: %v", host, err), nil)
				return
			}
			mu.Lock()
			scanners = append(scanners, scannerSource{
				scanner:   scn,
				sshClient: sshClient,
				host:      host,
				sources:   make([]string, 0),
			})
			mu.Unlock()
		}(host, h)
	}

	wg.Wait()

	if len(scanners) == 0 {
		return nil, errors.New("failed to connect to any configured host")
	}

	return scanners, nil
}
