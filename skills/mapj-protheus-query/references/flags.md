# Protheus Query — Flags Reference

Complete reference for all `mapj protheus query` and `mapj protheus connection` flags.

---

## `mapj protheus query "<SQL>"` flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--format` | string | `json` | Result format: `json` or `csv` (RFC 4180) |
| `--max-rows` | int | `10000` | Client-side row cap. `0` = no limit |
| `--connection` | string | *(active)* | Run against this named profile **without** changing the active connection |
| `--output-file` | string | *(stdout)* | Write result to file; stdout receives only a summary |

### Global output flag (all commands)

| Flag | Default | Description |
|---|---|---|
| `-o` / `--output` | `llm` | `llm` = compact JSON (default), `json` = indented + metadata |

---

## `--connection` — Cross-environment queries

Run a query against any registered profile without touching the active connection.

```bash
# Active is TOTALPEC_BIB. Query PRD without switching:
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection TOTALPEC_PRD

# Compare BIB vs PRD vs UNION side by side:
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"                             # BIB (active)
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection TOTALPEC_PRD  # PRD
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection UNION_BIB     # UNION

# After all three: active connection is still TOTALPEC_BIB — unchanged
```

**Key behavior:**
- Does NOT modify `ProtheusActive` in stored credentials
- If `--connection NAME` is not found → `PROFILE_NOT_FOUND` error with list hint
- Requires the VPN for the target server to be active

---

## `--output-file` — Large result sets

Write the full result to disk instead of stdout. Stdout only gets a summary.

```bash
mapj protheus query "SELECT * FROM SA1010" --output-file ./sa1010.json
# stdout → {"ok":true,"result":{"rows":1500,"columns":45,"format":"json","output_file":"./sa1010.json"}}

mapj protheus query "SELECT * FROM SA1010" --format csv --output-file ./sa1010.csv
# stdout → {"ok":true,"result":{"rows":1500,"columns":45,"format":"csv","output_file":"./sa1010.csv"}}
```

**When to use:**
- Result set > ~200 rows (avoids saturating LLM context window)
- Generating reports to be consumed by downstream tools
- Exporting to spreadsheets (`--format csv --output-file`)

**Error if directory doesn't exist:**
```json
{"ok":false,"error":{"code":"FILE_WRITE_ERROR","message":"...","hint":"Check that the directory exists and you have write access: ./path/file.json"}}
```

---

## `--format` — Output format

| Value | Description | When to use |
|---|---|---|
| `json` (default) | Structured `{columns:[], rows:[[]], count:N}` | LLM parsing, downstream processing |
| `csv` | RFC 4180 compliant CSV with header row | Excel/spreadsheet export |

**CSV escaping:** Fields containing commas, double-quotes, or newlines are properly quoted per RFC 4180.

---

## `--max-rows` — Row cap

Client-side limit applied **after** the query executes. Prefer using `TOP N` in SQL (server-side, more efficient).

```bash
# Preferred: server-side limit
mapj protheus query "SELECT TOP 100 * FROM SA1010"

# Client-side cap (download all, then truncate)
mapj protheus query "SELECT * FROM SA1010" --max-rows 100

# Pagination: SQL OFFSET (server-side)
mapj protheus query "SELECT * FROM SA1010 ORDER BY A1_COD OFFSET 100 ROWS FETCH NEXT 100 ROWS ONLY"
```

---

## `mapj protheus connection` subcommands

| Subcommand | Description |
|---|---|
| `connection add <name> --server ... --database ... --user ... --password ...` | Register new profile |
| `connection list` | List all profiles (JSON, `active` field marks current) |
| `connection use <name>` | Switch active profile |
| `connection ping [name]` | Test connectivity (latencyMs in result, VPN hint on failure) |
| `connection show [name]` | Show profile details (password masked) |
| `connection remove <name>` | Delete profile |

### `connection add` flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `--server` | ✅ | — | SQL Server host or IP |
| `--port` | — | `1433` | SQL Server port |
| `--database` | ✅ | — | Database name |
| `--user` | ✅ | — | Database user |
| `--password` | ✅ | — | Database password |
| `--use` | — | `false` | Set as active immediately after registering |

> First profile added is automatically set as active, regardless of `--use`.
