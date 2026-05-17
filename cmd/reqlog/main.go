package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sort"

	"github.com/sagarmaheshwary/reqlog/internal/cli"
	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
)

var version = "dev"

func init() {
	flag.Usage = cli.Usage
}

func main() {
	cfg, err := cli.ParseConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	if cfg.Version {
		fmt.Printf("reqlog version %s\n", cliVersion())
		return
	}

	if flag.NArg() < 1 {
		if flag.NArg() < 1 {
			fmt.Println(cli.Info())
			os.Exit(1)
		}
	}

	cfg.SearchValue = flag.Arg(0)

	run(cfg)
}

func run(cfg *config.Config) {
	lp := scanner.NewLineProcessor(&scanner.Config{
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
	}, scanner.NewTimeParser())
	scn, err := scanner.New(cfg.Source, lp)
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

	entries, err := scn.Scan(sources)
	if err != nil {
		log.Fatal(err)
	}

	f := formatter.NewFormatter(&formatter.Opts{
		Entries:    entries,
		SearchKeys: cfg.Keys,
		Output:     cfg.Output,
		Context:    cfg.Context,
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	for _, e := range entries {
		fmt.Println(f.Format(e))
	}

	if cfg.Follow {
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
