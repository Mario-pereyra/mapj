# mapj CLI Architecture

## Overview

mapj is a CLI tool for the TOTVS ecosystem that provides:
1. TDN (TOTVS Developer Network) documentation search
2. Confluence page export to Markdown
3. Protheus ERP SQL Server query execution
4. Query presets with parameter interpolation

## System Architecture

```
mapj (CLI entry point)
├── cli/           # Command definitions (clap)
├── auth/          # Credential encryption/storage
├── output/        # Response formatters
├── tdn/           # TDN/Confluence API client
├── confluence/    # HTML-to-Markdown conversion
├── protheus/     # SQL Server client
└── preset/        # Query preset storage
```

## Data Flow

### Authentication Flow
1. User runs `mapj auth login <service> --token X`
2. Credentials encrypted with AES-256-GCM
3. Stored in `~/.config/mapj/credentials.enc`
4. On subsequent commands, credentials decrypted and used

### TDN Search Flow
1. User runs `mapj tdn search "query"`
2. CLI builds CQL from flags
3. HTTP request to Confluence API
4. Results formatted (LLM/TOON/Auto) and returned

### Confluence Export Flow
1. User runs `mapj confluence export <url-or-id>`
2. URL parsed to extract page ID
3. Page fetched via Confluence REST API
4. HTML converted to Markdown with YAML front matter
5. Written to disk or stdout

### Protheus Query Flow
1. User runs `mapj protheus query "SELECT ..."`
2. SQL validated (prefix, forbidden keywords, injection)
3. Connection established via SQL Server driver
4. Query executed with --max-rows limit
5. Results formatted and returned

### Preset System Flow
1. User creates preset with `preset add --query "SELECT :param ..."`
2. Parameters detected from `:placeholder` syntax
3. Preset stored in `~/.config/mapj/presets.json`
4. On `preset run`, parameters interpolated with escaping
5. Final query executed against Protheus

## Output Formats

### Envelope Structure
All commands return a JSON envelope:
```json
{
  "ok": true,
  "command": "mapj tdn search",
  "result": {...}
}
```

Error:
```json
{
  "ok": false,
  "command": "mapj tdn search",
  "error": {
    "code": "SEARCH_ERROR",
    "message": "...",
    "hint": "...",
    "retryable": false
  }
}
```

### TOON Format (Tabular Object Notation)
Designed for ~40% token savings vs JSON:

**Uniform object arrays** (same keys in all elements):
```
result[3]{id,name,status}:
001,John,active
002,Jane,inactive
003,Bob,active
```

**Primitive arrays**:
```
result[3]: 1,2,3
```

**Objects**:
```
result:
  id: "001"
  name: "John"
  status: "active"
```

**Strings with special chars** are quoted and escaped.

## Security Model

### SQL Injection Prevention
1. Query must start with SELECT/WITH/EXEC only
2. Forbidden keywords detected anywhere in query
3. Semicolons (multiple statements) rejected
4. Parameter values checked for injection patterns:
   - Semicolons followed by dangerous keywords
   - OR with always-true conditions
   - UNION SELECT patterns
   - Comment injection

### Credential Storage
- AES-256-GCM encryption with 12-byte nonce
- Key derived from machine (hostname + username) or MAPJ_ENCRYPTION_KEY env var
- Never stored in plaintext

## File Locations

| Purpose | Path |
|---------|------|
| Credentials (encrypted) | `~/.config/mapj/credentials.enc` |
| Presets | `~/.config/mapj/presets.json` |
| Config | `~/.config/mapj/config.toml` (optional) |

## Concurrency

- Confluence exports use 10 concurrent workers
- HTTP requests use connection pooling
- SQL Server connections are pooled

## Error Handling

Exit codes:
- 0: Success
- 1: General error
- 2: Usage error (invalid args, SQL validation)
- 3: Auth error
- 4: Retryable (network timeout, rate limit)

All errors return structured JSON envelope with actionable hints.
