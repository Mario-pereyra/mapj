---
name: mapj-tdn-search
description: >
  Search TOTVS Developer Network (TDN) documentation, a Confluence-based system at tdninterno.totvs.com.
  Use when: finding technical documentation, looking up Protheus API references, searching ADVPL guides,
  finding ERP customization docs, or any TOTVS technical search.
  Do NOT use when: exporting pages (use mapj-confluence-export), querying Protheus database
  (use mapj-protheus-query), or searching non-TDN content.
  Triggers: "search TDN", "find documentation", "look up TOTVS", "TDN search",
  "search Protheus docs", "find API docs", "ADVPL documentation", "search framework".
compatibility: Requires mapj CLI at PATH and network access to tdninterno.totvs.com with valid PAT.
metadata:
  version: 1.1.0
  language: en
  author: Mario Pereira
  tags:
    - tdn
    - totvs
    - confluence
    - documentation
    - search
  capabilities:
    - search
    - filter-by-space
    - filter-by-limit
  related:
    - mapj-confluence-export
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj tdn search — Agent Skill v1.1

Search documentation in TOTVS Developer Network (TDN), a Confluence-based system.

---

## Prerequisites

```bash
# TDN uses Bearer PAT — no --username
mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT

# Verify
mapj auth status   # → "TDN: ✓ authenticated"
```

---

## Usage

```
mapj tdn search "<query>" [--space KEY] [--limit N] [--output json|table]
```

| Flag | Default | Description |
|---|---|---|
| `--space` | (all) | Filter by space key (e.g., `PROT`, `LDT`, `FLUIG`) |
| `--limit` | 10 | Max results (1–100) |
| `--output` | json | `json` or `table` (human-readable) |

---

## Common Searches

```bash
# Basic search
mapj tdn search "REST API authentication"

# Filter to Protheus space, limit results
mapj tdn search "invoice processing" --space PROT --limit 5

# Human-readable output
mapj tdn search "WebService" --output table

# ADVPL-specific search
mapj tdn search "MT0795" --space LDT --limit 20
```

---

## Known Spaces

| Key | Name |
|---|---|
| `PROT` | Protheus ERP |
| `LDT` | Linha Datasul |
| `FLUIG` | Fluig Platform |
| `EngMP` | Engineering Protheus |

---

## Output Schema (JSON)

```json
{
  "ok": true,
  "result": {
    "results": [
      {
        "id": "123456789",
        "title": "REST API Authentication Guide",
        "url": "https://tdninterno.totvs.com/...",
        "space": { "key": "PROT", "name": "Protheus" },
        "excerpt": "...matched text..."
      }
    ],
    "count": 3,
    "total": 42
  }
}
```

---

## Error Reference

| Code | Condition | Fix |
|---|---|---|
| `AUTH_ERROR` | Not authenticated | `mapj auth login tdn ...` |
| `USAGE_ERROR` | `--limit` out of range (1-100) | Adjust limit |
| `SEARCH_ERROR` | Server error | Retry with backoff |
| `RATE_LIMITED` (retryable) | Too many requests | Wait 1s and retry |

---

## After Searching → Export a Found Page

When search returns a page you want to export:
```bash
# Use the ID from search results
mapj confluence export <id-from-search-result> --output-path ./docs

# Or use the URL directly
mapj confluence export "<url-from-search-result>" --output-path ./docs
```

> See `mapj-confluence-export/SKILL.md` for full export workflow.
