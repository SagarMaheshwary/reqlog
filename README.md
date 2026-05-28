# reqlog

<p align="center">
  <b>Search and trace requests across files, Docker logs, and remote hosts.</b><br/>
  Debug distributed systems from your terminal using simple key/value search without relying on centralized tracing.
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

## Installation

**Go Install**

```bash
go install github.com/sagarmaheshwary/reqlog/cmd/reqlog@latest
```

**macOS / Linux**

```bash
curl -sSL https://raw.githubusercontent.com/sagarmaheshwary/reqlog/master/install.sh | bash
```

- Auto-detects OS/arch
- Installs latest version
- Installs to `/usr/local/bin`

Verify:

```bash
reqlog -v
```

**Windows**

Download from:

[https://github.com/sagarmaheshwary/reqlog/releases](https://github.com/sagarmaheshwary/reqlog/releases)

Then:

- unzip
- add to `PATH`

Verify:

```bash
reqlog -v
```

## Quick Start

Search logs using common request tracing keys like:

`request_id`, `req_id`, `trace_id`, and `correlation_id`.

Search log files:

```bash
reqlog abc123
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
2026-03-20T14:10:01.000Z [api-gateway]       | calling order service level=info request_id=abc123
2026-03-20T14:10:02.000Z [order-service]     | fetching order level=info request_id=abc123
2026-03-20T14:10:03.000Z [inventory-service] | checking stock level=info request_id=abc123
```

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

**Docker Logs**

```bash
reqlog -S docker -s api-gateway abc123
```

**Remote Logs over SSH**

Search logs across remote hosts using SSH by defining hosts in `config.yaml`.

Place `config.yaml` in one of the following locations:

- macOS/Linux: `~/.config/reqlog/config.yaml`
- Windows: `%APPDATA%\reqlog\config.yaml`

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

**Live Log Streaming**

```bash
reqlog -f abc123
```

> Full usage guide: [docs/usage.md](./docs/usage.md)

## Why not just use `grep`?

| Problem                         |      grep |      reqlog |
| ------------------------------- | --------: | ----------: |
| Multi-file search               | ⚠️ manual | ✅ built-in |
| Multi-host / SSH log search     |        ❌ |          ✅ |
| Request tracing across services |        ❌ |          ✅ |
| JSON log search                 |        ❌ |          ✅ |
| Chronological request flow      |        ❌ |          ✅ |
| Service-aware context           |        ❌ |          ✅ |

> `reqlog = grep for distributed systems`

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

## Roadmap

**Core Features**

- [x] Flexible timestamp parsing (RFC3339 / RFC3339Nano)
- [x] Text log parsing (key=value)
- [x] JSON log parsing
- [x] Wildcard support in `--service` (e.g. order-service\*)
- [x] Unix timestamp support (logs + `--since`)
- [x] Optimize `--limit` (early exit / streaming)
- [x] `--latest` flag (Return latest N entries globally)
- [x] `--context` flag (show surrounding lines)
- [x] `--output=json` for piping and integrations
- [ ] `--fields` flag for JSON logs

**Performance & Scalability**

- [ ] Parallel scanning
- [ ] General performance improvements

**Integrations**

- [x] File logs
- [x] Docker logs
- [x] SSH-based multi-host logs

> Companion web UI for reqlog: https://github.com/sagarmaheshwary/reqlog-ui

## Support & Contributions

If you find this project useful, consider giving it a ⭐ — it helps others discover it.

Feedback, contributions, and discussions are very welcome.
Feel free to open an issue or submit a PR.

## License

MIT
