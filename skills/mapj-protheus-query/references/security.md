# Protheus Security Rules — Blocked SQL Keywords

All keywords blocked BEFORE the query reaches the database.

---

## Validation Algorithm

1. Strip all SQL comments (`--` and `/* */`)
2. Uppercase the cleaned query
3. Check each blocked keyword using word-boundary regex `\bKEYWORD\b`
4. Verify query starts with `SELECT` or `WITH`
5. If any check fails → return `USAGE_ERROR` (exit 2, retryable: false)

---

## Blocked Keywords by Category

| Category | Keywords |
|---|---|
| DML | `INSERT`, `UPDATE`, `DELETE`, `MERGE` |
| DDL | `CREATE`, `ALTER`, `DROP`, `TRUNCATE` |
| DCL | `GRANT`, `REVOKE`, `DENY` |
| Execution | `EXEC`, `EXECUTE` |
| Data movement | `INTO` ← blocks SELECT INTO as well |
| Backup | `BACKUP`, `RESTORE` |
| Other | `REPLACE` |

---

## The `INTO` Special Case

`INTO` is blocked to prevent `SELECT INTO #temp`. This also means:
- ❌ `SELECT * INTO #temp FROM SA1010` — blocked
- ❌ `INSERT INTO SA1010` — blocked (INSERT also blocked)

Workarounds for temp table patterns:
```sql
-- ✅ Use CTE instead of SELECT INTO
WITH temp_data AS (
    SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL != '1'
)
SELECT COUNT(*) FROM temp_data;

-- ✅ Use subquery
SELECT COUNT(*) FROM (
    SELECT A1_COD FROM SA1010 WHERE A1_MSBLQL != '1'
) AS sub;
```

---

## Allowed Patterns

```sql
-- ✅ Standard SELECT
SELECT TOP 10 * FROM SA1010

-- ✅ CTE (WITH clause)
WITH cte AS (SELECT A1_COD FROM SA1010)
SELECT * FROM cte

-- ✅ Subquery
SELECT * FROM SA1010 WHERE A1_COD IN (SELECT A2_COD FROM SA2010)

-- ✅ Aggregation
SELECT COUNT(*), MAX(A1_COD) FROM SA1010

-- ✅ JOIN
SELECT a.A1_COD, b.A2_COD FROM SA1010 a JOIN SA2010 b ON a.A1_COD = b.A2_COD

-- ✅ ORDER BY, GROUP BY, HAVING
SELECT A1_ESTADO, COUNT(*) AS qty FROM SA1010 GROUP BY A1_ESTADO ORDER BY qty DESC
```

---

## Getting Around Blocked Admin Operations

| Blocked operation | Alternative (allowed) |
|---|---|
| `EXEC sp_help SA1010` | `SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'SA1010'` |
| `EXEC sp_tables` | `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE'` |
| `SELECT * INTO #t FROM ...` | Use CTE or subquery |
| `CREATE TABLE temp` | Not needed — use CTEs |
