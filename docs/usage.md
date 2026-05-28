# Usage Guide

## Table of Contents

- [Basic Syntax](#basic-syntax)
- [Flags](#flags)
- [Common Workflows](#common-workflows)
- [Remote Logs Over SSH](#remote-logs-over-ssh)
- [JSON Output](#json-output)
- [Supported Log Formats](#supported-log-formats)
- [Supported Timestamp Formats](#supported-timestamp-formats)
- [Tips](#tips)

## Basic Syntax

```bash
reqlog [flags] <search_value>
```

reqlog searches using common tracing keys by default:

- `request_id`
- `req_id`
- `trace_id`
- `correlation_id`

Use `--key` (`-k`) to override the default search key.

## Flags

| Flag            | Shorthand | Description                                 |
| --------------- | --------- | ------------------------------------------- |
| `--dir`         | `-d`      | Directory containing log files              |
| `--key`         | `-k`      | Field key to match                          |
| `--service`     | `-s`      | Filter by service/container name            |
| `--source`      | `-S`      | Log source backend (`file` or `docker`)     |
| `--format`      | `-F`      | Log parsing format (`auto`, `json`, `text`) |
| `--output`      | `-o`      | Output format (`pretty` or `json`)          |
| `--limit`       | `-n`      | Limit number of matches                     |
| `--latest`      | `-l`      | Return globally latest matches              |
| `--follow`      | `-f`      | Follow logs in real time                    |
| `--context`     | `-c`      | Show surrounding log lines                  |
| `--since`       | `-t`      | Filter logs newer than timestamp/duration   |
| `--ignore-case` | `-i`      | Case-insensitive search                     |
| `--recursive`   | `-r`      | Scan directories recursively                |
| `--version`     | `-v`      | Print version information                   |

## Common Workflows

### Basic Search

Search logs in `./logs`:

```bash
reqlog abc123
```

Search a specific directory:

```bash
reqlog -d ./logs abc123
```

### Key-Based Search

```bash
reqlog -k request_id abc123
reqlog -k event_key order.created
```

### Log Format Detection

reqlog automatically detects JSON logs by default.

```bash
reqlog -k trace_id trace-123
```

Force parsing mode explicitly when needed:

```bash
reqlog --format json abc123
reqlog --format text abc123
```

Available formats:

- `auto` (default)
- `json`
- `text`

> `--json` has been replaced with `--format=json`.

### Docker Logs

```bash
reqlog -S docker abc123
```

Filter specific containers:

```bash
reqlog -S docker -s api-gateway abc123
```

### Service Filtering

Filter logs by service name.

- When using Docker logs, this filters **container names**
- Otherwise, it filters **log file names**

```bash
reqlog -s api-gateway,order-service abc123
```

#### Wildcard Support

```bash
reqlog -s order-service* abc123
```

### Context Around Matches

Show surrounding log lines before and after each match:

```bash
reqlog -c 2 abc123
```

### Limiting Results

Return first N matches per source:

```bash
reqlog -n 10 abc123
```

Tail-style shorthand is also supported:

```bash
reqlog -100 abc123
```

Return globally latest N matches across all sources:

```bash
reqlog -l -n 10 abc123
```

> `--latest` (`-l`) scans all matching logs to determine the newest entries globally, so it may be slower on very large log files or containers.

### Time Filtering

`--since` (`-t`) accepts either a Go duration or an absolute timestamp.

#### Duration Examples

```bash
reqlog -t 10m abc123
reqlog -t 2h abc123
reqlog -t 1h30m abc123
```

Supported duration formats:

- `30s`
- `5m`
- `1h`
- `1h30m`

#### Absolute Timestamp Examples

```bash
reqlog -t 2026-04-29 abc123
reqlog -t 2026-04-29T14:00:00Z abc123
reqlog -t 2026-04-29T14:00:00.123Z abc123
reqlog -t 1710943200 abc123
```

Supported timestamp formats:

- `YYYY-MM-DD`
- RFC3339 / ISO-8601 timestamps
- RFC3339 timestamps with fractional seconds
- Unix timestamps
  - seconds (10 digits)
  - milliseconds (13 digits)
  - microseconds (16 digits)
  - nanoseconds (19 digits)

### Case-Insensitive Search

```bash
reqlog -i -k event_key ORDER.CREATED
```

### Live Log Streaming

```bash
reqlog -f abc123
```

### Non-Recursive Scan

Disable recursive directory scanning:

```bash
reqlog --recursive=false abc123
```

## Remote Logs Over SSH

Search logs across remote hosts using SSH by defining hosts in `config.yaml`.

Place `config.yaml` in one of the following locations:

- macOS/Linux: `~/.config/reqlog/config.yaml`
- Windows: `%APPDATA%\reqlog\config.yaml`

**Example `config.yaml`**

```yaml
version: 1

defaults:
  key: ~/.ssh/id_rsa
  timeout: 30s

hosts:
  srv1:
    host: 10.0.0.10
    user: ubuntu

  srv2:
    host: 10.0.0.11
    user: ec2-user
    port: 2222
    key: ~/.ssh/prod.pem
    timeout: 60s
```

Search logs on a single remote host:

```bash
reqlog -H srv1 abc123
```

Search across multiple hosts:

```bash
reqlog -H srv1,srv2 abc123
```

Search Docker logs on remote hosts:

```bash
reqlog -H srv1,srv2 -S docker abc123
```

> Host names passed to `-H` must match entries under `hosts` in `config.yaml`.
>
> SSH logs include host context in outputs:
> - Pretty output: `[host:service]`
> - JSON output includes a `host` field

Specify a custom config file:

```
reqlog --config ./config.yaml -H srv1 abc123
```

## JSON Output

Structured output for piping and integrations:

```bash
reqlog -o json abc123
reqlog -o json abc123 | jq .
```

Example output:

```json
{
  "timestamp": "2026-03-20T14:00:05Z",
  "service": "order-service",
  "message": "request started",
  "level": "info",
  "request_id": "abc123"
}
```

## Supported Log Formats

reqlog supports both text and JSON logs.

In `auto` mode (default), reqlog automatically detects the log format per line.

Use `--format` to force a specific parser when needed.

### Text Logs

Requirements:

- timestamp must appear first
- supports mixed message and `key=value` fields

Examples:

```text
2026-03-20T14:00:00Z request_id=abc123 start request
2026-03-20T14:00:00Z start request request_id=abc123
2026-03-20T14:00:00Z level=info request_id=abc123 request started
1710943200 request_id=abc123 unix seconds
1710943200123 request_id=abc123 unix milliseconds
```

### JSON Logs

Requirements:

- one JSON object per line
- supported timestamp fields:
  - `time`
  - `timestamp`
  - `ts`

Examples:

```json
{ "time": "2026-03-20T14:10:00Z", "request_id": "abc", "message": "start" }
{ "time": "2026-03-20T14:10:00.456Z", "request_id": "abc", "message": "processing" }
{ "ts": 1710943200, "request_id": "abc", "message": "unix seconds" }
{ "ts": 1710943200123, "request_id": "abc", "message": "unix milliseconds" }
```

## Supported Timestamp Formats

### RFC3339 / ISO-8601

```text
2026-03-20T14:00:00Z
2026-03-20T14:00:00.123Z
```

### Unix Timestamps

```text
1710943200
1710943200123
1710943200123456
1710943200123456789
```

Supported precisions:

- seconds (10 digits)
- milliseconds (13 digits)
- microseconds (16 digits)
- nanoseconds (19 digits)

Timestamps are normalized to millisecond precision in output.

## Tips

- Use `-k` for faster and more precise searches
- Use `-n` or tail-style `-100` for high-volume logs
- Use `-o json` for integrations and piping
- Prefer structured logs for better filtering and output
