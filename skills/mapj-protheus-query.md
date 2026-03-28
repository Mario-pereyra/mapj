---
name: mapj-protheus-query
description: >
  Execute SELECT queries on Protheus ERP SQL Server database for reporting and data retrieval.
  Use when: querying Protheus database tables, generating reports, extracting data from ERP,
  or managing named Protheus connection profiles (add, list, switch, remove, ping).
  Also use when: configuring a Protheus connection, switching between databases (BIB/PRD/DES),
  verifying connection status, testing connectivity, or comparing data between environments.
  Triggers: "query Protheus", "SELECT from Protheus", "Protheus database", "ERP query",
  "extract Protheus data", "SQL query Protheus", "Protheus tables", "report from Protheus",
  "configure Protheus", "switch Protheus database", "connect to Protheus", "ping Protheus",
  "list Protheus connections", "add Protheus connection".
metadata:
  version: 3.0.0
  language: en
  author: Mario Pereira
  tags:
    - protheus
    - totvs
    - erp
    - database
    - sql
    - query
    - select
    - reporting
    - mssql
    - sqlserver
  capabilities:
    - query
    - select-only
    - connection-management
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

# mapj protheus — Agent Skill v2.0

Execute SELECT queries on the Protheus ERP SQL Server database, and manage the active connection.

> **Important:** The CLI stores **one active Protheus connection** at a time (encrypted).
> To switch databases, simply re-run `auth login protheus` with the new credentials.

---

## Connection Architecture

Protheus runs on **Microsoft SQL Server**. The CLI connects using the Go `mssql` driver with:

```
server=HOST;port=PORT;database=DATABASE;user id=USER;password=PASS;encrypt=disable
```

The `encrypt=disable` setting matches the `trustServerCertificate: true` / `encrypt: false`
configuration used in the environment. This is required for internal servers without TLS.

---

## Known Connections (TOTALPEC environment)

All on the same server: `192.168.99.102:1433`

| Name | Database | User | Purpose |
|------|----------|------|---------|
| **TOTALPEC_BIB** | `P1212410_BIB` | `P1212410_BIB` | **Default — BI/Consulting** |
| TOTALPEC_PRD | `P1212410_PRD` | `P1212410_PRD` | Production |
| TOTALPEC_DES | `P1212410_DES` | `P1212410_DES` | Development |
| TOTALPEC_DESII | `P1212410_DESII` | `P1212410_DESII` | Development II |

**UNION environment** (different servers):

| Name | Server | Database | User |
|------|--------|----------|------|
| UNION_BIB | 192.168.7.97 | P1212410_BIB | P1212410_BIB |
| UNION_PRD | 192.168.7.215 | P1212410_PRD | P1212410_PRD |
| UNION_UPG | 192.168.7.135 | P1212410_UPG | P1212410_UPG |

> ⚠️ Unless told otherwise, **always use TOTALPEC_BIB** (192.168.99.102, P1212410_BIB).

---

## Managing the Connection

### Check current connection

```bash
mapj auth status
```

Output:
```
Authentication Status:
  TDN:        ✓ authenticated
  Confluence: ✓ authenticated
  Protheus:   ✓ authenticated     ← only shows authenticated/not, not which DB
```

> `auth status` does NOT show which database is active. If you need to know,
> you must query the DB: `mapj protheus query "SELECT DB_NAME()"`.

### Configure / Switch connection (Add or Modify)

To set or change the active Protheus connection:

```bash
# Default: TOTALPEC BIB
mapj auth login protheus \
  --server 192.168.99.102 \
  --port 1433 \
  --database P1212410_BIB \
  --user P1212410_BIB \
  --password P1212410_BIB

# Switch to Production
mapj auth login protheus \
  --server 192.168.99.102 \
  --port 1433 \
  --database P1212410_PRD \
  --user P1212410_PRD \
  --password P1212410_PRD

# Switch to Development II
mapj auth login protheus \
  --server 192.168.99.102 \
  --port 1433 \
  --database P1212410_DESII \
  --user P1212410_DESII \
  --password P1212410_DESII

# Switch to UNION BIB (different server)
mapj auth login protheus \
  --server 192.168.7.97 \
  --port 1433 \
  --database P1212410_BIB \
  --user P1212410_BIB \
  --password P1212410_BIB
```

The new login **always overwrites** the previous connection. There is no confirmation prompt.

### Verify active database after login

```bash
# Always verify which DB is actually active after switching
mapj protheus query "SELECT DB_NAME() AS active_database, @@SERVERNAME AS server"
```

### Remove connection (Logout)

```bash
# Removes all stored Protheus credentials
mapj auth logout protheus
```

After logout, any `protheus query` will return `NOT_AUTHENTICATED` error.

---

## Executing Queries

### Usage

```
mapj protheus query "<sql>" [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `json` | Output format: `json` or `csv` |
| `--max-rows` | `10000` | Truncate result to N rows. `0` = no limit |
| `--connection` | (active) | Run against a specific named profile **without changing the active** |

### Using --connection (cross-env queries without switching)

```bash
# Currently active is TOTALPEC_BIB, but query PRD without switching
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection TOTALPEC_PRD

# Compare count between BIB and PRD in sequence
mapj protheus query "SELECT COUNT(*) FROM SA1010" # BIB (active)
mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD
```

### Output Formats

#### JSON (default)

```bash
mapj protheus query "SELECT TOP 5 A1_COD, A1_NOME FROM SA1010"
```

```json
{
  "ok": true,
  "command": "mapj protheus query",
  "result": {
    "columns": ["A1_COD", "A1_NOME"],
    "rows": [
      ["010715", "PABLO ALBERTO SAUTO RODRIGUEZ"],
      ["010102", "GLOVERT ESTEBAN EGUEZ FOIANINI"]
    ],
    "count": 2
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T21:41:47Z"
}
```

#### CSV

```bash
mapj protheus query "SELECT TOP 5 A1_COD, A1_NOME FROM SA1010" --format csv
```

```json
{
  "ok": true,
  "command": "mapj protheus query",
  "result": {
    "format": "csv",
    "content": "A1_COD,A1_NOME\n010715,PABLO ALBERTO SAUTO RODRIGUEZ\n010102,GLOVERT ESTEBAN EGUEZ FOIANINI"
  }
}
```

> Note: CSV format returns the CSV as a JSON string in `result.content`, not raw text.

---

## Security: SELECT-Only Enforcement

**CRITICAL**: Only SELECT queries are allowed. This is enforced in code BEFORE the query
reaches the database. No bypass is possible.

### Allowed

```sql
-- Standard SELECT
SELECT TOP 10 * FROM SA1010

-- CTEs (WITH clause)
WITH cte AS (SELECT A1_COD FROM SA1010)
SELECT * FROM cte

-- Subqueries
SELECT * FROM SA1010 WHERE A1_COD IN (SELECT A1_COD FROM SA2010)

-- Aggregations, JOINs, ORDER BY — all fine
SELECT COUNT(*) FROM SA1010
```

### Blocked — Will return `USAGE_ERROR`

| Category | Keywords Blocked |
|----------|-----------------|
| DML | `INSERT`, `UPDATE`, `DELETE`, `MERGE` |
| DDL | `CREATE`, `ALTER`, `DROP`, `TRUNCATE` |
| DCL | `GRANT`, `REVOKE`, `DENY` |
| Execution | `EXEC`, `EXECUTE` |
| Data movement | `INTO` (blocks SELECT INTO), `REPLACE` |
| Backup | `BACKUP`, `RESTORE` |

> ⚠️ **The `INTO` keyword is blocked.** This means `SELECT ... INTO #temp` is not allowed.
> Use subqueries or CTEs instead for temporary aggregation.

### Validation Logic

1. Strip SQL comments (`--` and `/* */`)
2. Uppercase the query
3. Check for forbidden keywords using word-boundary regex (`\bKEYWORD\b`)
4. Verify the query starts with `SELECT` or `WITH`
5. If any check fails → return `validation error: query contains forbidden keyword: X`

---

## Query Patterns (Generic)

```bash
# Check active database
mapj protheus query "SELECT DB_NAME() AS db, @@SERVERNAME AS server, @@VERSION AS version"

# List tables in current database
mapj protheus query "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"

# Describe a table structure
mapj protheus query "SELECT COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'SA1010' ORDER BY ORDINAL_POSITION"

# Count rows in a table
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"

# Paginate large results
mapj protheus query "SELECT TOP 100 * FROM SA1010 ORDER BY A1_COD"
mapj protheus query "SELECT * FROM SA1010 ORDER BY A1_COD OFFSET 100 ROWS FETCH NEXT 100 ROWS ONLY"

# Limit via max-rows flag (soft cap on client side, after DB execution)
mapj protheus query "SELECT * FROM SA1010" --max-rows 500

# Export to CSV string
mapj protheus query "SELECT A1_COD, A1_NOME FROM SA1010" --format csv

# CTE example
mapj protheus query "WITH activos AS (SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL != '1') SELECT COUNT(*) AS activos FROM activos"
```

---

## Error Handling

### Error Schema

```json
{
  "ok": false,
  "command": "mapj protheus query",
  "error": {
    "code": "QUERY_ERROR",
    "message": "query failed: ...",
    "retryable": true
  }
}
```

### Error Codes

| Code | Condition | Retryable | Resolution |
|------|-----------|-----------|------------|
| `NOT_AUTHENTICATED` | No Protheus creds stored | No | Run `mapj auth login protheus ...` |
| `USAGE_ERROR` | Non-SELECT or forbidden keyword | No | Rewrite query with only SELECT |
| `QUERY_ERROR` | DB execution error (syntax, table not found) | Yes | Fix SQL syntax or table name |
| `QUERY_ERROR` | Connection refused / timeout | Yes | Check network, re-login |

### Common Error Messages

| Message contains | Meaning |
|-----------------|---------|
| `validation error: query contains forbidden keyword` | Hit the SELECT-only guard |
| `validation error: query must be a SELECT statement` | Query doesn't start with SELECT/WITH |
| `failed to connect` | Wrong server/port/credentials |
| `login error` | Wrong user/password |
| `Invalid object name` | Table doesn't exist in this database |

---

## Agentic Decision Tree

```
Need to interact with Protheus?
│
├─ List available profiles
│   mapj protheus connection list
│
├─ Is Protheus configured?
│   mapj auth status → "Protheus: ✓ authenticated [...]"
│   If NOT → mapj protheus connection add <name> --server ... (use TOTALPEC_BIB by default)
│
├─ Need to check which DB is active?
│   mapj auth status  ←  now shows active profile name and DB
│
├─ Need to switch to a different DB?
│   mapj protheus connection use <name>   ← no credentials re-entry
│
├─ Need to test if server is reachable?
│   mapj protheus connection ping [name]
│
├─ Need to query a specific DB without switching?
│   mapj protheus query "SELECT ..." --connection OTHER_PROFILE
│
├─ The query involves INSERT/UPDATE/DELETE?
│   → NOT ALLOWED. Read-only only.
│
└─ Execute query on active profile
    mapj protheus query "SELECT ..." [--format csv] [--max-rows N]
```

---

## Workflow: Switch DB, Query, Restore

```bash
#!/bin/bash
# Switch to PRD, run a query, switch back to BIB

# 1. Save current (BIB is default)
# 2. Switch to PRD
mapj auth login protheus \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_PRD --user P1212410_PRD --password P1212410_PRD

# 3. Verify
mapj protheus query "SELECT DB_NAME()"

# 4. Query PRD
mapj protheus query "SELECT TOP 10 A1_COD FROM SA1010"

# 5. Switch back to BIB
mapj auth login protheus \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB
```

---

## Limitations

- **Single active connection**: Only one Protheus connection at a time (no connection pooling at CLI level)
- **SELECT only**: No INSERT, UPDATE, DELETE, DROP, EXEC, or any data modification
- **No stored procedures**: EXEC/EXECUTE blocked
- **`INTO` blocked**: Cannot use `SELECT INTO #temp` — use CTEs or subqueries
- **`--max-rows` is client-side**: The DB still processes the full query; `--max-rows` just truncates the returned rows. Use `TOP N` in SQL for true DB-side limiting
- **No transactions**: Each query is a single auto-commit operation
- **Connection timeout**: 30 seconds (hardcoded in the driver)
- **CSV escaping**: Current CSV output does NOT escape commas within field values. Use JSON format if field values may contain commas

---

## See Also

- [SKILL.md](SKILL.md) — Main manifest
- [mapj-tdn-search](mapj-tdn-search.md) — Search TDN documentation
- [mapj-confluence-export](mapj-confluence-export.md) — Export Confluence pages
