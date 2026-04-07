---
name: mapj-protheus-query
description: >
  Execute SELECT queries on Protheus ERP SQL Server and manage named connection profiles.
  Use when: querying Protheus tables, generating ERP reports, managing named connections,
  discovering table structure, or comparing data between environments.
  Do NOT use when: writing to Protheus, DML/DDL operations, running stored procedures,
  or querying non-Protheus SQL Server instances.
  Triggers: "query Protheus", "SELECT from Protheus", "Protheus database", "ERP query",
  "switch Protheus connection", "ping Protheus", "list connections", "add connection",
  "Protheus tables", "report from Protheus", "compare environments", "table schema".
compatibility: Requires mapj CLI at PATH. VPN required: TOTALPEC (192.168.99.x), UNION (192.168.7.x).
metadata:
  version: 3.2.0
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
    - schema-discovery
    - auto-file-fallback
  security:
    - select-only-enforced
    - prefix-validation
    - no-dml
    - no-ddl
  related:
    - mapj-tdn-search
    - mapj-confluence-export
allowed-tools: Bash
---

# mapj protheus — Agent Skill v3.2

Execute SELECT queries on Protheus ERP SQL Server and manage named connection profiles.
All queries are **read-only** — the CLI enforces a strict **prefix-based validation** 
(SELECT, WITH, EXEC) before reaching the database.

> ⚠️ **Security**: DML/DDL operations are blocked. No exceptions.
> ⚠️ **VPN required**: Internal servers require VPN connection.

---

## Prerequisites

```bash
# Check if connection is configured
mapj auth status
# → {"ok":true,"result":{"protheus":{"authenticated":true,"activeProfile":"TOTALPEC_BIB",...},...}}

# If not configured → add first profile
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --database P1212410_BIB --user U --password P --use
```

---

## Connection Management Workflow

```bash
mapj protheus connection list                    # list all + active
mapj protheus connection use TOTALPEC_PRD        # switch active
mapj protheus connection ping                    # test active + VPN hint
mapj protheus connection show                    # show active details (masked)
mapj protheus connection remove TOTALPEC_DESII   # delete profile
```

---

## Schema Discovery

AI Agents should always check the table structure before querying to avoid hallucinations:

```bash
# Get columns and types for a table
mapj protheus schema SA1010
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
├─ Result too large for context? (> 500 rows)
│   → CLI automatically triggers a **Safety Tripwire**
│   → Saves to a temp .toon file and returns a summary
│   → Manually force: --output-file ./result.toon
│
└─ Prefer compact tabular output (default)
    mapj protheus query "SELECT A1_COD, A1_NOME FROM SA1010" -o toon
```

### Context Protection

Large query results saturate the LLM context.
- **Safety Tripwire**: If result > 500 rows and no `--output-file` is set, CLI auto-saves to disk.
- **Manual Redirect**: Use `--output-file` for any size to keep stdout clean.
- **Max Rows**: Use `--max-rows N` to cap the result. The CLI will close the DB cursor early for efficiency.

### Essential queries

```bash
# Verify active DB
mapj protheus query "SELECT DB_NAME() AS bd, @@SERVERNAME AS srv"

# Describe a table (via schema command)
mapj protheus schema SA1010

# Count table rows
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"

# Top N rows
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"

# Filtered query with CTE
mapj protheus query "WITH active AS (SELECT A1_COD FROM SA1010 WHERE A1_MSBLQL != '1') SELECT COUNT(*) FROM active"
```

---

## Error Reference

| Code | Condition | Fix |
|---|---|---|
| `NOT_AUTHENTICATED` | No profile configured | `mapj protheus connection add ...` |
| `USAGE_ERROR` | SQL doesn't start with SELECT/WITH/EXEC | Rewrite as read-only |
| `QUERY_ERROR` | Server unreachable or bad creds | Check VPN hint or connection details |
| `FILE_WRITE_ERROR` | Output path inaccessible | Check disk permissions |

---

## Extended Reference

| Need | File |
|---|---|
| All known connections (TOTALPEC + UNION) | `references/connections.md` |
| Security rules & prefix validation | `references/security.md` |
| All flags for query and connection commands | `references/flags.md` |
