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

```text
15:00:01 [api-gateway]       | calling order service level=info request_id=abc123
15:00:02 [order-service]     | fetching order level=info request_id=abc123
15:00:03 [inventory-service] | checking stock level=info request_id=abc123
```

## Supported Log Formats

### Text Logs

- ISO-8601 timestamp at start
- key=value format

```text
2026-03-20T14:00:00Z request_id=abc123 start request
```

### JSON Logs

- One JSON per line
- Timestamp fields:
  - time
  - timestamp
  - ts

```json
{ "time": "2026-03-20T14:10:00Z", "request_id": "abc", "message": "start" }
```

## Limitations

- No support for numeric timestamps (epoch)
- No multi-line logs
- Pretty logs not supported yet

## Tips

- Use `--key` for better performance
- Use `--limit` for high-frequency queries
- Prefer JSON logs for structured search
