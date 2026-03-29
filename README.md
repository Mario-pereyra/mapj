# mapj — CLI for TOTVS Ecosystem

> Search TDN documentation, export Confluence pages to Markdown, and query Protheus ERP — designed for **LLM agent consumption** and human productivity.

[![Go 1.23+](https://img.shields.io/badge/go-1.23+-blue)](https://go.dev) [![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## What it does

| Task | Command | Output |
|------|---------|--------|
| Search TDN docs | `mapj tdn search "REST API"` | JSON with matching pages |
| Search with child count | `mapj tdn search "AdvPL" --check-children` | JSON + `childCount` per page |
| Search → export pipeline | `mapj tdn search "AdvPL" --space PROT --export-to ./docs` | Exports all found pages |
| Export Confluence page | `mapj confluence export <url-or-id>` | Markdown file + YAML front matter |
| Export full page tree | `mapj confluence export <url> --with-descendants` | Full directory tree |
| Export entire space | `mapj confluence export-space <key> --output-path ./docs` | Full directory tree |
| Query Protheus ERP | `mapj protheus query "SELECT TOP 10 * FROM SA1010"` | JSON with columns/rows |
| Save query to file | `mapj protheus query "SELECT ..." --output-file ./result.json` | File (only summary to stdout) |
| Manage DB connections | `mapj protheus connection list/add/use/ping` | Named profiles |
| List TDN spaces | `mapj tdn spaces list` | All available spaces |

All commands output **compact JSON by default** (LLM-optimized). Use `-o json` for human-readable indented output.

---

## Installation

### Prerequisites
- Go 1.23+ (`go version`)
- VPN access to TOTVS network (for internal Protheus/Confluence instances)

### Option A — Pre-built Windows executable

Download `mapj.exe` from the [Releases](../../releases) page and add it to your PATH:
```powershell
# Move to a directory in your PATH
Move-Item .\mapj.exe "$env:LOCALAPPDATA\Programs\mapj\mapj.exe"
```

### Option B — Build from source

```bash
git clone <repo-url>
cd mapj_cli

# Development build
go build -o mapj.exe ./cmd/mapj

# Production build (smaller, stripped debug symbols)
go build -ldflags="-s -w" -o mapj.exe ./cmd/mapj
```

### Verify

```bash
mapj --help         # full onboarding guide + command inventory
mapj auth status    # check what's already authenticated
```

---

## Quick Start

### 1 — Authenticate (one-time setup)

```bash
# TDN public (tdn.totvs.com) — no auth needed for public content

# Confluence Server/DC (e.g. tdninterno.totvs.com) — Bearer PAT, NO --username
mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_PAT

# Confluence Cloud (company.atlassian.net) — Basic Auth
mapj auth login confluence \
  --url https://company.atlassian.net \
  --username you@company.com \
  --token YOUR_API_TOKEN

# Protheus — add named connection profiles
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB \
  --use
```

### 2 — Use it

```bash
# TDN search (no auth required for public TDN)
mapj tdn search "AdvPL" --space PROT --limit 10
mapj tdn search "ponto de entrada" --space PROT --since 1m --check-children

# Confluence export
mapj confluence export "https://tdn.totvs.com/display/PROT/AdvPL" --output-path ./docs
mapj confluence export 235312129 --output-path ./docs --with-descendants   # 171 pages

# Protheus query
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"
mapj protheus query "SELECT * FROM SA1010" --output-file ./result.json     # don't flood context
```

---

## Output Format

### LLM mode (default — compact, no metadata noise)

```bash
mapj tdn search "AdvPL" --space PROT --limit 1
```

```json
{"ok":true,"command":"mapj tdn search","result":{"results":[{"id":"235312129","type":"page","title":"AdvPL","url":"https://tdn.totvs.com/display/PROT/AdvPL","childCount":1}],"count":1,"total":1,"hasNext":true,"cql":"siteSearch ~ \"AdvPL\" AND space = \"PROT\" AND type = page"}}
```

### Human mode (`-o json`)

```bash
mapj tdn search "AdvPL" --space PROT --limit 1 -o json
```

```json
{
  "ok": true,
  "command": "mapj tdn search",
  "result": { "..." },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-29T00:07:08Z"
}
```

### Error envelope

```json
{"ok":false,"command":"mapj protheus query","error":{"code":"USAGE_ERROR","message":"query contains forbidden keyword: INSERT","hint":"Only SELECT queries are allowed. Rewrite without INSERT/UPDATE/DELETE/EXEC.","retryable":false}}
```

`hint` provides actionable recovery guidance for the agent.

### Output flag options

| Flag | Format | Use for |
|------|--------|---------|
| *(default)* | Compact JSON | LLM/agent consumption |
| `-o json` | Indented JSON + metadata | Human debugging |
| `-o csv` | RFC 4180 CSV | Protheus results → spreadsheet |

### Exit codes

| Code | Meaning | Agent action |
|------|---------|--------------|
| `0` | Success | Parse `result` |
| `1` | General error | Read `error.message` |
| `2` | Usage error (bad args, forbidden SQL) | Fix command |
| `3` | Auth error | Run `mapj auth login <service>` |
| `4` | Retryable (timeout, rate limit) | Wait 2s, retry ≤3× |

---

## Authentication

Credentials stored encrypted at `~/.config/mapj/credentials.enc` (AES-256-GCM).

```bash
mapj auth status
# {"ok":true,"command":"mapj auth status","result":{"tdn":{"authenticated":false},"confluence":{"authenticated":true,"url":"https://tdninterno.totvs.com"},"protheus":{"authenticated":true,"activeProfile":"TOTALPEC_BIB","server":"192.168.99.102","database":"P1212410_BIB","totalProfiles":7}}}
```

### Auto-detection

| URL contains | Auth scheme | Flags required |
|---|---|---|
| `atlassian.net` | Basic (email + API token) | `--username` + `--token` |
| anything else | Bearer PAT | `--token` only |

> ⚠️ Never add `--username` for Server/DC URLs — causes 401.

---

## TDN Search — Key Flags

```bash
mapj tdn search "query" [flags]
```

| Flag | Description |
|------|-------------|
| `--space PROT` | Filter to a specific space |
| `--spaces PROT,TEC` | Filter to multiple spaces |
| `--label versao_12` | Filter by label |
| `--labels l1,l2` | Multiple labels (AND logic) |
| `--since 1m` | Pages updated in last 1 month (also: `1w`, `2y`, ISO date) |
| `--ancestor 187531295` | Descendants of a specific page |
| `--type page\|attachment` | Filter by content type |
| `--limit N` | Result count (default: 25, max: 100) |
| `--start N` | Pagination offset |
| `--check-children` | Add `childCount` to each result (concurrent API calls) |
| `--export-to ./dir` | Search → export pipeline: export all found pages to directory |

> ⚠️ **`childCount` ≠ total descendants**: `childCount: 1` means 1 direct child — but that subtree can have 171+ pages. Always preview before `--with-descendants`.

---

## Confluence Export — Key Flags

```bash
mapj confluence export <url-or-id> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output-path PATH` | stdout | Directory to save files |
| `--with-descendants` | false | Export full page tree recursively |
| `--with-attachments` | false | Download images and files |
| `--format` | markdown | `markdown`, `html`, `json` |
| `--verbose` | false | Show per-page progress |
| `--debug` | false | Save raw HTML to `.debug/` |

### Output structure

```
output-path/
├── spaces/SPACE_KEY/
│   ├── README.md                    ← index with all pages
│   ├── pages/PAGE_ID-slug.md        ← one per page (YAML front matter)
│   └── attachments/PAGE_ID/         ← with --with-attachments
├── manifest.jsonl                   ← one JSON line per exported page
└── export-errors.jsonl              ← one JSON line per failure
```

### Retry failed exports

```bash
mapj confluence retry-failed --output-path ./docs
mapj confluence retry-failed --output-path ./docs --error-code HTTP_TIMEOUT
```

---

## Protheus Query — Key Flags

```bash
mapj protheus query "<SQL SELECT>" [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | json | `json` or `csv` |
| `--max-rows` | 10000 | Client-side row cap |
| `--connection NAME` | (active) | Run against specific profile without switching |
| `--output-file PATH` | stdout | Write result to file; stdout gets summary only |

**Security — SELECT-only enforcement:**  
`INSERT` `UPDATE` `DELETE` `MERGE` `CREATE` `ALTER` `DROP` `TRUNCATE` `EXEC` `INTO` `REPLACE` `GRANT` `REVOKE` `BACKUP` are all blocked.

> `SELECT INTO #temp` is blocked — use CTEs: `WITH t AS (SELECT ...) SELECT * FROM t`

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `401` on tdninterno | Old credentials with wrong auth type | `mapj auth login confluence --url ... --token TOKEN` (no `--username`) |
| `PAGE_NOT_FOUND` | Wrong URL or private page | Try `pageId=` URL or check access |
| `i/o timeout` | VPN not connected | Connect TOTALPEC VPN (192.168.99.x) or UNION VPN (192.168.7.x) |
| `forbidden keyword: INTO` | `SELECT INTO #temp` | Use CTE: `WITH t AS (SELECT ...) SELECT * FROM t` |
| `Invalid object name` | Wrong database active | `mapj protheus connection use CORRECT_PROFILE` |
| Large result floods context | No file output | Add `--output-file ./result.json` to Protheus query |

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `MAPJ_ENCRYPTION_KEY` | 32-byte key to encrypt credentials (CI/CD). If unset: derived from hostname+user |

---

## Project Structure

```
mapj/
├── cmd/mapj/main.go         # Entry point
├── internal/
│   ├── cli/                 # Command definitions (tdn.go, confluence.go, protheus.go, auth.go)
│   ├── auth/                # Credential storage + login/logout/status (AES-256-GCM)
│   ├── errors/              # Exit codes
│   └── output/              # LLM-optimized JSON formatters (llm/human/csv)
├── pkg/
│   ├── confluence/          # Confluence + TDN REST API client
│   └── protheus/            # Protheus SQL Server client
├── skills/                  # Agent skill files (LLM consumption)
│   ├── mapj/SKILL.md        # Main orchestrator skill
│   ├── mapj-tdn-search/     # TDN search skill + CQL reference
│   ├── mapj-confluence-export/ # Export skill + auth guide
│   └── mapj-protheus-query/ # Protheus skill + connection/security refs
└── docs/                    # Human-readable guides
    ├── confluence-export-guide.md
    └── protheus-guide.md
```

---

## Agent Skills (LLM)

> **Auto-discovery first:** Run `mapj --help` and `mapj <command> --help`.
> Every command is self-describing: output schema, GOTCHAs, and next steps are inline.
> Skills provide deeper context for complex multi-step workflows.

The `skills/` directory contains structured documentation for LLM agents:

```bash
# Main orchestrator skill — routes to sub-skills
skills/mapj/SKILL.md

# Sub-skills (load when use case matches)
skills/mapj-tdn-search/SKILL.md           # TDN search + --check-children + pipeline
skills/mapj-confluence-export/SKILL.md    # Export auth, URL formats, retry-failed
skills/mapj-protheus-query/SKILL.md       # Query workflow, --output-file, connections
skills/mapj-protheus-query/references/    # flags.md, connections.md, security.md
```

---

## Further Reading

- [`docs/confluence-export-guide.md`](docs/confluence-export-guide.md) — Detailed Confluence guide
- [`docs/protheus-guide.md`](docs/protheus-guide.md) — Detailed Protheus guide  
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — Development guide
- [`CHANGELOG.md`](CHANGELOG.md) — Version history
