---
name: mapj
description: >
  CLI tool for AI agents to interact with TOTVS ecosystem.
  Use when: searching TDN documentation, exporting Confluence pages to markdown,
  exporting with descendants, exporting spaces, downloading attachments, retrying failed exports,
  querying Protheus ERP database, managing Protheus connection profiles (add/list/switch/ping/remove),
  or comparing data between Protheus environments.
  Do NOT use for writing to Confluence, modifying Protheus data, or any DML/DDL operations.
  Triggers: "search TDN", "export confluence", "export to markdown", "export with descendants",
  "export space", "export attachments", "retry failed export", "query Protheus",
  "SELECT from Protheus", "look up TOTVS docs", "Protheus connection", "list connections",
  "switch database", "ping Protheus", "test connection", "add connection profile".
compatibility: Requires Go 1.23+ built binary at PATH. Network access to TDN/Confluence and Protheus SQL Server (VPN required for internal servers).
metadata:
  version: 2.0.0
  language: en
  author: Mario Pereira
  license: MIT
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
    - connection-management
related:
  - mapj-tdn-search
  - mapj-confluence-export
  - mapj-protheus-query
allowed-tools: Bash
---

# mapj CLI — Agentic Tool for TOTVS Ecosystem

**mapj** is an agentic CLI designed for AI agents to interact with TOTVS enterprise systems.
All output is JSON with a consistent envelope. All operations are read-only (no data modification).

> ⚠️ **Documentation mandate:** Any change to commands, flags, behavior, or data models
> MUST update: the relevant skill file, the relevant `docs/` guide, and `CONTRIBUTING.md`.
> Never let code diverge from docs.

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
| `mapj tdn search <query>` | Search TDN/Confluence documentation | JSON results array |
| `mapj confluence export <url-or-id>` | Export single page → Markdown | JSON or files on disk |
| `mapj confluence export <url> --with-descendants` | Export page tree recursively | Files on disk, manifest.jsonl |
| `mapj confluence export-space <key>` | Export all pages in a space | Files on disk, manifest.jsonl |
| `mapj confluence retry-failed` | Re-export pages that failed | Files on disk, updated logs |
| `mapj protheus query <sql>` | Execute SELECT on Protheus DB | JSON columns/rows |
| `mapj protheus query <sql> --connection NAME` | Query specific profile without switching active | JSON columns/rows |
| `mapj protheus connection list` | List all saved connection profiles | Text table, active marked |
| `mapj protheus connection add <name>` | Register a named connection profile | Status message |
| `mapj protheus connection use <name>` | Switch active profile (no re-login) | Status message |
| `mapj protheus connection ping [name]` | Test connectivity (VPN hint on failure) | Status with latency |
| `mapj protheus connection show [name]` | Show profile details (password masked) | Text |
| `mapj protheus connection remove <name>` | Delete a profile | Status message |
| `mapj auth login <service>` | Authenticate a service | Status message |
| `mapj auth status` | Show auth status + active Protheus profile | Text |
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

Credentials stored encrypted at `~/.config/mapj/credentials.enc` (AES-256-GCM, machine-bound key).

### Decision tree: which auth to use?

```
URL contains atlassian.net?
├── YES → Basic Auth: --username email + --token API_TOKEN
└── NO  → Bearer PAT: --token TOKEN only (NEVER add --username → causes 401)
```

### TDN / Confluence Server (tdninterno.totvs.com)

```bash
# ✅ CORRECT — Bearer PAT, no --username
mapj auth login confluence \
  --url https://tdninterno.totvs.com \
  --token YOUR_PAT_TOKEN
```

### Confluence Cloud (company.atlassian.net)

```bash
# ✅ CORRECT — Basic Auth with email
mapj auth login confluence \
  --url https://company.atlassian.net \
  --username your-email@company.com \
  --token YOUR_CLOUD_API_TOKEN
```

### Protheus — named profiles (v2 model)

```bash
# Register
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB --use

# Switch (no credentials re-entry)
mapj protheus connection use TOTALPEC_PRD

# Test connectivity
mapj protheus connection ping [name]
```

> See [mapj-protheus-query skill](mapj-protheus-query.md) for full connection management reference.

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
