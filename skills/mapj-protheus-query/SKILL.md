---
name: mapj-protheus-query
description: >
  Execute SELECT queries on Protheus ERP SQL Server and manage named connection profiles.
  Use when: querying Protheus tables, generating ERP reports, managing named connections,
  discovering table structure, comparing data between environments, or managing query presets.
  Do NOT use when: writing to Protheus, DML/DDL operations, running stored procedures,
  or querying non-Protheus SQL Server instances.
  Triggers: "query Protheus", "SELECT from Protheus", "Protheus database", "ERP query",
  "switch Protheus connection", "ping Protheus", "list connections", "add connection",
  "Protheus tables", "report from Protheus", "compare environments", "table schema",
  "preset", "saved query", "parameterized query", "query template".
compatibility: Requires mapj CLI at PATH. VPN required: TOTALPEC (192.168.99.x), UNION (192.168.7.x).
metadata:
  version: 4.0.0
  language: en
  author: Mario Pereira
  tags:
    - protheus
    - totvs
    - erp
    - sql
    - database
    - mssql
    - presets
    - parameterized-queries
  capabilities:
    - query
    - select-only
    - connection-management
    - schema-discovery
    - auto-file-fallback
    - preset-management
    - parameter-interpolation
  security:
    - select-only-enforced
    - prefix-validation
    - no-dml
    - no-ddl
    - sql-injection-protection
  related:
    - mapj-tdn-search
    - mapj-confluence-export
allowed-tools: Bash
---

# mapj protheus — Agent Skill v4.0

Execute SELECT queries on Protheus ERP SQL Server, manage named connection profiles, and work with query presets.
All queries are **read-only** — the CLI enforces a strict **prefix-based validation**
(SELECT, WITH, EXEC) before reaching the database.

> ⚠️ **Security**: DML/DDL operations are blocked. No exceptions.
> ⚠️ **VPN required**: Internal servers require VPN connection.

---

## Role

Eres un agente especializado en consultar bases de datos Protheus ERP. Tu responsabilidad es
ejecutar queries SELECT-only sobre tablas del ERP. Operas con validación estricta de seguridad SQL
(prefix-based: SELECT, WITH, EXEC permitidos). Nunca modificas datos.

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

## Preset System Overview

Presets allow you to save frequently used queries with parameter definitions for reuse.
Parameters use `:placeholder` syntax and are automatically detected and validated.

**Storage**: `~/.config/mapj/presets.json` (JSON format, 0600 permissions)

**Key Features**:
- Automatic parameter detection (`:param` syntax)
- Type validation (string, int, date, datetime, bool, list)
- SQL injection protection with pattern detection
- Default values for optional parameters
- Agent-friendly JSON output (no interactive prompts)

---

## Preset Commands Reference

### `preset add <name>` — Create a new preset

```bash
mapj protheus preset add <name> --query "SQL with :placeholders" [flags]
```

**Required Flags**:
- `--query TEXT` — SQL query with `:parameter` placeholders

**Optional Flags**:
- `--description TEXT` — Human-readable description
- `--connection NAME` — Default connection profile to use
- `--max-rows N` — Default row limit (0 = unlimited)
- `--param-def DEF` — Parameter definition (repeatable). Format: `name:type[:default][:description]`
- `--tags TAGS` — Comma-separated tags (e.g., "report,daily")
- `--use` — Set as active preset immediately

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "name": "my-preset",
    "query": "SELECT :name FROM users WHERE id = :id",
    "detectedParameters": ["name", "id"],
    "parameters": [{"name": "id", "type": "int", "required": true}],
    "createdAt": "2024-01-15T10:30:00Z",
    "updatedAt": "2024-01-15T10:30:00Z"
  }
}
```

**Examples**:
```bash
# Simple preset without parameters
mapj protheus preset add client-count \
  --query "SELECT COUNT(*) AS total FROM SA1010" \
  --description "Count all clients" \
  --tags "report,daily"

# Parameterized preset with type definitions
mapj protheus preset add client-by-status \
  --query "SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL = :status" \
  --param-def "status:string:2:Block status (1=blocked, 2=active)" \
  --connection TOTALPEC_BIB \
  --max-rows 1000 \
  --use

# Preset with multiple parameters
mapj protheus preset add orders-by-date \
  --query "SELECT * FROM SC5010 WHERE C5_EMISSAO BETWEEN :start_date AND :end_date AND C5_TIPO = :order_type" \
  --param-def "start_date:date" \
  --param-def "end_date:date" \
  --param-def "order_type:string:N" \
  --tags "report,orders"
```

---

### `preset list` — List all presets

```bash
mapj protheus preset list [flags]
```

**Optional Flags**:
- `--tag TAG` — Filter by tag
- `--connection NAME` — Filter by connection profile

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "presets": [
      {
        "name": "client-count",
        "query": "SELECT COUNT(*) AS total FROM SA1010",
        "tags": ["report", "daily"],
        "active": true
      }
    ],
    "count": 1
  }
}
```

**Examples**:
```bash
# List all presets
mapj protheus preset list

# Filter by tag
mapj protheus preset list --tag report

# Filter by connection
mapj protheus preset list --connection TOTALPEC_PRD
```

---

### `preset show [name]` — Show preset details

```bash
mapj protheus preset show [name]
```

- Without name: shows the active preset
- Returns `activePreset: null` if no active preset is set

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "name": "client-by-status",
    "query": "SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL = :status",
    "parameters": [
      {
        "name": "status",
        "type": "string",
        "required": true,
        "default": "2",
        "description": "Block status (1=blocked, 2=active)"
      }
    ],
    "connection": "TOTALPEC_BIB",
    "maxRows": 1000,
    "active": true
  }
}
```

**Examples**:
```bash
# Show specific preset
mapj protheus preset show client-by-status

# Show active preset
mapj protheus preset show
```

---

### `preset run <name>` — Execute a preset query

```bash
mapj protheus preset run <name> [flags]
```

**Optional Flags**:
- `--param KEY=VALUE` — Parameter value (repeatable). Example: `--param status=1 --param date=2024-01-15`
- `--connection NAME` — Override preset's connection profile
- `--max-rows N` — Override preset's row limit
- `--output-file PATH` — Write results to file instead of stdout

**Connection Resolution** (priority order):
1. `--connection` flag (highest)
2. Preset's saved connection
3. Active connection profile
4. Error: `NO_CONNECTION`

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "columns": ["A1_COD", "A1_NOME"],
    "rows": [["000001", "CLIENTE ABC"], ["000002", "CLIENTE XYZ"]],
    "count": 2,
    "params_used": {"status": "1"},
    "connection_used": "TOTALPEC_BIB"
  }
}
```

**Output** (with `--output-file`):
```json
{
  "ok": true,
  "result": {
    "rows": 1000,
    "columns": 5,
    "output_file": "./report.json",
    "params_used": {"status": "1"},
    "connection_used": "TOTALPEC_BIB"
  }
}
```

**Examples**:
```bash
# Run preset with defaults
mapj protheus preset run client-count

# Run with parameters
mapj protheus preset run client-by-status --param status=1

# Run with multiple parameters
mapj protheus preset run orders-by-date \
  --param start_date=2024-01-01 \
  --param end_date=2024-01-31 \
  --param order_type=N

# Run on different connection
mapj protheus preset run client-count --connection TOTALPEC_PRD

# Run and save to file
mapj protheus preset run orders-by-date \
  --param start_date=2024-01-01 \
  --param end_date=2024-01-31 \
  --output-file ./orders.json
```

---

### `preset edit <name>` — Modify a preset

```bash
mapj protheus preset edit <name> [flags]
```

**Optional Flags** (only provided fields are updated):
- `--description TEXT` — Update description
- `--query TEXT` — Update SQL query
- `--connection NAME` — Update default connection
- `--max-rows N` — Update row limit
- `--param-def DEF` — Replace parameter definitions (repeatable)
- `--tags TAGS` — Replace tags (comma-separated)

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "name": "client-count",
    "fields_updated": ["description", "query"],
    "updatedAt": "2024-01-15T12:00:00Z"
  }
}
```

**Examples**:
```bash
# Update description
mapj protheus preset edit client-count --description "Updated description"

# Update query and re-detect parameters
mapj protheus preset edit client-by-status \
  --query "SELECT A1_COD, A1_NOME, A1_EST FROM SA1010 WHERE A1_MSBLQL = :status AND A1_EST = :state" \
  --param-def "status:string:2" \
  --param-def "state:string::State code"
```

---

### `preset remove <name>` — Delete a preset

```bash
mapj protheus preset remove <name> [flags]
```

**Optional Flags**:
- `--force` — Skip confirmation prompt (agent-friendly, always on)

**Output** (success):
```json
{
  "ok": true,
  "result": {
    "removed": "old-preset",
    "was_active": true
  }
}
```

**Note**: If the removed preset was active, the active reference is automatically cleared.

**Examples**:
```bash
# Remove a preset
mapj protheus preset remove old-preset --force
```

---

### `preset use [name]` — Set or show active preset

```bash
mapj protheus preset use [name]
```

- With name: sets that preset as active
- Without name: shows current active preset

**Output** (success - set active):
```json
{
  "ok": true,
  "result": {
    "active_preset": "client-count",
    "preset": {
      "name": "client-count",
      "query": "SELECT COUNT(*) AS total FROM SA1010"
    }
  }
}
```

**Output** (no active preset):
```json
{
  "ok": true,
  "result": {
    "active_preset": null
  }
}
```

**Examples**:
```bash
# Set active preset
mapj protheus preset use client-count

# Show current active preset
mapj protheus preset use
```

---

## Parameter Types Reference

| Type | Format | Validation | SQL Output |
|------|--------|------------|------------|
| `string` | Any value | Accepts all including empty | `'escaped_value'` |
| `int` | Integer | Accepts: positive, negative, zero. Rejects floats | `123` (unquoted) |
| `date` | YYYY-MM-DD | Validates month 1-12, day per month | `'2024-01-15'` |
| `datetime` | YYYY-MM-DD HH:MM:SS or YYYY-MM-DDTHH:MM:SS | Validates date + time components | `'2024-01-15 10:30:00'` |
| `bool` | true/false, TRUE/FALSE, 1/0, yes/no | Normalizes to true/false | `1` or `0` (SQL Server bit) |
| `list` | CSV (a,b,c) | Accepts any value | `'a', 'b', 'c'` (IN clause) |

**Parameter Definition Format**:
```
name:type[:default][:description]
```

**Examples**:
```bash
--param-def "status:string:2:Block status"
--param-def "id:int::User ID"
--param-def "start_date:date"
--param-def "active:bool:true"
--param-def "codes:list:A,B:Coded values"
```

---

## SQL Injection Protection

The preset system includes multi-layer defense against SQL injection:

**Detected Patterns**:
- `; DROP`, `; DELETE`, `; INSERT`, `; UPDATE`, `; TRUNCATE`, `; EXEC`
- `OR 1=1`, `OR '1'='1'`, `AND 1=1` (always-true conditions)
- `UNION SELECT` (data exfiltration)
- `--` comment injection

**Escaping Applied**:
- Single quotes doubled: `'` → `''`
- List values converted to escaped IN clause: `a,b,c` → `'a', 'b', 'c'`

**Error on Detection**:
```json
{
  "ok": false,
  "error": {
    "code": "SQL_INJECTION_DETECTED",
    "message": "potential SQL injection detected in parameter 'name': patterns [DROP, COMMENT_INJECTION]",
    "retryable": false
  }
}
```

---

## Error Codes Reference

| Code | Condition | Fix |
|------|-----------|-----|
| `MISSING_REQUIRED_FIELD` | `--query` not provided for `preset add` | Add `--query "SELECT ..."` |
| `PRESET_EXISTS` | Preset with same name already exists | Use `preset edit <name>` to modify |
| `PRESET_NOT_FOUND` | Preset name doesn't exist | Use `preset list` to see available |
| `INVALID_PARAM_DEF` | Malformed `--param-def` format | Use format: `name:type[:default][:description]` |
| `INVALID_PARAM_NAME` | Parameter name has invalid chars | Use only letters, digits, underscore |
| `NO_FIELDS_TO_UPDATE` | `preset edit` without any flags | Provide at least one field to update |
| `MISSING_PARAMETER` | Required parameter not provided | Add `--param name=value` for missing params |
| `TYPE_MISMATCH` | Parameter value doesn't match type | Check expected type with `preset show` |
| `SQL_INJECTION_DETECTED` | Injection pattern in parameter value | Remove malicious patterns |
| `NO_CONNECTION` | No connection available | Use `--connection` or set active profile |
| `CONNECTION_NOT_FOUND` | Connection profile doesn't exist | Use `connection list` to see available |
| `CONNECTION_FAILED` | Database connection error | Check VPN, credentials, server status |
| `QUERY_VALIDATION_FAILED` | Query validation error | Check query syntax, SELECT-only |
| `STORE_ERROR` | File read/write error | Check file permissions, disk space |
| `FILE_WRITE_ERROR` | Cannot write to `--output-file` | Check directory exists, write permissions |

---

## Agent-Friendly Patterns

**Discover Parameters Before Execution**:
```bash
# Show preset to see parameters and types
mapj protheus preset show my-preset --json
# → {"ok":true,"result":{"parameters":[{"name":"status","type":"string","required":true}]}}
```

**Execute Non-Interactively**:
```bash
# Provide all required params
mapj protheus preset run my-preset --param status=1 --json
# → {"ok":true,"result":{"rows":[...],"count":10}}
```

**Parse JSON Output**:
```bash
# Use jq for programmatic access
mapj protheus preset list --json | jq '.result.count'
mapj protheus preset show my-preset --json | jq '.result.parameters[].name'
```

**Handle Errors Programmatically**:
```json
// All errors have consistent structure
{
  "ok": false,
  "error": {
    "code": "MISSING_PARAMETER",
    "message": "missing required parameters: status, date",
    "hint": "Provide values: --param status=value --param date=value",
    "retryable": false
  }
}
```

**Batch Operations**:
```bash
# Create multiple presets
for name in client-count orders-by-date stock-report; do
  mapj protheus preset add "$name" --query "..."
done

# Run preset on multiple connections
for conn in TOTALPEC_BIB TOTALPEC_PRD; do
  mapj protheus preset run my-report --connection "$conn" --output-file "report_$conn.json"
done
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

## Examples

### Example 1: Verify active database
**Input:** Usuario pregunta "verifica la base de datos activa"
**Command:** `mapj protheus query "SELECT DB_NAME() AS bd, @@SERVERNAME AS srv"`
**Output:**
```json
{
  "ok": true,
  "result": {
    "rows": [{"bd": "P1212410_BIB", "srv": "192.168.99.102"}]
  }
}
```

### Example 2: Query clients table
**Input:** Usuario pregunta "lista los primeros 10 clientes"
**Command:** `mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"`
**Output:** JSON con rows conteniendo A1_COD y A1_NOME

### Example 3: Count table rows
**Input:** Usuario pregunta "cuántos clientes hay en la base"
**Command:** `mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"`
**Output:**
```json
{
  "ok": true,
  "result": {"rows": [{"total": 15432}]}
}
```

### Example 4: Query specific connection without switching
**Input:** Usuario pregunta "consulta en producción sin cambiar la conexión activa"
**Command:** `mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD`
**Output:** JSON con resultado; conexión activa permanece igual

### Example 5: Switch connection profile
**Input:** Usuario pregunta "cambia a la conexión de producción"
**Command:** `mapj protheus connection use TOTALPEC_PRD`
**Output:**
```json
{
  "ok": true,
  "result": {"activeProfile": "TOTALPEC_PRD"}
}
```

---

## Success Criteria

- [ ] Output es JSON válido con `ok: true`
- [ ] Exit code es 0
- [ ] Query retorna rows (o count correcto)
- [ ] Schema válido cuando se usa `mapj protheus schema`
- [ ] SELECT-only enforced (no DML/DDL ejecutado)
- [ ] Conexión activa verificada antes de query

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
