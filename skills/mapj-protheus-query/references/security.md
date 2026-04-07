# Protheus Security Rules — Read-Only Enforcement

The CLI enforces read-only access to Protheus SQL Server databases.

---

## Validation Algorithm (Prefix-Based)

Instead of searching for "dangerous" keywords anywhere in the query (which could be bypassed with comments), the CLI uses a strict **Prefix Validation** strategy:

1. **Clean**: Leading SQL comments (`--` and `/* */`) are stripped.
2. **Normalize**: The resulting query is trimmed and uppercased.
3. **Check**: The query **MUST** start with one of these keywords:
   - `SELECT`
   - `WITH` (for CTEs)
   - `EXEC` (for safe metadata procedures)
4. **Deny**: Any query that does not start with these keywords is rejected immediately with a `USAGE_ERROR`.

---

## Allowed Query Patterns

```sql
-- ✅ Standard SELECT
SELECT TOP 10 * FROM SA1010

-- ✅ CTE (WITH clause)
WITH cte AS (SELECT A1_COD FROM SA1010)
SELECT * FROM cte

-- ✅ Safe Metadata Exec
EXEC sp_help SA1010
```

---

## Blocked Patterns

```sql
-- ❌ DML (Data Modification)
INSERT INTO SA1010 ...
UPDATE SA1010 SET ...
DELETE FROM SA1010 ...

-- ❌ DDL (Data Definition)
DROP TABLE SA1010
ALTER TABLE SA1010 ...
CREATE TABLE ...

-- ❌ Data Movement
SELECT * INTO #temp FROM SA1010  -- blocked because INTO is not a valid prefix
```

---

## Getting Around Blocked Admin Operations

| Blocked pattern | Recommended Alternative |
|---|---|
| `SELECT * INTO #t` | Use **CTEs** (`WITH`) or **Subqueries** |
| `CREATE TABLE` | Not supported. Use temp logic within a single SELECT if possible |
| Administrative SPs | Use the `mapj protheus schema <table_name>` command |
