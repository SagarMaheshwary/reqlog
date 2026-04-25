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

## Time Filtering

```bash
reqlog --since 10m --key request_id abc123
```

Formats:

- 30s
- 5m
- 1h
- 1h30m

## Limit Results

```bash
reqlog --limit 20 --key event_key order.created
```

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

## Example Output

```shell
2026-03-20T14:10:01.000Z [api-gateway]       | calling order service level=info request_id=abc123
2026-03-20T14:10:02.000Z [order-service]     | fetching order level=info request_id=abc123
2026-03-20T14:10:03.000Z [inventory-service] | checking stock level=info request_id=abc123
```

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

## Limitations

- No support for numeric timestamps (epoch) yet
- No multi-line logs

## Tips

- Use `--key` for better performance
- Use `--limit` for high-frequency queries
- Prefer JSON logs for structured search
