package scanner

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/docker"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type DockerScanner struct {
	lineProcessor *LineProcessor
	dockerClient  docker.DockerClient
	out           io.Writer
	errOut        io.Writer
}

func NewDockerScanner(lp *LineProcessor,
	client docker.DockerClient,
	out io.Writer,
	errOut io.Writer,
) *DockerScanner {
	return &DockerScanner{
		lineProcessor: lp,
		dockerClient:  client,
		out:           out,
		errOut:        errOut,
	}
}

func (ds *DockerScanner) Scan(containers []string) ([]domain.LogEntry, error) {
	cfg := ds.lineProcessor.config

	var h entryHeap
	var results []domain.LogEntry
	sinceTime, err := parseSince(cfg.Since, time.Now())
	if err != nil {
		return nil, err
	}

	if cfg.Limit > 0 {
		heap.Init(&h)
	}

	for _, container := range containers {
		reader, err := ds.dockerClient.Logs(container, false, cfg.Since)
		if err != nil {
			logScanError(ds.errOut, container, err)
			continue
		}

		func() {
			defer reader.Close()

			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				entry, ok := ds.lineProcessor.ProcessLine(line, container)
				if ok && passesSince(entry, sinceTime) {
					ds.lineProcessor.AddEntry(*entry, &results, &h)
				}
			}
		}()
	}

	if cfg.Limit > 0 {
		results = drainHeap(&h)
	}

	return results, nil
}

func (ds *DockerScanner) Follow(ctx context.Context, containers []string, f formatter.LogFormatter) {
	var wg sync.WaitGroup

	for _, container := range containers {
		wg.Add(1)
		go func(container string) {
			defer wg.Done()

			reader, err := ds.dockerClient.Logs(container, true, "")
			if err != nil {
				logScanError(ds.errOut, container, err)
				return
			}
			defer reader.Close()

			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				entry, ok := ds.lineProcessor.ProcessLine(line, container)
				if ok {
					fmt.Fprintln(ds.out, f.Format(*entry))
				}
			}
		}(container)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (ds *DockerScanner) ListSources() ([]string, error) {
	containers, err := ds.dockerClient.ListContainers()
	if err != nil {
		return nil, err
	}

	exact := map[string]struct{}{}
	prefixes := []string{}

	for _, s := range ds.lineProcessor.config.Services {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if before, ok := strings.CutSuffix(s, "*"); ok {
			prefixes = append(prefixes, before)
		} else {
			exact[s] = struct{}{}
		}
	}

	matchesService := func(name string) bool {
		if len(exact) == 0 && len(prefixes) == 0 {
			return true
		}
		if _, ok := exact[name]; ok {
			return true
		}

		for _, p := range prefixes {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}

	matches := make([]string, 0, len(containers))
	for _, name := range containers {
		if !matchesService(name) {
			continue
		}

		matches = append(matches, name)
	}

	return matches, nil
}
