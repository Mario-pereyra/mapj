---
name: mapj-protheus-query
description: >
  Execute SELECT queries on Protheus ERP SQL Server database for reporting and data retrieval.
  Use when: querying Protheus database tables, generating reports, extracting data from ERP,
  finding customer records, product information, invoice data, or any SQL SELECT on Protheus.
  Triggers: "query Protheus", "SELECT from Protheus", "Protheus database", "ERP query",
  "extract Protheus data", "SQL query Protheus", "Protheus tables", "report from Protheus".
metadata:
  version: 1.0.0
  language: en
  author: Mario Pereira
  license: MIT
  repository: https://github.com/Mario-pereyra/mapj
  tags:
    - protheus
    - totvs
    - erp
    - database
    - sql
    - query
    - select
    - reporting
  capabilities:
    - query
    - select-only
  security:
    - select-only-enforced
    - no-insert
    - no-update
    - no-delete
  related:
    - mapj-tdn-search
    - mapj-confluence-export
allowed-tools: Bash
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

| Flag | Short | Required | Default | Description |
|------|-------|----------|---------|-------------|
| `--format` | `-f` | No | json | Output format: `json` or `csv` |
| `--output` | `-o` | No | json | CLI output format: `json` or `table` |

## Examples

### Basic SELECT Query

```bash
mapj protheus query "SELECT TOP 10 * FROM SA1010"
```

### Count Query

```bash
mapj protheus query "SELECT COUNT(*) FROM SA1010"
```

### Export as CSV

```bash
mapj protheus query "SELECT * FROM SA1010 WHERE A1_COD = '000001'" --format csv
```

### Complex Query with JOIN

```bash
mapj protheus query "SELECT a.A1_COD, a.A1_NOME, b.B1_DESC FROM SA1010 a INNER JOIN SB1010 b ON a.A1_COD = b.B1_COD"
```

### Filter Results

```bash
mapj protheus query "SELECT * FROM SA1010 WHERE A1_MSBLQL != '1' AND A1_LOJA != ''"
```

### Pagination with TOP

```bash
# First 100 rows
mapj protheus query "SELECT TOP 100 * FROM SA1010"

# Next 100 rows (OFFSET)
mapj protheus query "SELECT * FROM SA1010 ORDER BY A1_COD OFFSET 100 ROWS FETCH NEXT 100 ROWS ONLY"
```

## Prerequisites

1. **Authenticate first**:
   ```bash
   mapj auth login protheus \
     --server 192.168.99.102 \
     --port 1433 \
     --database P1212410_BIB \
     --user USERNAME \
     --password PASSWORD
   ```

2. **Verify auth**:
   ```bash
   mapj auth status
   ```

## Output Schema

### JSON Format (Default)

```json
{
  "ok": true,
  "command": "mapj protheus query \"SELECT TOP 10 * FROM SA1010\"",
  "result": {
    "columns": ["A1_COD", "A1_LOJA", "A1_NOME", "A1_NREDUZ"],
    "rows": [
      ["000001", "01", "CLIENTE TESTE LTDA", "CLIENTE TESTE"],
      ["000002", "01", "FORNECEDOR ABC", "FORNECEDOR ABC"]
    ],
    "count": 2
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### CSV Format

```json
{
  "ok": true,
  "command": "mapj protheus query \"SELECT * FROM SA1010\" --format csv",
  "result": {
    "format": "csv",
    "content": "A1_COD,A1_LOJA,A1_NOME,A1_NREDUZ\n000001,01,CLIENTE TESTE LTDA,CLIENTE TESTE\n000002,01,FORNECEDOR ABC,FORNECEDOR ABC"
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

### Error Response

```json
{
  "ok": false,
  "command": "mapj protheus query \"INSERT INTO SA1010...\"",
  "error": {
    "code": "USAGE_ERROR",
    "message": "query contains forbidden keyword: INSERT. Only SELECT queries are allowed.",
    "retryable": false
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-26T23:00:00Z"
}
```

## Security: SELECT-Only Enforcement

**CRITICAL**: Only SELECT queries are allowed. This is a security constraint to prevent data corruption.

### Forbidden Keywords

The following SQL keywords are blocked:

| Category | Keywords |
|----------|----------|
| DML | INSERT, UPDATE, DELETE, MERGE |
| DDL | CREATE, ALTER, DROP, TRUNCATE |
| DCL | GRANT, REVOKE, DENY |
| Exec | EXEC, EXECUTE, SP_ |
| Backup | BACKUP, RESTORE |

### Blocked Examples

```bash
# These will all fail with USAGE_ERROR (code 2)
mapj protheus query "INSERT INTO SA1010 VALUES (...)"
mapj protheus query "UPDATE SA1010 SET A1_NOME = 'new'"
mapj protheus query "DELETE FROM SA1010 WHERE A1_COD = '000001'"
mapj protheus query "DROP TABLE SA1010"
mapj protheus query "EXEC some_procedure"
```

### Validation Pattern

Queries are validated using case-insensitive regex matching for forbidden keywords before execution.

## Common Protheus Tables

| Table | Description | Key Fields |
|-------|-------------|------------|
| SA1010 | Clientes (Customers) | A1_COD, A1_LOJA, A1_NOME, A1_NREDUZ |
| SB1010 | Productos (Products) | B1_COD, B1_DESC, B1_TIPO |
| SC7010 | Pedidos (Orders) | C7_NUM, C7_CLIENTE, C7_PRODUTO |
| SF2010 | Notas Fiscais | F2_DOC, F2_SERIE, F2_CLIENTE |
| SPED050 | SPED Fiscal | Various fiscal fields |

**Note**: Protheus tables typically use 6-character prefix + version suffix (e.g., SA1**010**).

## Error Cases

| Code | Condition | Resolution |
|------|-----------|------------|
| USAGE_ERROR (2) | Non-SELECT query or syntax error | Use only SELECT statements |
| AUTH_ERROR (3) | Not authenticated | Run `mapj auth login protheus ...` |
| QUERY_ERROR (1) | Database error | Check query syntax, table exists |
| CONNECTION_ERROR (1) | Cannot connect | Verify server:port/database |

## Limitations

- **SELECT only**: No INSERT, UPDATE, DELETE, DROP, or any data modification
- **No stored procedures**: EXEC/SP_ calls blocked
- **Large result sets**: Use `TOP N` to limit rows
- **Connection timeout**: 30 seconds default
- **No transactions**: Auto-commit only

## Performance Tips

```bash
# Always limit results for large tables
mapj protheus query "SELECT TOP 100 * FROM SA1010 ORDER BY A1_COD"

# Use proper WHERE clauses to filter
mapj protheus query "SELECT * FROM SA1010 WHERE A1_COD >= '000100' AND A1_COD <= '000200'"

# Avoid SELECT * on large tables - specify columns
mapj protheus query "SELECT A1_COD, A1_NOME, A1_NREDUZ FROM SA1010"
```

## Workflow Example

```bash
#!/bin/bash
# Generate a customer report

# Query active customers
mapj protheus query "SELECT A1_COD, A1_LOJA, A1_NOME, A1_NREDUZ, A1_EST FROM SA1010 WHERE A1_MSBLQL != '1'" --format csv > customers.csv

# Check row count
echo "Report generated: $(wc -l < customers.csv) rows"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Query successful |
| 1 | Query failed (error) |
| 2 | Usage error (non-SELECT or syntax) |
| 3 | Auth error (not authenticated) |

## See Also

- [SKILL.md](../SKILL.md) - Main manifest
- [mapj-tdn-search](mapj-tdn-search.md) - Search TDN documentation
- [mapj-confluence-export](mapj-confluence-export.md) - Export Confluence pages
