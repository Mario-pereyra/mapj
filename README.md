# mapj — CLI for TOTVS Ecosystem

> Export Confluence documentation and query Protheus ERP from the command line.  
> Designed for human productivity and LLM agent consumption.

---

## What it does

`mapj` is a CLI tool that connects to TOTVS enterprise systems:

| What | Command | Result |
|------|---------|--------|
| Search TDN docs | `mapj tdn search "REST API"` | JSON with matching pages |
| Export Confluence page to Markdown | `mapj confluence export <url-or-id>` | Markdown file with YAML front matter |
| Export page tree recursively | `mapj confluence export <url> --with-descendants` | Full directory tree |
| Export entire space | `mapj confluence export-space <key>` | Full directory tree |
| Query Protheus ERP database | `mapj protheus query "SELECT ..."` | JSON with columns/rows |
| Manage saved DB connections | `mapj protheus connection list/add/use/ping` | Named profiles |

---

## Installation

### Prerequisites
- Go 1.23+ (check: `go version`)
- VPN access to TOTVS network (for internal instances)

### Build from source

```bash
git clone <repo-url>
cd mapj_cli

# Build binary
go build -o mapj ./cmd/mapj

# Add to PATH (Windows PowerShell)
Move-Item .\mapj.exe $env:LOCALAPPDATA\Programs\mapj\mapj.exe
# Or just copy to any directory in your PATH
```

### Verify installation

```bash
mapj --help
```

---

## Quick Start

### Step 1 — Authenticate (one-time)

```bash
# TDN internal (tdninterno.totvs.com) — Bearer PAT token only, NO --username
mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_PAT

# TDN public (tdn.totvs.com) — no auth needed, skip this

# Protheus — register named connection profiles
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB \
  --use
```

### Step 2 — Use it

```bash
# Export a Confluence page
mapj confluence export "https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548" \
  --output-path ./docs

# Export full page tree
mapj confluence export 152798711 --output-path ./docs --with-descendants

# Query Protheus
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"
```

---

## Authentication

Credentials are stored encrypted at `~/.config/mapj/credentials.enc` (AES-256-GCM).

### Check status

```bash
mapj auth status
```

```
Authentication Status:
  TDN:        ✗ not configured
  Confluence: ✓ authenticated
  Protheus:   ✓ authenticated  [active: TOTALPEC_BIB → 192.168.99.102/P1212410_BIB | 7 profile(s)]
```

### Confluence — auth type is auto-detected

| URL contains | Auth used | Required flags |
|---|---|---|
| `atlassian.net` | Basic (email + API token) | `--username` + `--token` |
| anything else | Bearer PAT | `--token` only |

```bash
# Server / DC (tdninterno.totvs.com) — Bearer PAT
mapj auth login confluence --url https://tdninterno.totvs.com --token PAT_TOKEN

# Cloud (company.atlassian.net) — Basic Auth
mapj auth login confluence --url https://company.atlassian.net \
  --username you@company.com --token API_TOKEN

# Force auth type if auto-detect is wrong
mapj auth login confluence --url https://... --token TOKEN --auth-type bearer|basic
```

> ⚠️ Adding `--username` to a Server/DC URL activates Basic Auth and causes 401. The CLI warns and ignores it now, but re-login if you have old credentials.

### Protheus — named profiles

```bash
# Add
mapj protheus connection add <name> --server <ip> --database <db> --user <u> --password <p>

# List all + active
mapj protheus connection list

# Switch active (no credentials re-entry)
mapj protheus connection use TOTALPEC_PRD

# Test connectivity (with VPN hint on failure)
mapj protheus connection ping [name]

# Remove
mapj protheus connection remove <name>

# Logout (removes all Protheus credentials)
mapj auth logout protheus
```

---

## Confluence Export — Reference

### Supported URL formats

| Input | Example |
|-------|---------|
| Page ID | `22479548` |
| ViewPage URL | `https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548` |
| Display URL | `https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus` |
| Public display URL | `https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus` |
| Cloud URL | `https://company.atlassian.net/wiki/spaces/TEAM/pages/12345/Title` |

### Export flags

```bash
mapj confluence export <url-or-id> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output-path PATH` | (stdout) | Directory to save files. Without it: JSON to stdout |
| `--with-descendants` | false | Also export all child pages recursively |
| `--with-attachments` | false | Download images and binary attachments |
| `--format` | markdown | `markdown`, `html`, `json` |
| `--verbose` | false | Show per-page progress |
| `--debug` | false | Save raw HTML to `.debug/` |
| `--dump-debug` | false | Full diagnostic dump for a single page |

### Output structure (when using `--output-path`)

```
output-path/
├── spaces/
│   └── SPACE_KEY/
│       ├── README.md                    ← index with links to all pages
│       ├── pages/
│       │   └── PAGE_ID-slug-title.md   ← one file per page (YAML front matter)
│       └── attachments/PAGE_ID/         ← only with --with-attachments
├── manifest.jsonl                        ← one JSON line per exported page
└── export-errors.jsonl                   ← one JSON line per failure
```

### Retry failed pages

```bash
mapj confluence retry-failed --output-path ./docs
mapj confluence retry-failed --output-path ./docs --error-code HTTP_TIMEOUT
```

---

## Protheus Query — Reference

```bash
mapj protheus query "<SQL SELECT>" [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | json | `json` or `csv` |
| `--max-rows` | 10000 | Client-side row cap (`0` = unlimited) |
| `--connection NAME` | (active) | Run against specific profile without switching active |

### Security: SELECT-only enforcement

Only `SELECT` and `WITH` (CTEs) are allowed. The following keywords are blocked at the application level before the query reaches the database:

`INSERT` `UPDATE` `DELETE` `MERGE` `CREATE` `ALTER` `DROP` `TRUNCATE` `EXEC` `EXECUTE` `INTO` `REPLACE` `GRANT` `REVOKE` `DENY` `BACKUP` `RESTORE`

> ⚠️ **`INTO` is blocked**: This includes `SELECT INTO #temp`. Use CTEs instead.

---

## Output Format

All commands return a JSON envelope:

```json
{
  "ok": true,
  "command": "mapj protheus query",
  "result": { "columns": [...], "rows": [[...]], "count": 10 },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T22:00:00Z"
}
```

Error:
```json
{
  "ok": false,
  "command": "mapj confluence export",
  "error": { "code": "PAGE_NOT_FOUND", "message": "...", "retryable": false },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T22:00:00Z"
}
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error (wrong args, forbidden SQL) |
| 3 | Auth error (not configured) |
| 4 | Retryable error (timeout, rate limit) |

---

## Common Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `401` on tdninterno | Old creds have `authType` as basic | `mapj auth login confluence --url ... --token TOKEN` (no --username) |
| `PAGE_NOT_FOUND` | Wrong URL format or private page | Try URL with `pageId=` or check access |
| `ping failed: i/o timeout` | VPN not connected | Connect TOTALPEC VPN (192.168.99.x) or UNION VPN (192.168.7.x) |
| `validation error: forbidden keyword: INTO` | Used SELECT INTO | Use CTE instead: `WITH t AS (SELECT ...) SELECT * FROM t` |
| `Invalid object name 'TABLE'` | Wrong database active | `mapj protheus connection use CORRECT_PROFILE` |

---

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `MAPJ_ENCRYPTION_KEY` | 32-byte key to encrypt/decrypt credentials (optional, uses machine key if not set) | derived from hostname+user |

---

## Further Reading

- [`docs/confluence-export-guide.md`](docs/confluence-export-guide.md) — Detailed Confluence guide
- [`docs/protheus-guide.md`](docs/protheus-guide.md) — Detailed Protheus guide
- [`skills/`](skills/) — Agent skill files (for LLM consumption)
