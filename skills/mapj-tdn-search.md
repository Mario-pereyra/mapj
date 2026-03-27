---
name: mapj-tdn-search
description: >
  Search TOTVS Developer Network (TDN) documentation which is a Confluence-based system.
  Use when: finding technical documentation, looking up API references, searching for Protheus development guides,
  finding ERP customization documentation, or any TOTVS-related technical search.
  Triggers: "search TDN", "find documentation", "look up TOTVS", "search documentation",
  "find API docs", "search Protheus docs", "TDN search".
metadata:
  version: 1.0.0
  language: en
  author: Mario Pereira
  license: MIT
  repository: https://github.com/Mario-pereyra/mapj
  tags:
    - tdn
    - totvs
    - confluence
    - documentation
    - search
    - api
    - protheus
  capabilities:
    - search
    - filter-by-space
    - filter-by-label
  related:
    - mapj-confluence-export
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj tdn search

Search documentation in TDN (TOTVS Developer Network), a Confluence-based documentation system at `tdninterno.totvs.com`.

## Usage

```
mapj tdn search <query> [--space SPACE] [--limit N] [--output json|table]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<query>` | Yes | Search query string (CQL-compatible) |

## Flags

| Flag | Short | Required | Default | Description |
|------|-------|----------|---------|-------------|
| `--space` | `-s` | No | - | Filter by space key (e.g., "PROT", "LDT", "FLUIG") |
| `--limit` | `-l` | No | 10 | Maximum number of results (1-100) |
| `--output` | `-o` | No | json | Output format: `json` or `table` |

## Examples

### Basic Search

```bash
mapj tdn search "REST API authentication"
```

### Search with Space Filter

```bash
mapj tdn search "invoice" --space PROT --limit 5
```

### Human-Readable Output

```bash
mapj tdn search "WebService" --output table
```

### Combining Filters

```bash
mapj tdn search "MT0795" --space LDT --limit 20
```

## Prerequisites

1. **Authenticate first**:
   ```bash
   mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT_TOKEN
   ```

2. **Verify auth**:
   ```bash
   mapj auth status
   ```

## Output Schema

### Success Response

```json
{
  "ok": true,
  "command": "mapj tdn search \"REST API\"",
  "result": {
    "results": [
      {
        "id": "123456789",
        "type": "page",
        "title": "REST API Authentication Guide",
        "url": "https://tdninterno.totvs.com/wiki/spaces/PROT/pages/123456789/Title",
        "space": {
          "key": "PROT",
          "name": "Protheus"
        },
        "excerpt": "...matched text excerpt..."
      }
    ],
    "count": 3,
    "total": 42
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### Error Response

```json
{
  "ok": false,
  "command": "mapj tdn search \"query\"",
  "error": {
    "code": "AUTH_ERROR",
    "message": "not authenticated. Run: mapj auth login tdn --url https://tdninterno.totvs.com --token TOKEN",
    "retryable": false
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

## CQL Query Language

TDN uses Confluence Query Language (CQL) internally. The CLI automatically constructs CQL from your search terms.

### Basic Operators

| Operator | Example | Description |
|----------|---------|-------------|
| `text ~ "query"` | `text ~ "REST API"` | Full-text search (default) |
| `space = "KEY"` | `space = "PROT"` | Filter by space |
| `label = "tag"` | `label = "api"` | Filter by label |
| `type = "page"` | `type = "page"` | Content type |
| `title ~ "text"` | `title ~ "authentication"` | Title search |

### Combined Example

```bash
mapj tdn search "authentication" --space PROT
# Translates to: text ~ "authentication" AND space = "PROT"
```

## Known Spaces

| Space Key | Name | Description |
|-----------|------|-------------|
| PROT | Protheus | Protheus ERP documentation |
| LDT | Linha Datasul | Datasul product line |
| FLUIG | Fluig | Fluig platform |
| EngMP | Engenharia - Protheus | Protheus engineering |

## Error Cases

| Code | Condition | Resolution |
|------|-----------|------------|
| AUTH_ERROR (3) | Not logged in | Run `mapj auth login tdn ...` |
| USAGE_ERROR (2) | Invalid arguments | Check `--limit` is 1-100 |
| SEARCH_ERROR (1) | Server error | Retry with backoff (code 4) |
| RATE_LIMITED (4) | Too many requests | Wait 1s and retry |

### Rate Limiting

If you receive `RETRYABLE` errors, implement exponential backoff:

```bash
#!/bin/bash
for i in 1 2 3; do
  result=$(mapj tdn search "$query" 2>&1)
  if echo "$result" | jq -e '.ok'; then
    echo "$result"
    exit 0
  fi
  sleep $i
done
echo "Failed after 3 retries"
exit 1
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Search successful |
| 1 | Search failed (server error) |
| 2 | Usage error (invalid arguments) |
| 3 | Auth error (not authenticated) |
| 4 | Retry recommended (rate limited) |

## See Also

- [SKILL.md](../SKILL.md) - Main manifest
- [mapj-confluence-export](mapj-confluence-export.md) - Export found pages
- [mapj-protheus-query](mapj-protheus-query.md) - Query Protheus database
