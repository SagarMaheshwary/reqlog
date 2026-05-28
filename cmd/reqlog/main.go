package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/sagarmaheshwary/reqlog/internal/app"
	"github.com/sagarmaheshwary/reqlog/internal/cli"
)

var version = "dev"

func init() {
	flag.Usage = cli.Usage
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatal(err.Error())
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
