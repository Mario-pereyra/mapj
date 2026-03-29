---
name: mapj-protheus-query
description: >
  Execute SELECT queries on Protheus ERP SQL Server and manage named connection profiles.
  Use when: querying Protheus tables, generating ERP reports, managing named connections,
  switching databases without re-entering credentials, testing server connectivity, or
  comparing data between environments (BIB/PRD/DES) without switching the active connection.
  Do NOT use when: writing to Protheus, DML/DDL operations, running stored procedures,
  or querying non-Protheus SQL Server instances.
  Triggers: "query Protheus", "SELECT from Protheus", "Protheus database", "ERP query",
  "switch Protheus connection", "ping Protheus", "list connections", "add connection",
  "Protheus tables", "report from Protheus", "compare environments".
compatibility: Requires mapj CLI at PATH. VPN required: TOTALPEC (192.168.99.x), UNION (192.168.7.x).
metadata:
  version: 3.1.0
  language: en
  author: Mario Pereira
  tags:
    - protheus
    - totvs
    - erp
    - sql
    - database
    - mssql
  capabilities:
    - query
    - select-only
    - connection-management
  security:
    - select-only-enforced
    - no-dml
    - no-ddl
  related:
    - mapj-tdn-search
    - mapj-confluence-export
allowed-tools: Bash
---

# mapj protheus — Agent Skill v3.0

Execute SELECT queries on Protheus ERP SQL Server and manage named connection profiles.
All queries are **read-only** — DML/DDL is blocked before reaching the database.

> ⚠️ **Security**: INSERT, UPDATE, DELETE, DROP, EXEC, and INTO are blocked. No exceptions.
> ⚠️ **VPN required**: Internal servers are not reachable without VPN.

---

## Prerequisites

```bash
# Check if connection is configured
mapj auth status
# → "Protheus: ✓ authenticated [active: TOTALPEC_BIB → ...]"

# If not configured → add first profile
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB \
  --use
```

---

## Connection Management Workflow

```
Need to manage connections?
│
├─ First time setup → connection add ... --use
│
├─ List profiles → connection list        (shows active with *)
│
├─ Switch environment → connection use <name>
│   (no credentials re-entry)
│
├─ Test reachability → connection ping [name]
│   (shows VPN hint if unreachable)
│
├─ Inspect a profile → connection show [name]
│   (password masked)
│
└─ Remove a profile → connection remove <name>
    (auto-selects next if was active)
```

### Quick connection commands

```bash
mapj protheus connection list                    # list all + active
mapj protheus connection use TOTALPEC_PRD        # switch active
mapj protheus connection ping                    # test active connection
mapj protheus connection ping UNION_BIB          # test specific
mapj protheus connection show                    # show active details
mapj protheus connection remove TOTALPEC_DESII   # delete profile
```

---

## Query Workflow

```
Need to query?
│
├─ Query active connection (default)
│   mapj protheus query "SELECT TOP 10 * FROM SA1010"
│
├─ Query specific connection WITHOUT switching active
│   mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD
│
├─ Result too large for context? → write to file
│   mapj protheus query "SELECT * FROM SA1010" --output-file ./result.json
│   mapj protheus query "SELECT * FROM SA1010" --format csv --output-file ./result.csv
│   # stdout only gets: {"rows": N, "columns": M, "format": "json", "output_file": "./result.json"}
│
├─ Limit rows (add TOP in SQL, or client-side with --max-rows)
│   mapj protheus query "SELECT TOP 100 * FROM SA1010"
│   mapj protheus query "SELECT * FROM SA1010" --max-rows 500
│
└─ Need CSV output
    mapj protheus query "SELECT A1_COD, A1_NOME FROM SA1010" --format csv
    mapj protheus query "SELECT A1_COD, A1_NOME FROM SA1010" --format csv --output-file ./sa1010.csv
```

### Why `--output-file` matters for agents

Large query results (1000+ rows) saturate the LLM context window. With `--output-file`:
- Only a **summary** is returned to stdout (rows, columns, file path)
- The agent can reference the file path for downstream processing
- Supports both JSON and CSV format


### Essential queries

```bash
# Verify active DB
mapj protheus query "SELECT DB_NAME() AS bd, @@SERVERNAME AS srv"

# Count table rows
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"

# Describe a table
mapj protheus query "SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'SA1010' ORDER BY ORDINAL_POSITION"

# Top N rows
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"

# Filtered query with CTE
mapj protheus query "WITH active AS (SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL != '1') SELECT COUNT(*) FROM active"
```

---

## What This Skill Will NOT Do

- ❌ **INSERT, UPDATE, DELETE, MERGE** — data is read-only
  ✅ Use Protheus UI or ADVPL for data modification
- ❌ **EXEC / stored procedures** — blocked
  ✅ Use INFORMATION_SCHEMA queries instead of sp_help
- ❌ **SELECT INTO #temp** — INTO is blocked
  ✅ Use CTEs: `WITH t AS (SELECT ...) SELECT * FROM t`
- ❌ **DDL (CREATE, ALTER, DROP)** — blocked
- ❌ **Non-Protheus databases** — only configured SQL Server connections

---

## Error Reference

| Code | Condition | Fix |
|---|---|---|
| `NOT_AUTHENTICATED` | No profile configured | `mapj protheus connection add ...` |
| `PROFILE_NOT_FOUND` | `--connection NAME` not in list | `mapj protheus connection list` |
| `USAGE_ERROR` | Forbidden SQL keyword | Rewrite as SELECT only |
| `QUERY_ERROR` + i/o timeout | Server unreachable | Check VPN hint in `error.hint` |
| `QUERY_ERROR` + login error | Wrong credentials | `mapj protheus connection show` to verify |
| `Invalid object name` | Table not in this DB | `mapj protheus connection use CORRECT_PROFILE` |
| `FILE_WRITE_ERROR` | `--output-file` path inaccessible | Check directory exists and has write access |

---

## Extended Reference

Load when you need more detail:

| Need | File |
|---|---|
| All known connections (TOTALPEC + UNION) | `references/connections.md` |
| Full list of blocked SQL keywords | `references/security.md` |
| Extended query examples (pagination, JOINs, CTEs) | `references/query-patterns.md` |
| All flags for query and connection commands | `references/flags.md` |
