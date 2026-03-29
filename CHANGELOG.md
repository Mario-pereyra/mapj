# Changelog

All notable changes to `mapj` are documented here.  
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Versioning follows [SemVer](https://semver.org/).

---

## [2.0.1] — 2026-03-29

### Fixed

#### JSON Envelope — Complete Coverage
- **`protheus connection` commands** (`add`, `list`, `use`, `remove`, `show`, `ping`) were emitting
  plain text / table output. All now emit structured JSON envelopes via `GetFormatter()`.
  - `connection list` → `{profiles:[{name,server,port,database,user,active}], count, active}`
  - `connection ping` → `{profile, server, latencyMs, ok}` or `PING_FAILED` with VPN hint
  - `connection use` → `{previous, active, server, port, database}`
  - `connection show` → `{name, server, port, database, user, password(masked), active}`
  - `connection add` → `{name, server, port, database, setActive}`
- **`--format csv`** on `protheus query` was producing JSON to stdout (formatter mismatch).
  Fixed: stdout uses `CSVFormatter` when `--format csv` is active.
- **`auth status/login/logout`** were ignoring the global `-o/--output` flag.
  Fixed: all three now read `-o` from `os.Args` and route through `output.NewFormatter`.

### Added

#### Self-Describing Help Text
Every command now answers 4 LLM-agent questions inline via `--help`:
- What it does and its prerequisite context
- Full JSON output schema (field names, types, and semantics)
- GOTCHAs and known edge cases
- Recommended next steps

Affected files: `root.go`, `tdn.go`, `confluence.go`, `protheus.go`,
`protheus_connection.go`, `auth/login.go`, `auth/status.go`, `auth/logout.go`.

#### Skills Documentation
- `skills/mapj-protheus-query/references/flags.md` — created (was referenced but missing).
  Documents all query flags (`--connection`, `--output-file`, `--format`, `--max-rows`)
  and all `connection` subcommand flags.

---

## [2.0.0] — 2026-03-29

### Breaking Changes

- **Default output mode changed**: `--output` default is now `llm` (compact JSON) instead of `json` (pretty).
  - LLM mode omits `schemaVersion` and `timestamp` from the envelope (~40% fewer tokens)
  - Add `-o json` to restore indented output with metadata
- **Auth commands**: `auth status`, `auth login *`, `auth logout` now emit structured JSON instead of human-readable `fmt.Println` text.
  - Old: `"TDN login successful"` → New: `{"ok":true,"command":"...","result":{"service":"tdn","authenticated":true}}`
  - Old: `"Authentication Status:\n  TDN: ✓ ..."` → New: `{"ok":true,"result":{"tdn":{...},"confluence":{...},"protheus":{...}}}`
- **CSV output**: Protheus `--format csv` now produces RFC 4180-compliant CSV (proper quoting for fields with commas/newlines/quotes). Previously unescaped.

### Added

#### TDN Search v2.1 (`mapj tdn search`)
- `--check-children`: adds `childCount` to each result via concurrent API calls (bounded goroutine pool, max 5 concurrent)
  - `childCount: 0` = leaf page, `N` = has N direct children, `-1` = fetch error
  - ⚠️ `childCount` counts **direct children only** — a page with `childCount: 1` can have 171 total descendants
- `--since`: date filter supporting relative values (`1w`, `2m`, `1y`) and ISO dates (`2024-01-01`)
- `--ancestor <page-id>`: search within a page's descendant tree
- `--labels <l1,l2>`: multiple label filters with AND logic
- `--spaces <PROT,TEC>`: search across multiple spaces
- `--start N`: manual pagination offset
- `--export-to <dir>`: search → export pipeline (find + download in one command)
- `--type`: filter by content type (`page`, `attachment`, etc.)
- Uses `siteSearch` CQL field (searches title, body, labels — broader than `text ~`)
- Results now include: `ancestors`, `labels`, `version`, `lastUpdated`, `lastUpdatedBy`, `childCount` (when `--check-children`)

#### TDN Spaces
- `mapj tdn spaces list`: list all available public TDN spaces

#### Protheus Query
- `--output-file <path>`: write query result to file instead of stdout; stdout receives only a summary `{rows, columns, format, output_file}`
- Useful for large result sets that would saturate LLM context window

#### Output Layer
- `LLMFormatter`: compact JSON, no indent, no `schemaVersion`/`timestamp` — default output
- `HumanFormatter`: indented JSON with `schemaVersion` and `timestamp` — activated with `-o json`
- `CSVFormatter`: RFC 4180-compliant CSV with proper field escaping
- `error.hint`: new field in error envelope with actionable recovery step for agents
- `NewErrorEnvelopeWithHint()`: constructor for errors with recovery hints
- `WriteToFile()`: helper used by `--output-file`

#### Auth
- All auth commands (`login`, `logout`, `status`) now emit structured JSON envelopes
- Bearer auth warning for Server/DC now goes to stderr as structured JSON (not polluting stdout)

### Changed
- `skills/mapj/SKILL.md`: v2.1.0 — Output Modes section added, error.hint documented
- `skills/mapj-tdn-search/SKILL.md`: v2.1.0 — `--check-children` documented, decision tree for `--with-descendants`
- `skills/mapj-protheus-query/SKILL.md`: v3.1.0 — `--output-file` documented, FILE_WRITE_ERROR added

### Fixed
- CSV RFC 4180 compliance: fields containing commas, double-quotes, or newlines now correctly quoted
- Previous CSV implementation used raw `strings.Join` without escaping

---

## [1.0.0] — 2026-03-26

### Added

#### Commands
- `mapj tdn search` — Search TDN documentation with CQL, space and label filters
- `mapj confluence export` — Export Confluence pages to Markdown with YAML front matter
- `mapj confluence export-space` — Export an entire Confluence space
- `mapj confluence retry-failed` — Retry failed exports from `export-errors.jsonl`
- `mapj protheus query` — Execute SELECT queries on Protheus SQL Server
- `mapj protheus connection add/list/use/ping/show/remove` — Named connection profile management
- `mapj auth login/logout/status` — Service authentication

#### Core Features
- AES-256-GCM credential encryption at `~/.config/mapj/credentials.enc`
- JSON output envelope with `ok`, `command`, `result`/`error`, `schemaVersion`, `timestamp`
- Standardized exit codes (0-4) for agent error handling
- `retryable` flag in errors for automatic retry logic
- Auto-detect Confluence auth type (Bearer vs Basic) from URL
- Multi-profile Protheus connections with `--connection` flag for cross-environment queries
- VPN hint in Protheus connection errors (IP-range based)
- `--with-descendants` recursive export (171 pages/s tested)
- `--with-attachments` for binary attachment download
- JSONL error log (`export-errors.jsonl`) + retry workflow
- Manifest file (`manifest.jsonl`) for export inventory

#### Documentation
- `skills/mapj/SKILL.md` — Main agent orchestrator skill
- `skills/mapj-tdn-search/SKILL.md` — TDN search skill
- `skills/mapj-confluence-export/SKILL.md` — Confluence export skill
- `skills/mapj-protheus-query/SKILL.md` — Protheus query skill
- `docs/confluence-export-guide.md` — Human guide
- `docs/protheus-guide.md` — Human guide
- `CONTRIBUTING.md` — Development guide

#### Security
- SELECT-only enforcement: INSERT/UPDATE/DELETE/MERGE/CREATE/ALTER/DROP/TRUNCATE/EXEC/INTO/REPLACE/GRANT/REVOKE/BACKUP/RESTORE all blocked
- `MAPJ_ENCRYPTION_KEY` environment variable for CI/CD credential management

### Fixed
- `Version.By` parsing: Confluence API returns object, not string
- `Body.ExportView` parsing: Confluence API returns object, not string
- User-Agent header to bypass Cloudflare WAF on TDN
- Basic Auth vs Bearer PAT auto-detection and validation
