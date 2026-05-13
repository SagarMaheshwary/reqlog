# Usage Guide

## Basic Syntax

```bash
reqlog [flags] <search_value>
```

## Basic Search

Search logs in a directory:

```bash
reqlog --dir ./logs abc123
```

Default keys searched:

- request_id
- req_id
- trace_id
- correlation_id

## Key-Based Search (Recommended)

```bash
reqlog --key request_id abc123
reqlog --key event_key order.created
```

## JSON Logs

```bash
reqlog --dir ./logs --json --key trace_id trace-123
```

## Docker Logs

```bash
reqlog --source docker --service api-gateway abc123
```

## Service Filtering

Filter logs by service name.

- When using `--source docker`, this filters **container names**
- Otherwise, it filters **log file names**

```bash
reqlog \
  --service api-gateway,order-service \
  --key request_id \
  abc123
```

### Wildcard Support

```bash
reqlog --service order-service* abc123
```

## Context Around Matches

Show surrounding log lines before and after each match:

```bash
reqlog --context 2 --key request_id abc123
```

## Limiting Results

Return first N matches per source:

```bash
reqlog --limit 10 --key request_id abc123
```

Return globally latest N matches across all sources:

```bash
reqlog --latest --limit 10 --key request_id abc123
```

> `--latest` scans all matching logs to determine the newest entries globally, so it may be slower on very large log files or containers.

## Time Filtering

`--since` accepts either a Go duration or an absolute timestamp.

**Duration examples**

```bash
reqlog --since 10m --key request_id abc123
reqlog --since 2h --key request_id abc123
```

Formats:

- 30s
- 5m
- 1h
- 1h30m

**Absolute timestamp examples**

```bash
reqlog --since 2026-04-29 --key request_id abc123
reqlog --since 2026-04-29T14:00:00Z --key request_id abc123
reqlog --since 2026-04-29T14:00:00.123Z --key request_id abc123
reqlog --since 1710943200 --key request_id abc123
```

Supported absolute timestamp formats:

- `YYYY-MM-DD`
- RFC3339 / ISO-8601 timestamps
- RFC3339 / ISO-8601 timestamps with fractional seconds
- **Unix timestamps**
  - seconds (10 digits)
  - milliseconds (13 digits)
  - microseconds (16 digits)
  - nanoseconds (19 digits)

## Case-Insensitive Search

```bash
reqlog --ignore-case --key event_key ORDER.CREATED
```

## Follow Mode (Live Logs)

```bash
reqlog --follow --key request_id abc123
```

## Non-Recursive Scan

```bash
reqlog --recursive=false --dir ./logs abc123
```

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

## Limitations

- No multi-line logs

## Tips

- Use `--key` for better performance
- Use `--limit` for high-frequency queries
- Prefer JSON logs for structured search
