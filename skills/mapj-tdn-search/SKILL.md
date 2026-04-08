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
    - auto-pagination
  related:
    - mapj-confluence-export
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj tdn — Agent Skill v2.1

Search documentation in TOTVS Developer Network (TDN). **No authentication required** for
public content on `tdn.totvs.com`. Uses `siteSearch` CQL — searches title, body, and labels.
Features **Native Auto-Pagination** to reach your desired result count in one go.

---

## Role

Eres un agente especializado en buscar documentación TOTVS Developer Network (TDN). Tu responsabilidad
es encontrar páginas relevantes usando CQL. Operas sin autenticación para contenido público en
`tdn.totvs.com`. Siempre verificas que los resultados tengan URLs válidas.

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

# Filter by label (AND logic for multiple)
mapj tdn search --space PROT --label versao_12 --label ponto_de_entrada

# Filter by date (last N weeks/months/days/years)
mapj tdn search "api rest" --space PROT --since 1m
mapj tdn search "advpl" --space PROT --since 2024-01-01

# Check if results have children
mapj tdn search "AdvPL" --space PROT --max-results 10 --check-children

# Get all pages under an ancestor ID
mapj tdn search --ancestor 811253174 --type page

# Auto-Pagination (fetches multiple pages internally)
mapj tdn search "advpl" --space PROT --max-results 100

# Search AND export all found pages
mapj tdn search "ponto de entrada" --space PROT --export-to ./docs/pontos

# List all available spaces
mapj tdn spaces list
```

---

## Output Schema (Optimized for Tokens)

```json
{
  "ok": true,
  "result": {
    "results": [
      {
        "id": "663845988",
        "title": "PE MT261FIL - Filtro com regra AdvPL",
        "url": "https://tdn.totvs.com/pages/viewpage.action?pageId=663845988",
        "childCount": 1        // present only with --check-children. 0=leaf, N=has children
      }
    ],
    "count": 10,
    "start": 0,
    "hasNext": true,
    "nextStart": 10,
    "cql": "siteSearch ~ \"advpl\" AND ..."
  }
}
```

---

## Examples

### Example 1: Basic keyword search
**Input:** Usuario pregunta "busca documentación sobre AdvPL"
**Command:** `mapj tdn search "AdvPL"`
**Output:**
```json
{
  "ok": true,
  "result": {
    "results": [
      {"id": "224440806", "title": "AdvPL - Sobre", "url": "https://tdn.totvs.com/..."},
      {"id": "23888829", "title": "Funções AdvPL", "url": "https://tdn.totvs.com/..."}
    ],
    "count": 10,
    "hasNext": true
  }
}
```

### Example 2: Space-filtered search
**Input:** Usuario pregunta "puntos de entrada en Protheus"
**Command:** `mapj tdn search "ponto de entrada" --space PROT`
**Output:** JSON con resultados filtrados al espacio PROT (documentación de Protheus)

### Example 3: Recent content with label filter
**Input:** Usuario pregunta "documentación nueva de la versión 12"
**Command:** `mapj tdn search --space PROT --label versao_12 --since 1m`
**Output:** JSON con páginas actualizadas en el último mes con label versao_12

### Example 4: Search and export pipeline
**Input:** Usuario pregunta "busca y exporta documentación de puntos de entrada"
**Command:** `mapj tdn search "ponto de entrada" --space PROT --max-results 25 --export-to ./docs`
**Output:**
```json
{
  "ok": true,
  "searched": 25,
  "exported": 24,
  "failed": 1,
  "outputDir": "./docs"
}
```

---

## Deciding: Single Page vs Full Tree (`--with-descendants`)

Use `--check-children` to see if a page has children before exporting:

```bash
mapj tdn search "AdvPL" --space PROT --max-results 5 --check-children
```

Cada resultado incluye `childCount`:

| `childCount` | Significado | ¿Usar `--with-descendants`? |
|---|---|---|
| `0` | Página hoja — sin hijos | No, ambas opciones exportan igual |
| `1..N` | N hijos **directos** | Probablemente sí |
| `-1` | Error al consultar | CLI auto-retries; if persists, check VPN |

---

## Search → Export Pipeline

```bash
# 1. Preview con check-children para entender el scope
mapj tdn search "ponto de entrada" --space PROT --max-results 25 --check-children

# 2. Export all found pages (solo la página raíz de cada resultado)
mapj tdn search "ponto de entrada" --space PROT --max-results 25 --export-to ./docs

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

## Success Criteria

- [ ] Output es JSON válido con `ok: true`
- [ ] Exit code es 0
- [ ] Results contiene URLs válidas de TDN
- [ ] CQL fue ejecutado correctamente
- [ ] `count > 0` si hay resultados; `count: 0` si no hay coincidencias

---

## Error Reference

| Code | Condition | Fix |
|---|---|---|
| `USAGE_ERROR` | No query filters provided | Provide at least one filter |
| `USAGE_ERROR` | `--max-results` <= 0 | Use a positive number |
| `SEARCH_ERROR` | Server error | CLI auto-retries; check network if fails |
| `AUTH_ERROR` | Credential error | Skip auth for public TDN content |

---

## Extended Reference

| Need | File |
|---|---|
| CQL syntax reference (operators, fields, functions) | `references/cql-reference.md` |
| All known spaces with descriptions | `references/spaces.md` |
