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

### Text Logs

- ISO-8601 / RFC3339 timestamps at start
- Supports timestamps **with or without fractional seconds**
- Outputs timestamps in **millisecond precision (fixed 3 digits)**
- `key=value` fields

```text
2026-03-20T14:00:00Z request_id=abc123 start request
2026-03-20T14:00:00.123Z request_id=abc123 processing
```

### JSON Logs

- One JSON per line
- Timestamp fields: `time`, `timestamp`, `ts`
- Supports timestamps **with or without fractional seconds**
- Outputs timestamps in **millisecond precision (fixed 3 digits)**

```json
{ "time": "2026-03-20T14:10:00Z", "request_id": "abc", "message": "start" }
{ "time": "2026-03-20T14:10:00.456Z", "request_id": "abc", "message": "processing" }
```

> Timestamps are parsed using RFC3339 (nano precision) and normalized to millisecond precision in output.

## Roadmap

### Core Features

- [x] Flexible timestamp parsing (RFC3339 / RFC3339Nano)
- [x] Text log parsing (key=value)
- [x] JSON log parsing
- [x] Wildcard support in `--service` (e.g. order-service\*)

- [ ] Unix timestamp support (logs + `--since`)
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
