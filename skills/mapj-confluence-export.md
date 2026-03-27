---
name: mapj-confluence-export
description: >
  Export Confluence page content to markdown, HTML, or JSON format.
  Use when: exporting documentation to markdown, converting Confluence pages to HTML, getting page metadata as JSON,
  downloading technical docs for offline use, or converting Confluence content to other formats.
  Triggers: "export confluence", "export to markdown", "confluence to HTML", "download confluence page",
  "convert page to JSON", "export page content", "confluence page export".
metadata:
  version: 1.0.0
  language: en
  author: Mario Pereira
  license: MIT
  repository: https://github.com/Mario-pereyra/mapj
  tags:
    - confluence
    - export
    - markdown
    - html
    - json
    - documentation
    - conversion
  capabilities:
    - export
    - fetch-page
  related:
    - mapj-tdn-search
    - mapj-protheus-query
allowed-tools: Bash
---

# mapj confluence export

Export a Confluence page to various formats (markdown, HTML, JSON).

## Usage

```
mapj confluence export <url-or-page-id> [--format markdown|html|json] [--output json|table]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<url-or-page-id>` | Yes | Confluence page URL, relative path, or numeric page ID |

## Flags

| Flag | Short | Required | Default | Description |
|------|-------|----------|---------|-------------|
| `--format` | `-f` | No | markdown | Output format: `markdown`, `html`, or `json` |
| `--include-comments` | No | false | Include page comments in export |
| `--output` | `-o` | No | json | CLI output format: `json` or `table` |

## Examples

### Export by Page ID

```bash
mapj confluence export 573675873
```

### Export by Full URL

```bash
mapj confluence export https://tdninterno.totvs.com/wiki/spaces/LDT/pages/573675873
```

### Export as HTML

```bash
mapj confluence export 573675873 --format html
```

### Export as JSON with Full Metadata

```bash
mapj confluence export 573675873 --format json
```

### Export with Comments

```bash
mapj confluence export 573675873 --include-comments
```

## Prerequisites

1. **Authenticate first**:
   ```bash
   # For TDN (Confluence Cloud)
   mapj auth login confluence \
     --url https://tdninterno.totvs.com \
     --username your-email@totvs.com \
     --token YOUR_API_TOKEN
   ```

2. **Verify auth**:
   ```bash
   mapj auth status
   ```

## URL Formats Supported

| Format | Example |
|--------|---------|
| Full URL | `https://tdninterno.totvs.com/wiki/spaces/LDT/pages/573675873` |
| Relative path | `/wiki/spaces/LDT/pages/573675873` |
| Page ID only | `573675873` |

## Output Schema

### Success Response (markdown/html)

```json
{
  "ok": true,
  "command": "mapj confluence export 573675873 --format markdown",
  "result": {
    "pageId": "573675873",
    "title": "MT0795 - Geração de Históricos",
    "format": "markdown",
    "content": "<p>Page content in selected format...</p>",
    "url": "https://tdninterno.totvs.com/pages/viewpage.action?pageId=573675873"
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### Success Response (json)

```json
{
  "ok": true,
  "command": "mapj confluence export 573675873 --format json",
  "result": {
    "pageId": "573675873",
    "title": "MT0795 - Geração de Históricos",
    "format": "json",
    "content": {
      "id": "573675873",
      "type": "page",
      "title": "MT0795 - Geração de Históricos",
      "body": {
        "storage": {
          "value": "<p>HTML content...</p>",
          "representation": "storage"
        }
      },
      "space": {
        "key": "LDT",
        "name": "Linha Datasul"
      },
      "version": {
        "number": 4,
        "when": "2020-11-25T11:00:02.877-03:00"
      },
      "_links": {
        "webui": "/pages/viewpage.action?pageId=573675873"
      }
    },
    "url": "https://tdninterno.totvs.com/pages/viewpage.action?pageId=573675873"
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### Error Response

```json
{
  "ok": false,
  "command": "mapj confluence export 999999999",
  "error": {
    "code": "EXPORT_ERROR",
    "message": "page not found or access denied",
    "retryable": false
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

## Supported Formats

| Format | Description | Use Case |
|--------|-------------|----------|
| `markdown` | Best for documentation | Reading, converting to other docs |
| `html` | Raw Confluence storage HTML | Embedding in web pages |
| `json` | Full page metadata | Parsing, automation, integration |

## Error Cases

| Code | Condition | Resolution |
|------|-----------|------------|
| AUTH_ERROR (3) | Not logged in | Run `mapj auth login confluence ...` |
| INVALID_URL (2) | Malformed URL or page ID | Check URL format |
| EXPORT_ERROR (1) | Page not found or server error | Verify page ID exists, retry |
| RATE_LIMITED (4) | Too many requests | Wait 1s and retry |

## Limitations

- **Large pages**: Pages with many macros may timeout (>30s)
- **Attachments**: Binary attachments not exported (only page content)
- **Page history**: Only current version exported
- **Permissions**: User must have view permission on the page

## Workflow Example

```bash
#!/bin/bash
# Search for documentation and export the best match

# 1. Search for the topic
search_result=$(mapj tdn search "MT0795 Geração de Históricos")
page_id=$(echo "$search_result" | jq -r '.result.results[0].id')

# 2. Export the page
mapj confluence export "$page_id" --format markdown > documentation.md

# 3. Use the content
cat documentation.md
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Export successful |
| 1 | Export failed (server error) |
| 2 | Usage error (invalid URL/ID) |
| 3 | Auth error (not authenticated) |
| 4 | Retry recommended (rate limited) |

## See Also

- [SKILL.md](../SKILL.md) - Main manifest
- [mapj-tdn-search](mapj-tdn-search.md) - Find pages to export
- [mapj-protheus-query](mapj-protheus-query.md) - Query Protheus database
