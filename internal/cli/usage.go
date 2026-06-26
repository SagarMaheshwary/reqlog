package cli

import (
	"flag"
	"fmt"
)

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `reqlog - Search, trace, and stream logs across files, Docker containers, and remote hosts.

Usage:
  reqlog [flags] <search_value>

Examples:
  reqlog abc123
  reqlog -k request_id abc123
  reqlog -S docker -s order-service abc123
  reqlog -H srv1 -d /path/to/logs abc123
  reqlog --config ~/.config/reqlog/config.yaml -H srv1 -d /path/to/logs abc123
  reqlog -f -o json abc123 | jq

Flags:
  -d, --dir string
        directory containing log files
        (default "./logs")

  -r, --recursive
        scan directories recursively

  -S, --source string
        log source backend ("file" or "docker")
        (default "file")

  -H, --host string
        SSH host alias from config file

      --config string
        path to SSH config file

  -k, --key string
        field key to match
        (e.g. request_id, trace_id, event_key)

  -s, --service string
        filter by service name
        (comma-separated)

  -t, --since string
        filter logs newer than:
        duration (5m, 1h), RFC3339 timestamp, or YYYY-MM-DD

  -i, --ignore-case
        perform case-insensitive search

  -n, --limit int
        limit number of results

        Tail-style shorthand is also supported:
        reqlog -100 abc123

  -l, --latest
        return globally latest N matches across all sources

  -c, --context int
        show N lines of context before and after each match

      --fields string
        display only selected fields
        (comma-separated)

  -o, --output string
        output format ("pretty" or "json")
        (default "pretty")

  -F, --format string
        log parsing format ("auto", "json", or "text")
        (default "auto")

  -f, --follow
        follow logs in real time

  -V, --verbose
        show warnings and errors encountered during scanning

  -v, --version
        print version and exit
`)
}

func Info() string {
	return `Usage:
  reqlog [flags] <search_value>

Run 'reqlog --help' for more information.
`
}
