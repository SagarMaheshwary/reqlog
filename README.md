# reqlog

<p align="center">
  <b>Search and trace requests across microservices, files, and Docker logs — fast.</b><br/>
  Debug distributed systems using simple key/value search (e.g. request_id, trace_id) without relying on centralized tracing.
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

## Why `reqlog`?

Debugging logs in microservices usually means:

- jumping between multiple files
- dealing with inconsistent formats
- writing fragile `grep | awk | sort` pipelines

**`reqlog` fixes this in one command.**

- Search logs across **multiple services**
- Match **structured fields** like `request_id`, `trace_id`
- Get a **chronological flow** of a request
- Visually scan logs with **service-based colors**

> Companion web UI for reqlog: https://github.com/sagarmaheshwary/reqlog-ui

## Quick Start

```bash
reqlog --dir ./logs --key request_id abc123
```

Example output:

```shell
2026-03-20T14:10:00.000Z [api-gateway]    | start request
2026-03-20T14:10:01.000Z [order-service]  | fetching order
2026-03-20T14:10:02.000Z [inventory]      | checking stock
```

Follow a request across services in seconds.

## Features

- Key-based search (`--key request_id`)
- Fast scanning (millions of lines)
- Supports **plain text** + **JSON logs**
- Docker logs support
- Filter by service (`--service`, supports wildcards)
- Time filtering (`--since`)
- Colored output by service
- Live tailing (`--follow`)
- Case-insensitive search
- Works across multiple files & directories

## Installation

### Go Install

```bash
go install github.com/sagarmaheshwary/reqlog/cmd/reqlog@latest
```

### macOS / Linux

```bash
curl -sSL https://raw.githubusercontent.com/sagarmaheshwary/reqlog/master/install.sh | bash
```

- Auto-detects OS/arch
- Installs latest version
- Installs to `/usr/local/bin`

Verify:

```bash
reqlog --version
```

### Windows

Download from:

[https://github.com/sagarmaheshwary/reqlog/releases](https://github.com/sagarmaheshwary/reqlog/releases)

Then:

- unzip
- add to `PATH`

Verify:

```bash
reqlog --version
```

## Usage

```bash
reqlog [flags] <search_value>
```

### Basic Search

```bash
reqlog --dir ./logs abc123
```

### Key-Based Search (Recommended)

```bash
reqlog --key request_id abc123
reqlog --key event_key order.created
```

### JSON Logs

```bash
reqlog --dir ./logs --json --key trace_id trace-123
```

### Docker Logs

```bash
reqlog --source docker --service api-gateway abc123
```

Wildcard support:

```bash
reqlog --service order-service* abc123
```

### Time Filtering

```bash
reqlog --since 10m --key request_id abc123
reqlog --since 2026-04-29 --key request_id abc123
reqlog --since 2026-04-29T14:00:00Z --key request_id abc123
reqlog --since 2026-04-29T14:00:00.123Z --key request_id abc123
reqlog --since 1710943200 --key request_id abc123
```

### Follow Logs (Live)

```bash
reqlog --follow --key request_id abc123
```

> Full usage guide: [docs/usage.md](./docs/usage.md)

## Why not just use `grep`?

| Problem            | grep      | reqlog      |
| ------------------ | --------- | ----------- |
| Multi-file search  | ⚠️ manual | ✅ built-in |
| Request tracing    | ❌        | ✅          |
| JSON logs          | ❌        | ✅          |
| Chronological flow | ❌        | ✅          |
| Service context    | ❌        | ✅          |

`reqlog = grep for distributed systems`

## Performance

- ~9.6M lines scanned in **~2 seconds**
- ~9 MB memory usage
- Works efficiently on real-world datasets

> Optimized for sequential reads + minimal memory usage

## Supported Log Formats

### Supported Timestamp Formats

- **RFC3339 / ISO-8601**
  - with or without fractional seconds
- **Unix timestamps**
  - seconds (10 digits)
  - milliseconds (13 digits)
  - microseconds (16 digits)
  - nanoseconds (19 digits)

Timestamps are normalized to **millisecond precision** in output (fixed 3 digits).

### Text Logs

- Timestamp must appear as the first field
- Supports `key=value` fields

```text
2026-03-20T14:00:00Z request_id=abc123 start request
2026-03-20T14:00:00.123Z request_id=abc123 processing
1710943200 request_id=abc123 unix seconds
1710943200123 request_id=abc123 unix milliseconds
```

### JSON Logs

- One JSON object per line
- Supported timestamp fields: `time`, `timestamp`, `ts`

```json
{ "time": "2026-03-20T14:10:00Z", "request_id": "abc", "message": "start" }
{ "time": "2026-03-20T14:10:00.456Z", "request_id": "abc", "message": "processing" }
{ "ts": 1710943200, "request_id": "abc", "message": "unix seconds" }
{ "ts": 1710943200123, "request_id": "abc", "message": "unix milliseconds" }
```

## Roadmap

### Core Features

- [x] Flexible timestamp parsing (RFC3339 / RFC3339Nano)
- [x] Text log parsing (key=value)
- [x] JSON log parsing
- [x] Wildcard support in `--service` (e.g. order-service\*)

- [x] Unix timestamp support (logs + `--since`)
- [ ] Optimize `--limit` (early exit / streaming)
- [ ] `--context` flag (show surrounding lines)
- [ ] `--fields` flag for JSON logs
- [ ] `--output=json` for piping and integrations

### Performance & Scalability

- [ ] Parallel scanning
- [ ] General performance improvements

### Integrations

- [x] File logs
- [x] Docker logs
- [ ] Kubernetes logs

## Contributing

Contributions and feedback are welcome!

## License

MIT
