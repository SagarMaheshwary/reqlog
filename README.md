# reqlog

<p align="center">
  <b>CLI for searching, tracing, and streaming logs across files, Docker containers, and remote hosts.</b><br/>
  Trace requests across distributed systems using request_id, trace_id, correlation_id, and key/value log search — without centralized tracing.
</p>

<p align="center">
  <a href="https://github.com/sagarmaheshwary/reqlog/releases">
    <img src="https://img.shields.io/github/v/release/sagarmaheshwary/reqlog" />
  </a>
  <a href="https://github.com/sagarmaheshwary/reqlog/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/sagarmaheshwary/reqlog" />
  </a>
  <img src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-blue" />
  <img src="https://img.shields.io/badge/go-1.20+-00ADD8?logo=go" />
</p>

![reqlog demo](./assets/demo.gif)

## Table of Contents

- [What is reqlog](#what-is-reqlog)
- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Requirements](#requirements)
- [Companion UI](#companion-ui)
- [Usage](#usage)
- [Supported Log Formats](#supported-log-formats)
- [What's Next](#whats-next)

## What is reqlog?

reqlog is a CLI tool for searching, tracing, and streaming logs across:

- local log files
- Docker containers
- remote servers over SSH

It helps engineers debug requests across microservices and distributed systems using:

- `request_id`
- `trace_id`
- `correlation_id`
- custom log keys

Use reqlog when logs are spread across services or hosts, centralized tracing is unavailable, or you need fast terminal-based debugging.

> grep for distributed systems logs

Unlike `grep`, reqlog understands timestamps, JSON logs, Docker logs, multi-host SSH search, chronological request flow, and live log streaming.

## Features

- Search logs across multiple files
- Trace requests using `request_id`, `trace_id`, and `correlation_id`
- Stream live logs in real time (`-f`)
- Search and stream Docker container logs
- Search and stream logs on remote servers over SSH
- JSON log parsing with automatic detection
- Chronological request tracing across services
- Context around matches (`--context`)
- Structured JSON output for scripting
- Time-based filtering (`--since`)

## Quick Start

Search logs using common request tracing keys like:

`request_id`, `req_id`, `trace_id`, and `correlation_id`.

Search log files:

```bash
reqlog abc123
```

Stream live logs in real time:

```bash
reqlog -f abc123
```

Search Docker containers:

```bash
reqlog -S docker abc123
```

Search remote hosts over SSH:

```bash
reqlog -H srv1,srv2 abc123
```

> Remote host search requires `config.yaml` configuration. See **Remote Logs over SSH** below.

Example output:

```text
2026-03-20T14:10:01.000Z [api-gateway]       | INFO calling order service request_id=abc123
2026-03-20T14:10:02.000Z [order-service]     | INFO fetching order request_id=abc123
2026-03-20T14:10:03.000Z [inventory-service] | INFO checking stock request_id=abc123
```

## Installation

reqlog is a single statically linked binary with no runtime dependencies.

Supported Architectures

- `amd64`
- `arm64`

**Go Install**

```text
go install github.com/sagarmaheshwary/reqlog/cmd/reqlog@latest
```

**macOS**

```text
brew install sagarmaheshwary/tap/reqlog
```

**Windows**

```text
scoop bucket add sagarmaheshwary https://github.com/sagarmaheshwary/scoop-bucket
scoop install reqlog
```

**Linux packages**

Download the appropriate package for your distribution from the latest [release](https://github.com/sagarmaheshwary/reqlog/releases/latest):

- `.deb` (Debian/Ubuntu)
- `.rpm` (Fedora/RHEL/Rocky Linux/openSUSE)
- `.tar.gz` (manual installation)

**Verify Installation**

```text
reqlog --version
```

## Requirements

reqlog relies on a few system tools depending on the log source.

**Local files**

No dependencies.

**Docker logs**

Requires `docker` CLI (local and remote hosts).

**SSH file logs (Linux servers)**

Requires standard Unix utilities:

```bash
find, cat, tail, wc
```

**Notes**

- SSH log access is designed for Linux-based environments (VMs, EC2, containers, Kubernetes nodes, bare-metal servers).
- Windows SSH environments are not guaranteed.

## Companion UI

Visual interface for exploring reqlog results in the browser:

https://github.com/sagarmaheshwary/reqlog-ui

## Usage

```bash
reqlog [flags] <search_value>
```

**Basic Search**

```bash
reqlog abc123
```

> reqlog searches `./logs` by default. Use `-d` to search a specific directory.<br>
> JSON logs are detected automatically.

**Key-Based Search**

```bash
reqlog -k request_id abc123
reqlog -k event_key order.created
```

**Live Log Streaming**

```bash
reqlog -f abc123
```

**Docker Logs**

```bash
reqlog -S docker -s api-gateway abc123
```

**Remote Logs over SSH**

Search logs across remote hosts using SSH by defining hosts in `config.yaml`.

By default, reqlog looks for `config.yaml` in:

- Linux: `~/.config/reqlog/config.yaml`
- macOS: `~/Library/Application Support/reqlog/config.yaml`
- Windows: `%APPDATA%\reqlog\config.yaml`

You can also specify a custom configuration file:

```bash
reqlog --config ./config.yaml -H srv1 abc123
```

Example `config.yaml`:

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
    key: /home/ubuntu/.ssh/prod.pem
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

**Service Filtering**

```bash
reqlog -s order-service abc123
```

> `-s` filters **container names** when using Docker logs; otherwise, it filters **log file names**.

**Context Around Matches**

Show surrounding log lines before and after each match:

```bash
reqlog -c 2 abc123
```

**Limiting Results**

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

**Time Filtering**

```bash
reqlog --since 10m abc123
reqlog --since 2026-04-29 abc123
reqlog --since 2026-04-29T14:00:00Z abc123
reqlog --since 2026-04-29T14:00:00.123Z abc123
reqlog --since 1710943200 abc123
```

**JSON Output**

Structured output for piping and integrations:

```bash
reqlog -o json abc123
reqlog -o json abc123 | jq .
```

Output structure:

```json
{
  "timestamp": "2026-03-20T14:00:05Z",
  "service": "order-service",
  "message": "request started",
  "level": "info",
  "request_id": "abc123"
}
```

**Field Selection**

Display only the fields relevant to your investigation:

```bash
reqlog --fields request_id,path,status abc123
```

> Full usage guide: [docs/usage.md](./docs/usage.md)

## Supported Log Formats

**Supported Timestamp Formats**

- RFC3339 / ISO-8601
  - with or without fractional seconds
- Unix timestamps
  - seconds (10 digits)
  - milliseconds (13 digits)
  - microseconds (16 digits)
  - nanoseconds (19 digits)

Timestamps are normalized to millisecond precision in output (fixed 3 digits).

**Text Logs**

- Timestamp must appear as the first field
- Supports `key=value` fields

```text
2026-03-20T14:00:00Z request_id=abc123 start request
2026-03-20T14:00:00.123Z request_id=abc123 processing
1710943200 request_id=abc123 unix seconds
1710943200123 request_id=abc123 unix milliseconds
```

**JSON Logs**

- One JSON object per line
- Supported timestamp fields: `time`, `timestamp`, `ts`

```json
{ "time": "2026-03-20T14:10:00Z", "request_id": "abc", "message": "start" }
{ "time": "2026-03-20T14:10:00.456Z", "request_id": "abc", "message": "processing" }
{ "ts": 1710943200, "request_id": "abc", "message": "unix seconds" }
{ "ts": 1710943200123, "request_id": "abc", "message": "unix milliseconds" }
```

## What’s Next

`reqlog` now supports the core workflows it was built for:

- Search local log files
- Search Docker logs
- Search remote logs over SSH
- Trace requests across services using common keys like `request_id`, `trace_id`, and `correlation_id`
- Output structured JSON for piping and automation

Future updates will mainly focus on:

- Bug fixes and stability improvements
- Output and usability improvements
- Performance tuning over time
- Additional features based on real-world usage and feedback

## Support & Contributions

If you find this project useful, consider giving it a ⭐ — it helps others discover it.

Feedback, contributions, and discussions are very welcome.
Feel free to open an issue or submit a PR.

## License

MIT
