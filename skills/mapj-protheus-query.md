---
name: mapj-protheus-query
description: "Execute SELECT queries on Protheus SQL Server database"
metadata:
  version: 1.0.0
---

# mapj protheus query

Execute SELECT queries on Protheus ERP SQL Server database.

## Usage

```
mapj protheus query <sql> [--format json|csv] [--output json|table]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<sql>` | Yes | SQL SELECT query |

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--format` | No | json | Output format (json, csv) |
| `--output` | No | json | CLI output format (json, table) |

## Examples

```bash
# Basic SELECT query
mapj protheus query "SELECT TOP 10 * FROM SPED050"

# Count query
mapj protheus query "SELECT COUNT(*) FROM SA1010"

# Export as CSV
mapj protheus query "SELECT * FROM SPED050 WHERE CAMPO = 'value'" --format csv

# Complex query with JOIN
mapj protheus query "SELECT a.CAMPO1, b.DESCRI FROM SA1010 a INNER JOIN SB1010 b ON a.CAMPO = b.CAMPO"
```

## Output Schema

```json
{
  "ok": true,
  "command": "mapj protheus query \"SELECT TOP 10 * FROM SPED050\"",
  "result": {
    "columns": ["CAMPO1", "CAMPO2", "CAMPO3"],
    "rows": [
      ["value1", "value2", "value3"],
      ["value4", "value5", "value6"]
    ],
    "count": 2
  }
}
```

## CSV Output

```json
{
  "ok": true,
  "command": "mapj protheus query \"SELECT * FROM TABLE\" --format csv",
  "result": {
    "format": "csv",
    "content": "CAMPO1,CAMPO2,CAMPO3\nvalue1,value2,value3\nvalue4,value5,value6"
  }
}
```

## Security: SELECT-Only

**CRITICAL**: Only SELECT queries are allowed. The following are blocked:

- INSERT, UPDATE, DELETE
- DROP, ALTER, CREATE, TRUNCATE
- EXEC, EXECUTE, MERGE
- GRANT, REVOKE, DENY
- BACKUP, RESTORE

Attempting to execute non-SELECT queries returns:

```json
{
  "ok": false,
  "command": "mapj protheus query \"INSERT INTO...\"",
  "error": {
    "code": "USAGE_ERROR",
    "message": "validation error: query contains forbidden keyword: INSERT"
  }
}
```

Exit code: 2 (Usage Error)

## Prerequisites

1. Authenticate first:
```bash
mapj auth login protheus --server 192.168.1.100 --port 1433 --database PROTHEUS --user admin --password secret
```

## Common Protheus Tables

| Table | Description |
|-------|-------------|
| SPED050 | SPED Fiscal |
| SA1010 | Clientes (Customers) |
| SB1010 | Productos (Products) |
| SC7010 | Pedidos (Orders) |
| SF2010 | Notas Fiscais |

Note: Table names in Protheus typically have a 6-character prefix followed by a version suffix (e.g., SA1010).

## Error Cases

- **NOT_AUTHENTICATED**: Run `mapj auth login protheus` first
- **USAGE_ERROR**: Invalid SQL (non-SELECT, syntax error)
- **QUERY_ERROR**: Database error - check connection and query
