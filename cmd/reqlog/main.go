package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/parser"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
)

func main() {
	var (
		dir        string
		ignoreCase bool
		limit      int
		jsonMode   bool
		follow     bool
		key        string
		since      string
		recursive  bool
		service    string
	)

	flag.StringVar(&dir, "dir", "./logs", "log directory")
	flag.BoolVar(&ignoreCase, "ignore-case", false, "case insensitive search")
	flag.IntVar(&limit, "limit", 0, "limit output")
	flag.BoolVar(&jsonMode, "json", false, "parse JSON logs")
	flag.BoolVar(&follow, "follow", false, "follow logs (tail)")
	flag.StringVar(&key, "key", "", "request id key (e.g request_id, trace_id)")
	flag.StringVar(&since, "since", "", "filter logs (e.g 5m, 1h)")
	flag.BoolVar(&recursive, "recursive", true, "")
	flag.StringVar(&service, "service", "", "filter by service name")

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println(`Usage: 
 reqlog [flags] <request_id>
Examples:
 reqlog abc123
 reqlog -dir ./logs abc123
 reqlog -dir ./logs -json json-abc
 reqlog -dir ./logs -json -key trace_id json-trace-1
 reqlog -dir ./logs -since 5m abc123`)
		os.Exit(1)
	}

	SearchValue := flag.Arg(0)

	var parserType = parser.TypeText
	if jsonMode {
		parserType = parser.TypeJSON
	}

	p, err := parser.NewParser(parserType)
	if err != nil {
		log.Fatal(err)
	}

	keys := parser.DefaultKeys
	if key != "" {
		keys = []string{key}
	}

	services := []string{}
	if service != "" {
		services = strings.Split(service, ",")
	}

	cfg := scanner.ScanConfig{
		Dir:         dir,
		SearchValue: SearchValue,
		IgnoreCase:  ignoreCase,
		Keys:        keys,
		Since:       since,
		Limit:       limit,
		Recursive:   recursive,
		Services:    services,
	}
	scn := scanner.NewFileScanner(cfg, p)

	files, err := scn.ListLogFiles()
	if err != nil {
		log.Fatal(err)
	}

	entries := scn.Scan(files)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	colorizer := formatter.NewColorizer()
	for _, e := range entries {
		fmt.Println(formatter.Format(e, colorizer))
	}

	if follow {
		scn.Follow(files)
	}
}
