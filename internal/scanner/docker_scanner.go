package scanner

import (
	"bufio"
	"container/heap"
	"fmt"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/docker"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

type DockerScanner struct {
	lineProcessor *LineProcessor
	dockerClient  docker.CLIDockerClient
}

func NewDockerScanner(lp *LineProcessor, client docker.CLIDockerClient) *DockerScanner {
	return &DockerScanner{lineProcessor: lp, dockerClient: client}
}

func (ds *DockerScanner) Scan(containers []string) []domain.LogEntry {
	cfg := ds.lineProcessor.config

	var h entryHeap
	var results []domain.LogEntry
	sinceTime := parseSince(ds.lineProcessor.config.Since)

	if cfg.Limit > 0 {
		heap.Init(&h)
	}

	for _, container := range containers {
		reader, err := ds.dockerClient.Logs(container, false, cfg.Since)
		if err != nil {
			logScanError(container, err)
			continue
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			entry, ok := ds.lineProcessor.ProcessLine(line, container)
			if ok && passesSince(entry, sinceTime) {
				ds.lineProcessor.AddEntry(*entry, &results, &h)
			}
		}
	}

	if cfg.Limit > 0 {
		results = drainHeap(&h)
	}

	return results
}

func (ds *DockerScanner) Follow(containers []string) {
	f := formatter.NewFormatter([]domain.LogEntry{})

	done := make(chan struct{})

	for _, container := range containers {
		go func(container string) {
			reader, err := ds.dockerClient.Logs(container, true, "")
			if err != nil {
				logScanError(container, err)
				return
			}
			defer reader.Close()

			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				entry, ok := ds.lineProcessor.ProcessLine(line, container)
				if !ok {
					continue
				}
				fmt.Println(f.Format(*entry))
			}
		}(container)
	}

	<-done
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
