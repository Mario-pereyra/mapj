---
name: mapj
description: "CLI tool for LLM agents to search TDN documentation, export Confluence pages, and query Protheus database"
metadata:
  version: 1.0.0
  language: en
---

# mapj CLI

CLI tool for LLM/AI agents to interact with TOTVS ecosystem.

## Commands

| Command | Description |
|---------|-------------|
| `mapj tdn search <query>` | Search TDN documentation |
| `mapj confluence export <url-or-id>` | Export Confluence page |
| `mapj protheus query <sql>` | Execute SELECT on Protheus DB |
| `mapj auth login <service>` | Authenticate a service |
| `mapj auth status` | Show auth status |

## Authentication

Each service requires authentication before use:

```bash
# TDN (TOTVS Developer Network)
mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT_TOKEN

# Confluence
mapj auth login confluence --url https://your-company.atlassian.net --token YOUR_API_TOKEN

# Protheus Database
mapj auth login protheus --server 192.168.1.100 --port 1433 --database PROTHEUS --user admin --password secret
```

## Output Format

All commands return JSON by default. Use `--output table` for human-readable output.

```json
{
  "ok": true,
  "command": "mapj tdn search \"REST API\"",
  "result": { ... },
  "schemaVersion": "1.0"
}
```

## Error Handling

Errors return exit codes:
- 0: Success
- 1: General error
- 2: Usage error (invalid arguments)
- 3: Auth error (not logged in)
- 4: Retry recommended (rate limit, server error)
- 5: Conflict

```json
{
  "ok": false,
  "command": "mapj protheus query \"INSERT INTO...\"",
  "error": {
    "code": "USAGE_ERROR",
    "message": "query contains forbidden keyword: INSERT"
  }
}
```

## Command-Specific Documentation

- [mapj-tdn-search](mapj-tdn-search.md) - Search TDN documentation
- [mapj-confluence-export](mapj-confluence-export.md) - Export Confluence pages
- [mapj-protheus-query](mapj-protheus-query.md) - Query Protheus database
