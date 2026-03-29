---
name: mapj-confluence-export
description: >
  Export Confluence pages to Markdown files with YAML front matter, metadata, and structure.
  Use when: exporting a single Confluence page, exporting a page tree recursively (with descendants),
  exporting an entire Confluence space, downloading page attachments, or retrying failed exports.
  Do NOT use when: searching Confluence content (use mapj-tdn-search), creating or modifying
  Confluence pages, or querying Protheus database.
  Triggers: "export confluence", "export to markdown", "download confluence page",
  "export with children", "export space", "with descendants", "confluence to markdown",
  "retry failed export", "export attachments", "bulk export confluence".
compatibility: Requires mapj CLI at PATH. Bearer PAT for Server/DC, Basic Auth for atlassian.net.
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
    - bulk-export
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

Export Confluence pages to Markdown files with YAML front matter. Supports single pages,
recursive page trees, full spaces, and binary attachments.

> ⚠️ **Auth matters**: Using `--username` on Server/DC (non-atlassian.net) causes 401.
> Bear PAT is the correct auth for `tdninterno.totvs.com` and similar instances.

---

## Prerequisites — One-Time Auth Setup

```
URL contains atlassian.net?
├─ YES → Basic Auth (email + API token)
│        mapj auth login confluence --url URL --username EMAIL --token TOKEN
└─ NO  → Bearer PAT only (NEVER add --username → 401)
          mapj auth login confluence --url URL --token TOKEN
```

```bash
# Verify auth is configured
mapj auth status
# → {"ok":true,"result":{"confluence":{"authenticated":true,"url":"https://tdninterno.totvs.com"},...}}
```

---

## Export Workflow

```
What do you need to export?
│
├─ Single page (result to stdout)
│   mapj confluence export <url-or-id>
│
├─ Single page to directory
│   mapj confluence export <url-or-id> --output-path ./docs
│
├─ Page + ALL children recursively
│   mapj confluence export <url-or-id> --output-path ./docs --with-descendants
│
├─ Also include binary attachments (images, PDFs...)
│   mapj confluence export <url-or-id> --output-path ./docs \
│     --with-descendants --with-attachments
│
├─ Entire Confluence space
│   mapj confluence export-space <SPACE_KEY> --output-path ./docs
│
└─ Retry only the pages that failed in a previous run
    mapj confluence retry-failed --output-path ./docs
    mapj confluence retry-failed --output-path ./docs --error-code HTTP_TIMEOUT
```

---

## Supported URL Formats

All of these work as the `<url-or-id>` argument:

```bash
# Page ID only
mapj confluence export 22479548

# ViewPage action URL
mapj confluence export "https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548"

# Display URL
mapj confluence export "https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus"

# Public display URL
mapj confluence export "https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus"

# Cloud URL
mapj confluence export "https://company.atlassian.net/wiki/spaces/TEAM/pages/12345/Title"
```

---

## Output Structure (with `--output-path`)

```
output-path/
├── spaces/
│   └── SPACE_KEY/
│       ├── README.md           ← index of all pages in the space
│       ├── pages/
│       │   └── ID-slug-title.md  ← one per page, with YAML front matter
│       └── attachments/
│           └── PAGE_ID/        ← only present with --with-attachments
├── manifest.jsonl              ← one JSON line per exported page
└── export-errors.jsonl         ← one JSON line per failure (with retry_cmd)
```

Each page file has YAML front matter:
```yaml
---
page_id: "22479548"
title: "SDK Microsiga Protheus"
source_url: "https://..."
space_key: "framework"
labels: ["sdk", "protheus"]
updated_at: "2025-08-05T09:30:23Z"
exported_at: "2026-03-28T22:00:00Z"
---
```

---

## Debugging Failed Exports

```bash
# Verbose — shows per-page progress
mapj confluence export <url> --output-path ./docs --verbose

# Debug — saves raw HTML to .debug/
mapj confluence export <url> --output-path ./docs --debug

# Full diagnostic dump
mapj confluence export <url> --output-path ./docs --dump-debug

# Retry only failed pages
mapj confluence retry-failed --output-path ./docs
```

---

## What This Skill Will NOT Do

- ❌ **Create or edit** Confluence pages — read-only export only
- ❌ **Search** Confluence content → use `mapj-tdn-search` skill instead
- ❌ **Export non-Confluence** sources
- ❌ **Preserve Confluence macros exactly** — complex macros become text/simplified Markdown

---

## Error Reference

| Error | Condition | Fix |
|---|---|---|
| `401 Unauthorized` | Wrong auth type (username on Server/DC) | Re-login without `--username` |
| `NOT_AUTHENTICATED` | No credentials stored | `mapj auth login confluence ...` |
| `PAGE_NOT_FOUND` | Wrong URL or private page | Try with `?pageId=N` format, check access |
| `HTTP_TIMEOUT` | Slow server or large page | Retry; use `retry-failed` for bulk |
| `SPACE_NOT_FOUND` | Wrong space key | Check key in Confluence URL |

---

## Extended Reference

| Need | File |
|---|---|
| All supported URL formats with examples | `references/url-formats.md` |
| All flags with defaults | `references/flags.md` |
| Auth setup in detail (cloud vs server) | `references/auth.md` |
| Output file structure and manifest schema | `references/output-structure.md` |
