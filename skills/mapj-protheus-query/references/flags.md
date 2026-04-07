# Protheus Query — Flags Reference

Complete reference for all `mapj protheus query`, `mapj protheus connection`, and `mapj protheus schema` flags.

---

## `mapj protheus query "<SQL>"` flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--max-rows` | int | `10000` | Client-side row cap. The CLI closes the DB cursor early for efficiency. |
| `--connection` | string | *(active)* | Run against this named profile **without** changing the active connection |
| `--output-file` | string | *(stdout)* | Write result to file; stdout receives only a summary |

### Global output flag (all commands)

| Flag | Default | Description |
|---|---|---|
| `-o` / `--output` | `auto` | `auto` = TOON for tables, LLM for others. `llm` = compact JSON, `toon` = tabular YAML. |

---

## `mapj protheus schema <table_name>` flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--connection` | string | *(active)* | Run against this named profile |

---

## `--connection` — Cross-environment queries

Run a query against any registered profile without touching the active connection.

```bash
# Active is TOTALPEC_BIB. Query PRD without switching:
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection TOTALPEC_PRD
```

---

## `--output-file` — Large result sets

Write the full result to disk instead of stdout. Stdout only gets a summary.

```bash
mapj protheus query "SELECT * FROM SA1010" --output-file ./sa1010.toon
# stdout → {"ok":true,"command":"...","result":{"rows":1500,"columns":45,"format":"toon","output_file":"./sa1010.toon"}}
```

**Safety Tripwire:** If you query > 500 rows and forget `--output-file`, the CLI will automatically trigger a fallback to a temporary `.toon` file to protect the LLM context.

---

## `--max-rows` — Row cap

The CLI will read up to N rows and then **close the cursor early**. This tells the SQL Server to stop sending data, saving bandwidth and time.

```bash
# Efficient client-side cap
mapj protheus query "SELECT * FROM SA1010" --max-rows 100
```

---

## `mapj protheus connection` subcommands

| Subcommand | Description |
|---|---|
| `connection add <name> --server ... --database ... --user ... --password ...` | Register new profile |
| `connection list` | List all profiles (marks active) |
| `connection use <name>` | Switch active profile |
| `connection ping [name]` | Test connectivity (VPN hint on failure) |
| `connection show [name]` | Show profile details (masked) |
| `connection remove <name>` | Delete profile |
