---
name: mapj-confluence-export
description: "Export Confluence page content to markdown, HTML, or JSON"
metadata:
  version: 1.0.0
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
| `<url-or-page-id>` | Yes | Confluence page URL or numeric page ID |

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--format` | No | markdown | Output format (markdown, html, json) |
| `--include-comments` | No | false | Include page comments |
| `--output` | No | json | CLI output format (json, table) |

## Examples

```bash
# Export by page ID
mapj confluence export 12345

# Export by full URL
mapj confluence export https://company.atlassian.net/wiki/spaces/TEAM/pages/12345/Title

# Export as HTML
mapj confluence export 12345 --format html

# Export as JSON with full metadata
mapj confluence export 12345 --format json
```

## Output Schema

```json
{
  "ok": true,
  "command": "mapj confluence export 12345 --format markdown",
  "result": {
    "pageId": "12345",
    "title": "Page Title",
    "format": "markdown",
    "content": "# Page Title\n\nContent here...",
    "url": "https://company.atlassian.net/wiki/spaces/TEAM/pages/12345/Title"
  }
}
```

## Supported Formats

| Format | Description |
|--------|-------------|
| `markdown` | Markdown export (best for documentation) |
| `html` | Raw HTML storage format |
| `json` | Full page metadata as JSON |

## Prerequisites

1. Authenticate first:
```bash
mapj auth login confluence --url https://your-company.atlassian.net --token YOUR_API_TOKEN
```

## URL Formats Supported

- Full URL: `https://company.atlassian.net/wiki/spaces/SPACE/pages/12345/Title`
- Relative: `/spaces/SPACE/pages/12345`
- Page ID only: `12345`

## Error Cases

- **NOT_AUTHENTICATED**: Run `mapj auth login confluence` first
- **INVALID_URL**: Could not parse the provided URL
- **EXPORT_ERROR**: Server error - retry recommended
