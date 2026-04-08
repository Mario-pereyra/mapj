# Changelog

All notable changes to `mapj` are documented here.  
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Versioning follows [SemVer](https://semver.org/).

---

## [0.2.0-agentic] — 2026-04-07

### Breaking Changes

Esta versión introduce cambios significativos para optimizar la interacción con agentes LLM. Si vienes de la versión 1.x o 2.x, lee esta guía de migración:

#### 1. Formatos `csv` y `human` eliminados
Los formateadores `CSVFormatter` y `HumanFormatter` han sido removidos. Ahora debes usar `toon` o `json`.

**Antes (v1.x/v2.x):**
```bash
mapj protheus query --sql "SELECT * FROM users" --format csv
mapj protheus query --sql "SELECT * FROM users" --format human
```

**Ahora (v0.2.0+):**
```bash
# TOON: formato tabular optimizado para LLMs (~40% menos tokens)
mapj protheus query --sql "SELECT * FROM users" -o toon

# JSON: formato estructurado clásico
mapj protheus query --sql "SELECT * FROM users" -o json

# Auto: detección automática (TOON para tablas, JSON para objetos)
mapj protheus query --sql "SELECT * FROM users"  # default
```

#### 2. Comando `retry-failed` removido
El comando manual `mapj confluence retry-failed` y el archivo `export-errors.jsonl` han sido eliminados. El sistema ahora implementa reintentos automáticos internos con backoff exponencial.

**Antes (v1.x/v2.x):**
```bash
# Reintentar exports fallidos manualmente
mapj confluence retry-failed
```

**Ahora (v0.2.0+):**
```bash
# No se requiere acción manual
# Los errores HTTP 429 (rate limit) y 50x se reintentan automáticamente
# con backoff exponencial: 1s → 2s → 4s → 8s (máx 4 reintentos)
mapj confluence export --page-id 12345
```

**Nota:** Los errores permanentes (4xx, auth) se reportan inmediatamente sin reintentos.

#### 3. Flag `--limit` renombrado a `--max-results`
En `tdn search`, el flag `--limit` ha sido renombrado a `--max-results` para mayor claridad semántica.

**Antes (v1.x/v2.x):**
```bash
mapj tdn search "protheus" --limit 50
```

**Ahora (v0.2.0+):**
```bash
mapj tdn search "protheus" --max-results 50
```

### Added
- **Safety Tripwire (Protheus)**: Automatically intercepts large query results (> 500 rows) and saves them to a temporary `.toon` file to protect the LLM context window.
- **Auto-Healing (Confluence)**: Native exponential backoff for HTTP 429 (Rate Limit) and 50x (Server Error) in the core client.
- **Concurrent Worker Pool (Confluence)**: Export operations now use 10 concurrent workers for dramatic speed improvements.
- **Auto-Pagination (TDN)**: `tdn search` now automatically paginates internally to reach the requested `--max-results`.
- **Schema Discovery**: New command `mapj protheus schema <table_name>` to view table structure without hallucinations.
- **TOON Format**: Native support for Tabular Object Notation, saving ~40% tokens for tabular data.
- **Auto-Formatting**: Default output mode `auto` detects the best format (TOON for tables, LLM for objects).
- **Strongly Typed Errors**: Implementation of `ExitCoder` interface for reliable, machine-readable exit codes (0-4).

### Changed
- **Default Format**: Changed from `llm` to `auto`.
- **Search Flag**: Renamed `--limit` to `--max-results` in `tdn search`.
- **SQL Validation**: Moved from Regex-based to strict Prefix-based validation (SELECT, WITH, EXEC).
- **Markdown Conversion**: Replaced per-page converter instantiation with a thread-safe Singleton for better performance.
- **Protheus Row Limit**: `--max-rows` now closes the database cursor early, saving server resources and bandwidth.

### Removed
- **Manual Retry**: Deleted `mapj confluence retry-failed` and `export-errors.jsonl`. Resilience is now internal.
- **Debug Noise**: Removed `--debug` and `--dump-debug` flags and the generation of raw HTML debug files.
- **Legacy Formats**: Removed `CSVFormatter` and `HumanFormatter`. Purgued all "pretty print" human-centric logic.
- **Legacy Auth Logic**: Removed `ProtheusCreds` (v1) migration and structural redundancy in credential storage.

---

## [2.0.1] — 2026-03-29

### Fixed
- **JSON Envelope — Complete Coverage**: `protheus connection` commands now emit structured JSON.
- **Auth commands**: Now respect the global `-o/--output` flag.

### Added
- **Self-Describing Help Text**: Every command now answers LLM-agent questions inline via `--help`.

---

## [2.0.0] — 2026-03-29

### Breaking Changes
- **Default output mode changed**: `--output` default is `llm` (compact JSON) instead of `json` (pretty).
- **Auth commands**: Now emit structured JSON instead of human-readable text.

### Added
- **TDN Search v2.1**: Added `--check-children`, `--since`, `--ancestor`, `--labels`, `--spaces`.
- **Protheus Query**: Added `--output-file <path>`.
- **Output Layer**: Added `LLMFormatter`, `CSVFormatter`.

---

## [1.0.0] — 2026-03-26

### Added
- Initial release with TDN search, Confluence export, and Protheus query support.
