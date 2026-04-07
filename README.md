# mapj — CLI for TOTVS Ecosystem

> Search TDN documentation, export Confluence pages to Markdown, and query Protheus ERP — designed for **LLM agent consumption** and enterprise productivity.

[![Go 1.23+](https://img.shields.io/badge/go-1.23+-blue)](https://go.dev) [![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## What it does

| Task | Command | Output |
|------|---------|--------|
| Search TDN docs | `mapj tdn search "REST API"` | TOON/JSON matching results |
| Search + Child Count | `mapj tdn search "AdvPL" --check-children` | TOON/JSON + `childCount` |
| Search → Export Pipeline | `mapj tdn search "AdvPL" --export-to ./docs` | Bulk downloads found pages |
| Export Confluence page | `mapj confluence export <url-or-id>` | Markdown file + YAML front matter |
| Export Page Tree | `mapj confluence export <url> --with-descendants` | Concurrent recursive export |
| Export Entire Space | `mapj confluence export-space <key>` | Space-wide high-speed export |
| Query Protheus ERP | `mapj protheus query "SELECT * FROM SA1010"` | TOON tabular results |
| Discover Table Schema | `mapj protheus schema <table_name>` | Columns, types, and lengths |
| Manage DB Connections | `mapj protheus connection list/add/use` | Encrypted named profiles |

All commands output **Auto-detected formats** (TOON for tables, LLM for objects). Use `-o toon` or `-o llm` to force.

---

## Installation

```bash
# Clone and build
git clone <repo-url>
cd mapj_cli
go build -o mapj.exe ./cmd/mapj
```

---

## Quick Start

### 1 — Authenticate (one-time setup)

```bash
# Confluence (Server/DC) — Bearer PAT
mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_PAT

# Protheus — add named connection profile
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --database P1212410_BIB --user U --password P --use
```

### 2 — Use it

```bash
# TDN search (Auto-paginated)
mapj tdn search "AdvPL" --space PROT --max-results 50

# Confluence export (Concurrent & Auto-Healing)
mapj confluence export 235312129 --output-path ./docs --with-descendants

# Protheus query (Prefix-validated & Safety Tripwire)
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"
```

---

## Agentic Features (CLI v0.2.0)

- **TOON Format**: Native support for **Tabular Object Notation**. Anchors column headers and lists data in a YAML-like format, saving ~40% tokens for tabular results compared to JSON.
- **Safety Tripwire**: Protheus queries returning > 500 rows are automatically diverted to a temporary `.toon` file. The CLI returns a summary instead of flooding the agent's context.
- **Auto-Healing**: Confluence client handles network instability (429/50x) with native exponential backoff.
- **High Concurrency**: Worker pools parallelize heavy exports, processing trees 5-10x faster.
- **Prefix Validation**: Security enforcement ensures queries start with `SELECT`, `WITH`, or `EXEC`, preventing SQL bypasses.
- **Early Cursor Closure**: `--max-rows` aborts DB processing at the server level as soon as the limit is hit.

---

## Output Formats

| Flag | Format | Use for |
|------|--------|---------|
| *(default)* | `auto` | Best fit: TOON for lists/tables, LLM for others |
| `-o llm` | Compact JSON | Machine-readable deterministic parsing |
| `-o toon` | Tabular YAML | Highest token efficiency for agents |
| `-o json` | Pretty JSON | Human debugging (includes metadata) |

---

## Exit Codes

| Code | Meaning | Agent Action |
|------|---------|--------------|
| `0` | Success | Parse result |
| `1` | General error | Read `error.message` |
| `2` | Usage error | Fix syntax or SQL prefix |
| `3` | Auth error | Re-run `mapj auth login` |
| `4` | Retryable | Wait 2s, retry up to 3x |

---

## Extended Documentation

- [`docs/confluence-export-guide.md`](docs/confluence-export-guide.md) — Concurrent export & resiliencia
- [`docs/protheus-guide.md`](docs/protheus-guide.md) — Query security & schema discovery
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — Architecture & developer guide
- [`CHANGELOG.md`](CHANGELOG.md) — Version history (v0.2.0-agentic)

---

## Agent Skills (LLM)

Load specialized skills for complex workflows:
- `skills/mapj/SKILL.md` — Main orchestrator
- `skills/mapj-tdn-search/SKILL.md` — Search & Auto-pagination
- `skills/mapj-confluence-export/SKILL.md` — Exports & Attachments
- `skills/mapj-protheus-query/SKILL.md` — SQL Query & Connections
