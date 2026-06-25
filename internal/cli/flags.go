package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sagarmaheshwary/reqlog/internal/config"
)

type flagOptions struct {
	key     string
	service string
	source  string
	output  string
	format  string
	config  string
	fields  string
}

func ParseConfig() (*config.Config, error) {
	cfg := &config.Config{}

	opts := registerFlags(cfg)

	err := flag.CommandLine.Parse(normalizeArgs(os.Args[1:]))
	if err != nil {
		return nil, err
	}

	if err := applyDerivedConfig(cfg, opts); err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func registerFlags(cfg *config.Config) *flagOptions {
	opts := &flagOptions{}

	stringFlag(
		&cfg.Dir,
		"dir",
		"d",
		"./logs",
		"directory containing log files",
	)

	boolFlag(
		&cfg.IgnoreCase,
		"ignore-case",
		"i",
		false,
		"perform case-insensitive search",
	)

	intFlag(
		&cfg.Limit,
		"limit",
		"n",
		0,
		"limit number of results",
	)

	boolFlag(
		&cfg.Latest,
		"latest",
		"l",
		false,
		"return globally latest N matches across all sources",
	)

	boolFlag(
		&cfg.Follow,
		"follow",
		"f",
		false,
		"follow logs in real time",
	)

	stringFlag(
		&opts.key,
		"key",
		"k",
		"",
		"field key to match (e.g. request_id, trace_id, event_key)",
	)

	stringFlag(
		&cfg.Since,
		"since",
		"t",
		"",
		"filter logs newer than: duration (5m, 1h), RFC3339 timestamp, or YYYY-MM-DD (UTC)",
	)

	boolFlag(
		&cfg.Recursive,
		"recursive",
		"r",
		true,
		"scan directories recursively",
	)

	stringFlag(
		&opts.service,
		"service",
		"s",
		"",
		"filter by service name (comma-separated, e.g. order-service,inventory-service)",
	)

	stringFlag(
		&opts.source,
		"source",
		"S",
		"file",
		`log source backend ("file" or "docker")`,
	)

	boolFlag(
		&cfg.Version,
		"version",
		"v",
		false,
		"print version and exit",
	)

	intFlag(
		&cfg.Context,
		"context",
		"c",
		0,
		"show N lines of context before and after each match",
	)

	stringFlag(
		&opts.output,
		"output",
		"o",
		"pretty",
		`output format ("pretty" or "json")`,
	)

	stringFlag(
		&opts.format,
		"format",
		"F",
		"auto",
		`log parsing format ("auto", "json", or "text")`,
	)

	stringFlag(
		&opts.config,
		"config",
		"",
		"",
		"path to SSH config file",
	)

	stringFlag(
		&cfg.Host,
		"host",
		"H",
		"",
		"SSH host alias from config file",
	)

	stringFlag(
		&opts.fields,
		"fields",
		"",
		"",
		"display only selected fields (comma-separated, e.g. request_id,path,status)",
	)

	boolFlag(
		&cfg.Verbose,
		"verbose",
		"V",
		false,
		"show warnings and errors encountered during scanning",
	)

	return opts
}

func applyDerivedConfig(cfg *config.Config, opts *flagOptions) error {
	cfg.Keys = config.DefaultKeys

	if opts.key != "" {
		cfg.Keys = []string{opts.key}
	}

	if opts.service != "" {
		cfg.Services = strings.Split(opts.service, ",")
	}

	cfg.Source = config.Source(opts.source)
	cfg.Output = config.OutputFormat(opts.output)
	cfg.Format = config.LogFormat(opts.format)

	if cfg.Host != "" {
		var err error
		cfg.Config, err = config.NewSSH(opts.config)
		if err != nil {
			return err
		}
	}

	if opts.fields != "" {
		cfg.Fields = strings.Split(opts.fields, ",")
	}

	return nil
}

func validateConfig(cfg *config.Config) error {
	switch cfg.Output {
	case config.OutputPretty, config.OutputJSON:
	default:
		return fmt.Errorf("invalid output format: %s", cfg.Output)
	}

	switch cfg.Format {
	case config.FormatAuto, config.FormatJSON, config.FormatText:
	default:
		return fmt.Errorf("invalid log format: %s", cfg.Format)
	}

	if cfg.Latest && cfg.Limit == 0 {
		return fmt.Errorf("--latest requires --limit")
	}

	return nil
}

func normalizeArgs(args []string) []string {
	out := make([]string, 0, len(args))

	for _, arg := range args {
		if isTailStyleLimit(arg) {
			out = append(out, "-n", arg[1:])
			continue
		}

		out = append(out, arg)
	}

	return out
}

func isTailStyleLimit(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' {
		return false
	}

	for _, ch := range arg[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}

func stringFlag(target *string, name, short, def, usage string) {
	flag.StringVar(target, name, def, usage)

	if short != "" {
		flag.StringVar(target, short, def, "")
	}
}

func boolFlag(target *bool, name, short string, def bool, usage string) {
	flag.BoolVar(target, name, def, usage)

	if short != "" {
		flag.BoolVar(target, short, def, "")
	}
}

func intFlag(target *int, name, short string, def int, usage string) {
	flag.IntVar(target, name, def, usage)

	if short != "" {
		flag.IntVar(target, short, def, "")
	}
}
