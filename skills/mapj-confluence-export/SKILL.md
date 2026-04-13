---
name: mapj-confluence-export
description: >
  Export Confluence pages to Markdown files with YAML front matter, metadata, and structure.
  Use when: exporting a single Confluence page, exporting a page tree recursively (with descendants),
  exporting an entire Confluence space, or downloading page attachments.
  Do NOT use when: searching Confluence content (use mapj-tdn-search), creating or modifying
  Confluence pages, or querying Protheus database.
  Triggers: "export confluence", "export to markdown", "download confluence page",
  "export with children", "export space", "with descendants", "confluence to markdown",
  "export attachments", "bulk export confluence".
compatibility: Requires mapj CLI at PATH. Bearer PAT for Server/DC, Basic Auth for atlassian.net.
metadata:
  version: 2.1.0
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
    - url-resolution
    - auto-healing
    - concurrent-export
  related:
    - mapj-tdn-search
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj confluence export — Agent Skill v2.1

Export Confluence pages to Markdown files with YAML front matter. Supports single pages,
recursive page trees, full spaces, and binary attachments. Features **native Auto-Healing**
(exponential backoff for 429/50x) and **Concurrent Worker Pool** for high-speed exports.

> ⚠️ **Auth matters**: Using `--username` on Server/DC (non-atlassian.net) causes 401.
> Bear PAT is the correct auth for `tdninterno.totvs.com` and similar instances.

---

## Role

Eres un agente especializado en exportar páginas Confluence a Markdown. Tu responsabilidad es
convertir documentación estructurada a archivos con YAML front matter. Operas de manera read-only,
preservando la estructura original y descargando attachments cuando aplica.

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
└─ Entire Confluence space (Concurrent)
    mapj confluence export-space <SPACE_KEY> --output-path ./docs
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
└── manifest.jsonl              ← one JSON line per exported page
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

## Examples

### Example 1: Single page export to stdout
**Input:** Usuario pregunta "exporta la página 22479548"
**Command:** `mapj confluence export 22479548`
**Output:** Markdown content printed to stdout with YAML front matter

### Example 2: Single page to directory
**Input:** Usuario pregunta "exporta la página SDK Protheus a ./docs"
**Command:** `mapj confluence export "https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus" --output-path ./docs`
**Output:**
```json
{
  "ok": true,
  "result": {
    "pagesExported": 1,
    "outputPath": "./docs/spaces/framework/pages/22479548-sdk-microsiga-protheus.md"
  }
}
```

### Example 3: Recursive export with attachments
**Input:** Usuario pregunta "exporta la página raíz con todos los hijos y adjuntos"
**Command:** `mapj confluence export 22479548 --output-path ./docs --with-descendants --with-attachments`
**Output:**
```json
{
  "ok": true,
  "result": {
    "pagesExported": 15,
    "attachmentsDownloaded": 23,
    "outputPath": "./docs"
  }
}
```

### Example 4: Export entire space
**Input:** Usuario pregunta "exporta todo el espacio framework"
**Command:** `mapj confluence export-space framework --output-path ./docs`
**Output:** JSON con `pagesExported: N` y `outputPath`

---

## Debugging and Progress

```bash
# Verbose — shows per-page progress and successful exports
mapj confluence export <url> --output-path ./docs --verbose
```

**Auto-Healing:** The CLI automatically retries failed requests (429 Rate Limit, 50x Server Error) using exponential backoff. You don't need to manually retry unless it fails permanently.

---

## What This Skill Will NOT Do

- ❌ **Create or edit** Confluence pages — read-only export only
- ❌ **Search** Confluence content → use `mapj-tdn-search` skill instead
- ❌ **Export non-Confluence** sources
- ❌ **Preserve Confluence macros exactly** — complex macros become text/simplified Markdown

---

## Success Criteria

- [ ] Output es JSON válido con `ok: true`
- [ ] Exit code es 0
- [ ] Archivo .md creado con YAML front matter
- [ ] Attachments descargados si se usó `--with-attachments`
- [ ] No hay errores de autenticación (401)
- [ ] Estructura de directorios preservada con `--with-descendants`

---

## Health and Observability

```bash
# Check Confluence service health
mapj health --service=confluence

# View observability metrics (Prometheus format)
mapj observability metrics
```

Use `mapj health --service=confluence` to verify Confluence connectivity before exporting.
Use `mapj observability metrics` to monitor export operation traces and throughput.

---

## Error Reference

| Error | Condition | Fix |
|---|---|---|
| `401 Unauthorized` | Wrong auth type (username on Server/DC) | Re-login without `--username` |
| `NOT_AUTHENTICATED` | No credentials stored | `mapj auth login confluence ...` |
| `PAGE_NOT_FOUND` | Wrong URL or private page | Try with `?pageId=N` format, check access |
| `HTTP_TIMEOUT` | Slow server or network issue | CLI auto-retries 3x; if fails, check VPN |
| `SPACE_NOT_FOUND` | Wrong space key | Check key in Confluence URL |

---

## Extended Reference

| Need | File |
|---|---|
| All supported URL formats with examples | `references/url-formats.md` |
| All flags with defaults | `references/flags.md` |
| Auth setup in detail (cloud vs server) | `references/auth.md` |
| Output file structure and manifest schema | `references/output-structure.md` |
