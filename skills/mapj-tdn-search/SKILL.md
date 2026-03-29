---
name: mapj-tdn-search
description: >
  Search TOTVS Developer Network (TDN) documentation using CQL. Export found pages directly.
  Use when: searching TDN documentation, finding Protheus API references, searching ADVPL guides,
  finding ERP customization docs, exporting a set of pages by topic (search→export pipeline),
  browsing all pages under a section (--ancestor), filtering by label/tag, listing available spaces.
  Do NOT use when: exporting a specific known page (use mapj-confluence-export), querying Protheus
  database (use mapj-protheus-query), or creating/modifying Confluence pages.
  Triggers: "search TDN", "find documentation", "look up TOTVS", "TDN search", "search AdvPL",
  "find API docs", "ADVPL documentation", "search framework", "list TDN spaces",
  "export by topic", "find pages for", "search and export", "what pages exist for".
compatibility: No auth required for tdn.totvs.com public content. Optional PAT for private content.
metadata:
  version: 2.1.0
  language: en
  author: Mario Pereira
  tags:
    - tdn
    - totvs
    - confluence
    - documentation
    - search
    - cql
  capabilities:
    - search
    - filter-by-space
    - filter-by-label
    - filter-by-date
    - filter-by-ancestor
    - search-to-export-pipeline
    - list-spaces
  related:
    - mapj-confluence-export
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj tdn — Agent Skill v2.0

Search documentation in TOTVS Developer Network (TDN). **No authentication required** for
public content on `tdn.totvs.com`. Uses `siteSearch` CQL — searches title, body, and labels.

---

## Quick Routing

```
Need to...
├─ Find docs by keyword → tdn search "keyword"
├─ Find docs in a space → tdn search "keyword" --space PROT
├─ Get all pages under a section → tdn search --ancestor 187531295
├─ Filter by label/tag → tdn search --label versao_12 --space PROT
├─ Find what changed recently → tdn search "keyword" --since 1m
├─ Know if a page has children (before deciding --with-descendants)
│   → tdn search "keyword" --space PROT --check-children
├─ Search AND export at once → tdn search "keyword" --space PROT --export-to ./docs
└─ See all available spaces → tdn spaces list
```

---

## Core Commands

```bash
# Basic search (no auth needed)
mapj tdn search "AdvPL"

# Filter by space
mapj tdn search "ponto de entrada" --space PROT

# Filter by content type
mapj tdn search "apostila treinamento" --space PROT --type attachment

# Filter by label (AND logic for multiple)
mapj tdn search --space PROT --label versao_12 --label ponto_de_entrada

# Filter by date (last N weeks/months/days/years)
mapj tdn search "api rest" --space PROT --since 1m
mapj tdn search "advpl" --space PROT --since 2024-01-01

# Check if results have children (before deciding --with-descendants)
mapj tdn search "AdvPL" --space PROT --limit 10 --check-children

# Get all pages under an ancestor ID
mapj tdn search --ancestor 811253174 --type page

# Multi-space search
mapj tdn search "API REST" --spaces PROT,LDT,TASS

# Search AND export all found pages
mapj tdn search "ponto de entrada" --space PROT --export-to ./docs/pontos

# List all available spaces
mapj tdn spaces list

# Pagination
mapj tdn search "advpl" --space PROT --limit 25 --start 25  # page 2
```

---

## Output Schema

```json
{
  "ok": true,
  "result": {
    "results": [
      {
        "id": "663845988",
        "type": "page",
        "title": "PE MT261FIL - Filtro com regra AdvPL",
        "url": "https://tdn.totvs.com/pages/viewpage.action?pageId=663845988",
        "space": { "key": "PROT", "name": "Linha Microsiga Protheus" },
        "labels": ["versao_12", "ponto_de_entrada", "mata261"],
        "ancestors": [
          { "id": "187531295", "title": "TOTVS Linha Protheus" },
          { "id": "811253174", "title": "Pontos de Entrada - Protheus 12" }
        ],
        "version": 9,
        "lastUpdated": "2025-12-16T17:44:59-03:00",
        "lastUpdatedBy": "Vitor Badam Rafael",
        "childCount": 1        // present only with --check-children. 0=leaf, N=has children
      }
    ],
    "count": 10,
    "start": 0,
    "hasNext": true,
    "nextStart": 10,
    "cql": "siteSearch ~ \"advpl\" AND space = \"PROT\" AND type = page"
  }
}
```

---

## Deciding: Single Page vs Full Tree (`--with-descendants`)

Use `--check-children` to see if a page has children before exporting:

```bash
mapj tdn search "AdvPL" --space PROT --limit 5 --check-children
```

Cada resultado incluye `childCount`:

| `childCount` | Significado | ¿Usar `--with-descendants`? |
|---|---|---|
| `0` | Página hoja — sin hijos | No, ambas opciones exportan igual |
| `1..N` | N hijos **directos** | Probablemente sí |
| `-1` | Error al consultar | Verificar manualmente |

> ⚠️ **TRAMPA CRÍTICA — `childCount` ≠ total de páginas del árbol**
> `childCount` cuenta solo los hijos **directos** (un nivel).
> Una página con `childCount: 1` puede tener **171 páginas** en total
> si ese hijo tiene sus propios descendientes.
> El único número real es el que ves al ejecutar `--with-descendants`.

**Workflow correcto:**
```bash
# 1. Verificar si tiene hijos
mapj tdn search "AdvPL" --space PROT --check-children
# → childCount: 1  ← tiene al menos un hijo directo

# 2. Exportar sin descendencia (preview rápido)
mapj confluence export 235312129 --output-path ./preview
# → 1 página exportada

# 3. Exportar árbol completo
mapj confluence export 235312129 --output-path ./docs --with-descendants
# → 171 páginas exportadas  (sorpresa: 1 hijo directo → 171 total)
```

---

## Search → Export Pipeline

Use when you need to find AND export a group of pages (sin --with-descendants por página):

```bash
# 1. Preview con check-children para entender el scope
mapj tdn search "ponto de entrada" --space PROT --limit 25 --check-children

# 2. Export all found pages (solo la página raíz de cada resultado)
mapj tdn search "ponto de entrada" --space PROT --limit 25 --export-to ./docs

# Pipeline output
{
  "searched": 25,
  "exported": 24,
  "failed": 1,
  "outputDir": "C:/abs/path/to/docs",
  "pages": ["PE MT261FIL (663845988)", ...],
  "errors": ["page 'X' (ID): reason"]
}
```

---

## Key Spaces

| Key | Name |
|---|---|
| `PROT` | Linha Microsiga Protheus ⭐ |
| `LDT` / `DL` | Linha Datasul |
| `TASS` | TOTVS API Services |
| `INT` | Integrações |
| `FDI` | Aceleradores + IA |

For complete list → `mapj tdn spaces list`

## Useful Labels in PROT Space

| Label | Content |
|---|---|
| `versao_12` | Protheus 12 documentation |
| `ponto_de_entrada` | Pontos de entrada (user exits) |
| `documento_de_referencia` | Reference documentation |
| `base_de_conhecimento` | Knowledge base articles |
| `api` | API documentation |

---

## Error Reference

| Code | Condition | Fix |
|---|---|---|
| `USAGE_ERROR` | No query, --ancestor, or --label provided | Provide at least one filter |
| `USAGE_ERROR` | `--limit` out of range | Use 1–100 |
| `SEARCH_ERROR` (retryable) | Server error | Wait 1s, retry |
| `AUTH_ERROR` | Could not load stored credentials | Run `mapj auth login tdn --url URL --token TOKEN` or skip (public content works without auth) |

---

## What This Skill Will NOT Do

- ❌ **Export a specific known page by URL/ID** → use `mapj-confluence-export/SKILL.md`
- ❌ **Query Protheus database** → use `mapj-protheus-query/SKILL.md`
- ❌ **Create or edit pages** — read-only
- ❌ **Search non-TDN Confluence** — configure base URL via `mapj auth login tdn --url`

---

## Extended Reference

| Need | File |
|---|---|
| CQL syntax reference (operators, fields, functions) | `references/cql-reference.md` |
| All known spaces with descriptions | `references/spaces.md` |
