package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/sftp"
	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

func Run(ctx context.Context, cfg *config.Config) error {
	lp := newLineProcessor(cfg)

	scannerSources, err := scannersForConfig(cfg, lp)
	if err != nil {
		return err
	}

	defer func() {
		for _, ss := range scannerSources {
			ss.Close()
		}
	}()

	var allEntries []domain.LogEntry

	for i, ss := range scannerSources {
		sources, err := resolveSource(ctx, ss.scanner)
		if err != nil {
			return err
		}
		scannerSources[i].sources = append(ss.sources, sources...)

		entries, err := ss.scanner.Scan(ctx, sources)
		if err != nil {
			return err
		}

		allEntries = append(allEntries, entries...)
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

	f := formatter.NewFormatter(&formatter.Opts{
		Entries:    allEntries,
		SearchKeys: cfg.Keys,
		Output:     cfg.Output,
		Context:    cfg.Context,
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
	}, scanner.NewTimeParser())
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

func scannersForConfig(cfg *config.Config, lp *scanner.LineProcessor) ([]scannerSource, error) {
	if cfg.Host == "" {
		scn, err := scanner.New(&scanner.FactoryOpts{
			Source:        cfg.Source,
			LineProcessor: lp,
			FS:            transport.NewFileSystem(nil),
			Executor:      transport.NewExecutor(nil),
		})
		if err != nil {
			return nil, err
		}

		return []scannerSource{{scanner: scn}}, nil
	}

	hosts := strings.Split(cfg.Host, ",")
	scanners := make([]scannerSource, 0, len(hosts))

	for _, host := range hosts {
		h, ok := cfg.Config.Hosts[host]
		if !ok {
			return nil, fmt.Errorf("invalid value passed to --host: %s", host)
		}

		sshClient, err := transport.NewSSHClient(h)
		if err != nil {
			return nil, err
		}

		var sftpClient *sftp.Client
		if cfg.Source == config.SourceFile {
			sftpClient, err = sftp.NewClient(sshClient)
			if err != nil {
				return nil, err
			}
		}

		scn, err := scanner.New(&scanner.FactoryOpts{
			Source:        cfg.Source,
			LineProcessor: lp,
			FS:            transport.NewFileSystem(sftpClient),
			Executor:      transport.NewExecutor(sshClient),
			Host:          host,
		})
		if err != nil {
			return nil, err
		}

		scanners = append(scanners, scannerSource{
			scanner:    scn,
			sshClient:  sshClient,
			sftpClient: sftpClient,
			sources:    make([]string, 0),
		})
	}

	return scanners, nil
}
