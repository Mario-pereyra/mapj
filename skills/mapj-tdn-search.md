---
name: mapj-tdn-search
description: "Search TOTVS Developer Network (TDN) documentation"
metadata:
  version: 1.0.0
---

# mapj tdn search

Search documentation in TDN (TOTVS Developer Network), which is a Confluence-based documentation system.

## Usage

```
mapj tdn search <query> [--space SPACE] [--limit N] [--output json|table]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<query>` | Yes | Search query string |

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--space` | No | - | Filter by space key (e.g., "PROT", "FLUIG") |
| `--limit` | No | 10 | Maximum number of results |
| `--output` | No | json | Output format (json, table) |

## Examples

```bash
# Basic search
mapj tdn search "REST API authentication"

# Search with space filter
mapj tdn search "invoice" --space PROT --limit 5

# Human-readable output
mapj tdn search "WebService" --output table
```

## Output Schema

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
        "space": { "key": "PROT", "name": "Protheus" },
        "excerpt": "...matched text excerpt..."
      }
    ],
    "count": 1,
    "total": 42
  },
  "schemaVersion": "1.0"
}
```

## CQL Query Language

TDN uses Confluence Query Language (CQL) internally. The CLI automatically constructs CQL from your search terms. Advanced users can combine multiple filters:

- `text ~ "query"` - Full-text search
- `space = "KEY"` - Space filter
- `label = "tag"` - Label filter
- `type = "page"` - Content type

## Prerequisites

1. Authenticate first:
```bash
mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT_TOKEN
```

2. Verify auth:
```bash
mapj auth status
```

## Error Cases

- **NOT_AUTHENTICATED**: Run `mapj auth login tdn` first
- **SEARCH_ERROR**: Server error - retry recommended
