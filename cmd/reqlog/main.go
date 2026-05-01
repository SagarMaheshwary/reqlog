package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
)

var (
	dir         = flag.String("dir", "./logs", "directory containing log files")
	ignoreCase  = flag.Bool("ignore-case", false, "perform case-insensitive search")
	limit       = flag.Int("limit", 0, "limit number of results (returns newest N matches)")
	jsonMode    = flag.Bool("json", false, "parse logs as JSON (one JSON object per line)")
	follow      = flag.Bool("follow", false, "follow logs in real time (like tail -f)")
	key         = flag.String("key", "", "field key to match (e.g. request_id, trace_id, event_key)")
	since       = flag.String("since", "", "filter logs newer than: duration (5m, 1h), RFC3339 timestamp, or YYYY-MM-DD (UTC)")
	recursive   = flag.Bool("recursive", true, "scan directories recursively")
	service     = flag.String("service", "", "filter by service name (comma-separated, e.g. order-service,inventory-service)")
	source      = flag.String("source", "file", `log source backend: "file", "docker"`)
	showVersion = flag.Bool("version", false, "print version and exit")
)

var version = "dev"

var cliInfo = `Usage:
  reqlog [flags] <search_value>

Description:
  Search logs by exact key=value matching across log files.

Examples:
  # Basic search
  reqlog abc123

  # Search in a specific directory
  reqlog --dir ./logs abc123

  # Case-insensitive search
  reqlog --ignore-case abc123

  # Limit results (latest 10 matches)
  reqlog --limit 10 abc123

  # Filter by key
  reqlog --key request_id abc123
  reqlog --key event_key order.created

  # JSON logs
  reqlog --json --key trace_id trace-1

  # Filter recent logs
  reqlog --since 5m abc123

  # Filter specific services
  reqlog --service order-service,inventory-service abc123

  # Non-recursive scan
  reqlog --recursive=false abc123

  # Follow logs (tail mode)
  reqlog --follow abc123

  # Combined example (real-world usage)
  reqlog \
    --dir ./logs \
    --service api-gateway,order-service \
    --key event_key \
    --since 10m \
    --limit 20 \
    order.created`

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("reqlog version %s\n", cliVersion())
		return
	}

	if flag.NArg() < 1 {
		if flag.NArg() < 1 {
			fmt.Println(cliInfo)
			os.Exit(1)
		}
	}

	SearchValue := flag.Arg(0)

	keys := parser.DefaultKeys
	if *key != "" {
		keys = []string{*key}
	}

	services := []string{}
	if *service != "" {
		services = strings.Split(*service, ",")
	}

	var parserType = parser.TypeText
	if *jsonMode {
		parserType = parser.TypeJSON
	}

	p, err := parser.New(parserType)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &scanner.ScanConfig{
		Dir:         *dir,
		SearchValue: SearchValue,
		IgnoreCase:  *ignoreCase,
		Keys:        keys,
		Since:       *since,
		Limit:       *limit,
		Recursive:   *recursive,
		Services:    services,
	}
	scn, err := scanner.New(*source, scanner.NewLineProcessor(cfg, p))
	if err != nil {
		log.Fatal(err)
	}

	sources, err := scn.ListSources()
	if err != nil {
		log.Fatal(err)
	}

	if len(sources) == 0 {
		log.Fatal("no matching sources found")
	}

	entries := scn.Scan(sources)
	f := formatter.NewFormatter(entries, keys)

	printEntries(f, entries)

	if *follow {
		scn.Follow(context.Background(), sources, f)
	}
}

func cliVersion() string {
	if version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}

func printEntries(f formatter.LogFormatter, entries []domain.LogEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	for _, e := range entries {
		fmt.Println(f.Format(e))
	}
}
