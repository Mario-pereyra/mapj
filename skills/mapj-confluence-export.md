---
name: mapj-confluence-export
description: >
  Export Confluence pages to Markdown files with full metadata, recursive child pages, and attachments.
  Use when: exporting single pages, full page trees with descendants, entire Confluence spaces,
  retrying failed exports, downloading page attachments, or converting Confluence docs to LLM-ready Markdown.
  Do NOT use for creating or editing Confluence content, searching Confluence, or querying Protheus.
  Triggers: "export confluence", "export to markdown", "download confluence page", "export with children",
  "export space", "with descendants", "confluence to markdown", "retry failed export",
  "export attachments", "bulk export confluence".
compatibility: Requires mapj CLI at PATH and network access to the Confluence instance. Bearer PAT for Server/DC, Basic Auth for Cloud (atlassian.net).
metadata:
  version: 2.0.0
  language: en
  author: Mario Pereira
  tags:
    - confluence
    - export
    - markdown
    - tdn
    - totvs
    - llm
    - agentic
    - bulk-export
    - attachments
  capabilities:
    - export-page
    - export-recursive
    - export-space
    - export-attachments
    - retry-failures
    - url-resolution
  related:
    - mapj-tdn-search
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj confluence export — Agent Skill v2.0

Export Confluence pages to structured Markdown files on disk. Supports single pages, recursive
descendants, full spaces, and attachment downloads. Output is LLM-ready with YAML front matter
and `manifest.jsonl` for fast lookup.

---

## Prerequisites: Authentication

**ALWAYS verify auth before executing any export command.**

```bash
# Check if Confluence is already authenticated
mapj auth status
```

Expected when configured:
```
  Confluence: ✓ configured
```

### Auth Auto-Detection

The CLI automatically selects the correct auth scheme based on the URL:

| URL contains | Auth type | HTTP Header |
|---|---|---|
| `atlassian.net` | `basic` (email + API token) | `Authorization: Basic base64(email:token)` |
| anything else | `bearer` (PAT) | `Authorization: Bearer TOKEN` |

> ✅ You do NOT need to specify the auth type for common cases. It's detected automatically.

### If NOT configured — Login

#### Confluence Server / Data Center (e.g., tdninterno.totvs.com) — Bearer PAT

```bash
# Auto-detected: URL is not atlassian.net → bearer auth
mapj auth login confluence \
  --url "https://tdninterno.totvs.com" \
  --token "THE_PAT_TOKEN"

# Output: Confluence login successful (auth: bearer)
```

> ⚠️ Do NOT add `--username` here. If you do, the CLI will warn you and ignore it.
> Adding `--username` was the root cause of the 401 bug on Server/DC instances.

#### Confluence Cloud (Atlassian) — Basic Auth

```bash
# Auto-detected: URL contains atlassian.net → basic auth (requires --username)
mapj auth login confluence \
  --url "https://company.atlassian.net" \
  --username "user@company.com" \
  --token "CLOUD_API_TOKEN"

# Output: Confluence login successful (auth: basic)
```

#### Force Override (Edge Cases)

```bash
# Force basic auth on a non-atlassian server
mapj auth login confluence \
  --url "https://my-confluence.company.com" \
  --username "user" \
  --token "pass" \
  --auth-type basic
```

#### Public Confluence (e.g., tdn.totvs.com)

Public pages do NOT require auth. The CLI automatically falls back to HTML scraping
to resolve page IDs from display URLs. Just run the export directly.

> ⚠️ **If you have old credentials stored**, re-run the login to write the `authType`
> field correctly. Old credentials without `authType` default to `bearer` (backward-compatible).

---

## URL Resolution — What the CLI Accepts

The CLI resolves ANY of these input formats automatically:

| Input Type | Example |
|------------|---------|
| Numeric page ID | `22479548` |
| Cloud URL with ID | `https://company.atlassian.net/wiki/spaces/TEAM/pages/12345/Title` |
| Server ViewPage URL | `https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548` |
| Server ReleaseView URL | `https://tdninterno.totvs.com/pages/releaseview.action?pageId=22479548` |
| Display URL (Server) | `https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus` |
| Display URL (public prefix) | `https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus` |

**Resolution cascade (automatic, transparent to agent):**
1. Extract ID from URL if present → direct API call
2. Extract space + title → `GET /rest/api/content?spaceKey=...&title=...`
3. CQL search fallback
4. HTML scrape fallback (extract `ajs-page-id` meta tag — WAF bypass, same as GUI)

---

## Commands

### Command 1: `confluence export` — Single Page or Tree

```bash
mapj confluence export <url-or-page-id> [flags]
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output-path PATH` | inline | Directory to save files. Without this, prints JSON to stdout |
| `--format` | `markdown` | Output format: `markdown`, `html`, `json` |
| `--with-descendants` | `false` | Recursively export all child pages |
| `--with-attachments` | `false` | Download binary attachments (images, files) |
| `--verbose` | `false` | Show per-page progress to stderr |
| `--debug` | `false` | Save raw HTML to `.debug/PAGE_ID-body.html` |
| `--dump-debug` | `false` | Full diagnostic dump for a single page |

#### Usage Patterns

```bash
# Inline (no output-path) — returns JSON with content in 'result.content'
mapj confluence export 22479548

# Save to disk
mapj confluence export 22479548 --output-path ./docs

# From any URL format
mapj confluence export \
  "https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus" \
  --output-path ./docs

# Recursive export (page + ALL descendants)
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-descendants

# With attachments
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-descendants \
  --with-attachments

# Verbose progress + debug HTML
mapj confluence export 22479548 \
  --output-path ./docs \
  --verbose --debug
```

---

### Command 2: `confluence export-space` — Full Space Export

```bash
mapj confluence export-space <space-key> --output-path PATH [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output-path PATH` | **required** | Output directory |
| `--with-attachments` | `false` | Download attachments |
| `--verbose` | `false` | Show progress |
| `--debug` | `false` | Save raw HTML |

```bash
# Export all pages in the 'framework' space
mapj confluence export-space framework --output-path ./docs

# With attachments
mapj confluence export-space framework \
  --output-path ./docs \
  --with-attachments \
  --verbose
```

---

### Command 3: `confluence retry-failed` — Retry Only Failed Pages

After a large export, check for errors and retry only failures.

```bash
mapj confluence retry-failed --output-path PATH [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output-path PATH` | **required** | Same dir used in original export (contains `export-errors.jsonl`) |
| `--error-code CODE` | all | Only retry pages with this specific error code |
| `--with-attachments` | `false` | Download attachments on retry |
| `--verbose` | `false` | Show progress |

```bash
# Retry all failures
mapj confluence retry-failed --output-path ./docs

# Retry only timeout failures
mapj confluence retry-failed --output-path ./docs --error-code HTTP_TIMEOUT

# Retry with attachments this time
mapj confluence retry-failed --output-path ./docs --with-attachments
```

---

## Output Structure (Disk Mode)

```
output-path/
├── spaces/
│   └── SPACE_KEY/
│       ├── README.md                    ← space index with links to all pages
│       ├── pages/
│       │   └── PAGE_ID-slug-title.md   ← one file per page
│       └── attachments/                 ← created only with --with-attachments
│           └── PAGE_ID/
│               └── filename.ext
├── manifest.jsonl                        ← one JSON line per exported page
└── export-errors.jsonl                   ← one JSON line per failure
```

### Markdown File Format (YAML front matter)

```markdown
---
page_id: "22479548"
title: "SX1 - Perguntas do usuário"
source_url: "https://tdn.totvs.com/pages/viewpage.action?pageId=22479548"
space_key: "framework"
space_name: "Frameworksp"
labels:
  - "sx1"
updated_at: "2025-08-05T09:30:23.513-03:00"
author: "Sandro Constancio Ferreira"
version: 3
exported_at: "2026-03-28T20:48:15Z"
---
# SX1 - Perguntas do usuário

...content in Markdown...
```

### manifest.jsonl Schema

One JSON object per line, parseable with `jq`:

```jsonl
{"page_id":"22479548","title":"SX1 - Perguntas do usuário","slug":"sx1-perguntas-do-usuario","source_url":"...","space_key":"framework","space_name":"Frameworksp","export_path":"spaces/framework/pages/22479548-sx1-perguntas-do-usuario.md","exported_at":"2026-03-28T20:48:15Z"}
```

### export-errors.jsonl Schema

```jsonl
{"ts":"2026-03-28T20:00:00Z","page_id":"123456","title":"Page Title","phase":"FETCH","error_code":"HTTP_TIMEOUT","message":"request timeout","retry_cmd":"mapj confluence export 123456 --output-path ./docs"}
```

**Error codes:** `PAGE_NOT_FOUND`, `HTTP_TIMEOUT`, `CONVERSION_ERROR`, `WRITE_ERROR`, `PATH_TOO_LONG`, `AUTH_ERROR`

---

## Inline Export (stdout) — Schema

When `--output-path` is NOT provided, the result is a JSON envelope:

```json
{
  "ok": true,
  "command": "mapj confluence export 22479548",
  "result": {
    "pageId": "22479548",
    "title": "SX1 - Perguntas do usuário",
    "format": "markdown",
    "content": "---\npage_id: ...\n---\n# SX1...",
    "url": "https://tdn.totvs.com/pages/viewpage.action?pageId=22479548"
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T20:48:15Z"
}
```

Error envelope:
```json
{
  "ok": false,
  "command": "mapj confluence export 99999",
  "error": {
    "code": "PAGE_NOT_FOUND",
    "message": "cannot resolve page: title=\"...\" space=\"...\""
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T20:48:15Z"
}
```

---

## Agentic Decision Tree

```
User wants to export Confluence content
│
├─ Do I know the page ID or URL? ─── YES ──→ mapj confluence export <id-or-url>
│                                              + --output-path if saving to disk
│                                              + --with-descendants if want tree
│                                              + --with-attachments if want files
│
├─ Do I want an entire space? ──────────────→ mapj confluence export-space <key>
│                                              --output-path REQUIRED
│
└─ Retrying after partial failure? ─────────→ mapj confluence retry-failed
                                               --output-path SAME_DIR
```

---

## Agent Workflow: Large Export with Error Handling

```bash
#!/bin/bash
EXPORT_DIR="./confluence-docs"

# 1. Verify auth
if ! mapj auth status | grep -q "Confluence: ✓"; then
  echo "ERROR: Confluence not authenticated"
  exit 1
fi

# 2. Export with descendants
mapj confluence export \
  "https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus" \
  --output-path "$EXPORT_DIR" \
  --with-descendants \
  --verbose

# 3. Check for errors
ERROR_COUNT=$(wc -l < "$EXPORT_DIR/export-errors.jsonl" 2>/dev/null || echo 0)

if [ "$ERROR_COUNT" -gt 0 ]; then
  echo "⚠️  $ERROR_COUNT failures. Retrying..."
  mapj confluence retry-failed --output-path "$EXPORT_DIR"
fi

# 4. Find page by title in manifest
grep "REST API" "$EXPORT_DIR/manifest.jsonl" | jq -r '.export_path'
```

---

## Common Errors & Solutions

| Error | Cause | Fix |
|-------|-------|-----|
| `401 Basic Authentication Failure` | Used `--username` with PAT on Server | Re-login WITHOUT `--username` |
| `PAGE_NOT_FOUND` + 401 on all methods | Token expired or wrong instance URL | Re-login with updated token |
| `PAGE_NOT_FOUND` + CQL 401 + scrape OK | API auth fails but public page works | Expected for public `tdn.totvs.com` — scrape fallback handles it automatically |
| `PATH_TOO_LONG` | Windows path > 260 chars | Use shorter `--output-path` or enable long paths in Windows |
| `HTTP_TIMEOUT` on large pages | 30s default timeout | Use `retry-failed`, usually resolves on retry |

---

## Performance Reference (Observed)

| Test | Pages | Time | Result |
|------|-------|------|--------|
| Single page | 1 | ~8s | 100% |
| Page + 1 child | 2 | ~12s | 100% |
| SDK Microsiga Protheus (full tree) | 785 | ~6 min | 100%, 0 errors |

---

## See Also

- [confluence-export-guide.md](../docs/confluence-export-guide.md) — User guide (español)
- [mapj-tdn-search.md](mapj-tdn-search.md) — Find pages to export
- [mapj-protheus-query.md](mapj-protheus-query.md) — Query Protheus database
- [SKILL.md](SKILL.md) — Main tool manifest
