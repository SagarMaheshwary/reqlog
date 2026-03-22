# `reqlog`

**Trace a request ID across multiple log files in seconds.**

`reqlog` is a lightweight CLI tool for backend engineers to quickly follow a single request across distributed services without jumping between files or writing complex grep commands.

## Why `reqlog`?

Debugging in microservices often means:

- Logs scattered across multiple files/services
- Different formats (plain text, JSON)
- Painful manual searching

`reqlog` solves this by:

- Searching across multiple log files
- Extracting and matching request IDs
- Showing a clean, chronological flow of events
- Coloring output by service for easy scanning

## Features

- Search request ID across multiple log files
- Fast (handles millions of lines in seconds)
- Supports **plain text** and **JSON** logs
- Service-based colored output
- Follow mode for streaming logs (`-follow`)
- Time filtering (`-since`)
- Case-insensitive search (`-ignore-case`)
- Custom request ID key support

## Installation

```bash
go install github.com/sagarmaheshwary/reqlog/cmd/reqlog@latest
```

## Usage

### Basic

```bash
reqlog -dir ./logs abc123
```

### JSON logs

```bash
reqlog -dir ./logs -json abc123
```

### Follow logs (like `tail -f`)

```bash
reqlog -dir ./logs -follow abc123
```

### Last 10 minutes

```bash
reqlog -dir ./logs -since 10m abc123
```

### Limit results

```bash
reqlog -dir ./logs -limit 100 abc123
```

### Ignore case

```bash
reqlog -dir ./logs -ignore-case abc123
```

### Custom request ID key

```bash
reqlog -dir ./logs -key trace_id abc123
```

## Example Output

```text
[14:00:01] api | request_id=abc123 calling order service
[14:00:02] order | request_id=abc123 fetching order
[14:00:03] inventory | request_id=abc123 checking stock
```

- Each service is color-coded
- Logs are displayed in a clean, readable timeline

## Performance

`reqlog` is optimized for speed and low memory usage.

Example benchmark (10 files × 1M lines):

- ~1.5–2 seconds search time
- ~8–9 MB memory usage

## Why not just use `grep`?

Unlike `grep`, `reqlog`:

- Understands request IDs
- Works across multiple files/services
- Supports structured (JSON) logs
- Formats output as a readable timeline
- Adds service-level context with colors

## Roadmap

- [ ] Docker log support
- [ ] Kubernetes log support

## Contributing

Contributions, issues, and suggestions are welcome!

## License

MIT
