---
name: mapj
description: >
  CLI tool for AI agents to interact with TOTVS ecosystem (TDN/Confluence documentation and Protheus ERP database).
  Use when: searching TDN documentation, exporting Confluence pages to markdown/HTML/JSON, querying Protheus database tables,
  looking up TOTVS Developer Network articles, fetching technical documentation, or executing SELECT queries on Protheus SQL Server.
  Triggers: "search TDN", "find documentation", "export confluence page", "export to markdown", "query Protheus",
  "SELECT from Protheus", "look up TOTVS docs", "get API documentation", "SQL query Protheus".
compatibility: Requires Go 1.21+. Network access to TDN (tdninterno.totvs.com), Confluence instances, and Protheus SQL Server.
metadata:
  version: 1.0.0
  language: en
  author: Mario Pereira
  license: MIT
  repository: https://github.com/Mario-pereyra/mapj
  tags:
    - totvs
    - protheus
    - confluence
    - tdn
    - erp
    - database
    - documentation
    - cli
    - agentic
  capabilities:
    - search
    - export
    - query
    - authentication
related:
  - mapj-tdn-search
  - mapj-confluence-export
  - mapj-protheus-query
allowed-tools: Bash
---

# mapj CLI - Agentic Command-Line Tool for TOTVS Ecosystem

**mapj** is an agentic CLI designed for AI agents to interact with TOTVS enterprise systems. It provides machine-readable JSON output, structured error handling, and idempotent operations.

## Quick Start

```bash
# Authenticate (one-time setup)
mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT_TOKEN
mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_API_TOKEN
mapj auth login protheus --server 192.168.99.102 --port 1433 --database P1212410_BIB --user USER --password PASS

# Search TDN documentation
mapj tdn search "REST API authentication"

# Export Confluence page
mapj confluence export 573675873 --format markdown

# Query Protheus database
mapj protheus query "SELECT TOP 10 * FROM SA1010"
```

## Commands Overview

| Command | Purpose | Output |
|---------|---------|--------|
| `mapj tdn search <query>` | Search TDN/Confluence documentation | JSON with results array |
| `mapj confluence export <url-or-id>` | Export page to markdown/HTML/JSON | JSON with content field |
| `mapj protheus query <sql>` | Execute SELECT on Protheus DB | JSON with columns/rows |
| `mapj auth login <service>` | Authenticate a service | Status message |
| `mapj auth status` | Show auth status for all services | JSON summary |
| `mapj auth logout <service>` | Remove credentials | Status message |

## Output Format

All commands return **JSON by default** with a consistent envelope structure.

### Success Response

```json
{
  "ok": true,
  "command": "mapj tdn search \"REST API\"",
  "result": { ... },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### Error Response

```json
{
  "ok": false,
  "command": "mapj protheus query \"INSERT INTO...\"",
  "error": {
    "code": "USAGE_ERROR",
    "message": "query contains forbidden keyword: INSERT",
    "retryable": false
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

## Exit Codes

| Code | Name | Meaning | Action |
|------|------|---------|--------|
| 0 | SUCCESS | Operation completed successfully | - |
| 1 | ERROR | General error | Check error.message, fix issue |
| 2 | USAGE_ERROR | Invalid arguments or syntax | Review command usage |
| 3 | AUTH_ERROR | Not authenticated or invalid credentials | Run `mapj auth login <service>` |
| 4 | RETRYABLE | Transient error (rate limit, server error) | Wait and retry with backoff |
| 5 | CONFLICT | Resource conflict | Check current state |

## Authentication

Credentials are stored encrypted at `~/.config/mapj/credentials.enc` using AES-256-GCM.

### TDN / Confluence (Same System)

TDN is a Confluence Cloud instance. It supports **two auth methods**:

```bash
# Method 1: API Token + Email (Basic Auth) - For Python atlassian library
mapj auth login confluence \
  --url https://tdninterno.totvs.com \
  --username your-email@company.com \
  --token YOUR_API_TOKEN

# Method 2: PAT Token (Bearer Auth) - Direct API access
mapj auth login tdn \
  --url https://tdninterno.totvs.com \
  --token YOUR_PAT_TOKEN
```

### Protheus Database

```bash
mapj auth login protheus \
  --server 192.168.99.102 \
  --port 1433 \
  --database P1212410_BIB \
  --user USERNAME \
  --password PASSWORD
```

## Self-Discovery for AI Agents

```bash
# List all commands (useful for agent tool discovery)
mapj --help

# Get command-specific help as JSON
mapj tdn search --help

# Verify authentication status
mapj auth status
```

## Error Handling Best Practices

1. **Always check exit code**: `if [ $? -eq 0 ]; then ...`
2. **Parse JSON**: Use `jq` for reliable parsing: `mapj tdn search "query" | jq '.result.results[].id'`
3. **Retry on code 4**: Implement exponential backoff for RETRYABLE errors
4. **Validate before query**: For Protheus, use SELECT-only queries to avoid data corruption

## Limitations

- **Protheus queries**: Only SELECT statements allowed (security constraint)
- **Rate limiting**: TDN/Confluence may rate-limit; implement 1s delay between requests
- **Page size**: Large pages may timeout; consider fetching by section
- **Authentication expiry**: Tokens may expire; re-authenticate if 401 errors occur

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `MAPJ_CONFIG_DIR` | Override config directory | `~/.config/mapj` |
| `MAPJ_OUTPUT` | Default output format | `json` |
| `MAPJ_TIMEOUT` | Request timeout in seconds | `30` |

## Related Skills

- [mapj-tdn-search](mapj-tdn-search.md) - TDN/Confluence documentation search
- [mapj-confluence-export](mapj-confluence-export.md) - Export Confluence pages
- [mapj-protheus-query](mapj-protheus-query.md) - Protheus database queries

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.
