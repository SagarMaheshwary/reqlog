# `reqlog`

**Search and trace logs across services or a single file — fast.**

![reqlog demo](./assets/demo.gif)

`reqlog` is a lightweight CLI tool for backend engineers to quickly search logs by key/value (e.g. `request_id`, `trace_id`) across one or many log files.

It works equally well for:

- **Microservices** → trace a request across multiple services
- **Monoliths** → search and filter large single log files

No complex `grep` pipelines. No jumping between files.

## Why `reqlog`?

Debugging logs often means:

- Logs scattered across multiple files or services
- Different formats (plain text, JSON, pretty logs)
- Slow and error-prone manual searching

`reqlog` simplifies this by:

- Searching across multiple files in one command
- Matching structured key/value fields (not just raw text)
- Showing a clean, chronological flow of events
- Coloring output by service for easy scanning

## Features

- Key-based search (e.g. `--key request_id abc123`)
- Fast scanning (handles millions of lines efficiently)
- Supports **plain text** and **JSON** logs (single-line)
- Filter by specific services (`--service`, based on log file names)
- Recursive or non-recursive directory scanning
- Colored output by service
- Time filtering (`--since`)
- Case-insensitive search (`--ignore-case`)
- Follow mode for live logs (`--follow`)
- Custom key support (`request_id`, `trace_id`, `event_key`, etc.)

## Installation

### Go install

```bash
go install github.com/sagarmaheshwary/reqlog/cmd/reqlog@latest
```

### Linux

```bash
curl -sSL https://raw.githubusercontent.com/sagarmaheshwary/reqlog/master/install.sh | bash
```

### Windows

Download the latest release from GitHub:

https://github.com/sagarmaheshwary/reqlog/releases

Unzip and add to PATH.

## Usage

```bash
reqlog [flags] <search_value>
```

### Basic Search

Search all logs in a directory:

```bash
reqlog --dir ./logs abc123
```

By default, `reqlog` searches for common request tracing keys:

- `request_id`
- `req_id`
- `trace_id`
- `correlation_id`

### Key-Based Search (Recommended)

Match structured fields like `request_id`, `trace_id`, or `event_key`:

```bash
reqlog --key request_id abc123
reqlog --key event_key order.created
```

### JSON Logs

Parse logs as JSON (one object per line):

```bash
reqlog --dir ./logs --json --key trace_id trace-123
```

### Filter by Service

Search only specific services (based on log file names, e.g. `order-service.log`):

```bash
reqlog \
  --service api-gateway,order-service \
  --key request_id \
  abc123
```

### Time Filtering

Show logs from the last duration:

```bash
reqlog --since 10m --key request_id abc123
```

Supported duration formats:

- `30s` → last 30 seconds
- `5m` → last 5 minutes
- `1h` → last 1 hour
- `2h` → last 2 hours

You can also combine units:

- `1h30m` → last 1 hour 30 minutes
- `2m10s` → last 2 minutes 10 seconds

> Uses Go-style duration format.

### Limit Results

Return only the newest N matches:

```bash
reqlog --limit 20 --key event_key order.created
```

### Case-Insensitive Search

```bash
reqlog --ignore-case --key event_key ORDER.CREATED
```

### Follow Logs (Live)

Stream logs in real time (like `tail -f`):

```bash
reqlog --follow --key request_id abc123
```

### Non-Recursive Scan

Disable recursive directory scanning:

```bash
reqlog --recursive=false --dir ./logs abc123
```

### Example Output

```text
15:00:01 [api-gateway] | request_id=abc123 calling order service
15:00:02 [order-service] | request_id=abc123 fetching order
15:00:03 [inventory-service] | request_id=abc123 checking stock
```

## Performance

### Benchmark Environment

- **CPU:** AMD Ryzen Pro 5650U
- **Disk:** NVMe SSD (Gen3)
- **Execution:** Local machine (comparable to typical VM / cloud disk performance)

### Dataset

Benchmarks are run on **realistic multi-service logs**:

**Text Logs**

- `order-service.log` (~5.9M lines, ~1GB)
- `inventory-service.log` (~3.7M lines, ~758MB)

**JSON Logs**

- `order-service.log` (~5.9M lines, ~1.3GB)
- `inventory-service.log` (~3.7M lines, ~919MB)

> Total: ~9.6M lines across 2 services

### Results

| Command                                                                     | Scenario                         | Time    | Memory |
| --------------------------------------------------------------------------- | -------------------------------- | ------- | ------ |
| `reqlog --dir ./logs/text <request-id>`                                     | Plain text (match found)         | ~1.94s  | ~9 MB  |
| `reqlog --dir ./logs/text does-not-exist`                                   | Plain text (no match, full scan) | ~1.90s  | ~9 MB  |
| `reqlog --dir ./logs/text --ignore-case <request-id>`                       | Case-insensitive search          | ~2.79s  | ~9 MB  |
| `reqlog --dir ./logs/json --json <request-id>`                              | JSON logs (match found)          | ~2.53s  | ~9 MB  |
| `reqlog --dir ./logs/json --key request_id --json <request-id>`             | JSON with explicit key           | ~2.22s  | ~9 MB  |
| `reqlog --dir ./logs/json --limit 100 --key event_key --json order.created` | High-frequency + limit           | ~11.53s | ~9 MB  |

### Notes on Performance

- **Real-world dataset:** Logs are generated from actual service patterns (`order-service`, `inventory-service`), making results representative of real debugging scenarios.

- **Full scan behavior:**
  `reqlog` scans all files to reconstruct a **complete cross-service timeline**.
  Even with `--limit`, it does not early-exit to ensure correctness.

- **Key-based search matters:**
  Using `--key` avoids heuristic detection and improves performance, especially for JSON logs.

- **High-frequency queries are expensive:**
  Searching for common fields (e.g. `event_key=order.created`) produces many matches → more heap operations → slower execution.

- **CPU-bound workload:**
  Performance is dominated by:
  - string matching
  - parsing (especially JSON)
  - in-memory sorting (min-heap)

- **Disk impact:**
  Sequential reads are used. Gen3 SSD performance is comparable to many:
  - cloud VM disks (EBS / network SSD)
  - staging / production servers in typical setups

  Faster disks (Gen4+) may yield modest improvements.

- **Single-threaded (current design):**
  v1 processes logs sequentially. Parallelism may improve performance in future versions.

### Reproducing Benchmarks

You can generate similar datasets using:

```bash
go run cmd/loggen/main.go --format=[json,text] --orders=2000000
```

> Note: `--orders` controls business events, not raw line count.
> Each order generates multiple log entries across services.

## Why not just use `grep`?

`grep` is great for simple text search, but it falls short when debugging requests across distributed systems.

`reqlog` is designed specifically for this use case:

- **Request-aware search**  
  Automatically detects and filters by request IDs

- **Multi-service correlation**  
  Search across multiple log files and reconstruct a single request flow

- **Structured log support**  
  Works with JSON logs (not just plain text)

- **Timeline view**  
  Outputs logs in chronological order across services

- **Service-level context**  
  Colorized output makes it easy to distinguish services at a glance

> Think of `reqlog` as `grep` + context + structure for distributed systems.

## Roadmap

- [ ] Docker log support
- [ ] Kubernetes log support
- [ ] Performance optimizations (parallel scanning)
- [ ] Support for additional structured log formats

## Contributing

Contributions, issues, and suggestions are welcome!

## License

MIT
