# CONTRIBUTING — Developer Guide for `mapj`

> **Who this is for:** Anyone (human or LLM) who needs to understand, modify, extend,
> or debug the `mapj` codebase. Start here before touching any code.

---

> ## ⚠️ DOCUMENTATION MANDATE — Read before making any change
>
> **Every code change that affects behavior MUST include updates to:**
>
> | What changed | What to update |
> |---|---|
> | New command or flag | Skill file + `docs/` guide + `SKILL.md` commands table |
> | Changed command behavior | Skill file + `docs/` guide |
> | New/changed data model | `CONTRIBUTING.md` section 4 (Credential Store) or relevant section |
> | New service added | `store.go`, `login.go`, `logout.go`, `status.go`, new skill, new guide, `SKILL.md` |
> | Bug fix that changes output | Skill error table + guide troubleshooting section |
>
> **Never commit code changes without the corresponding documentation update in the same commit.**
> This is non-negotiable. A future agent or developer must be able to trust the docs.

---

## 1. Architecture Overview

```
cmd/mapj/main.go          ← Binary entry point. 3 lines. Just calls cli.Execute().
│
internal/
├── cli/                  ← Cobra command definitions + glue logic
│   ├── root.go           ← rootCmd, Execute(), GetFormatter()
│   ├── auth.go           ← Wires auth package commands to rootCmd
│   ├── confluence.go     ← `mapj confluence export` + `export-space` commands
│   ├── protheus.go       ← `mapj protheus query` + `schema` commands
│   ├── protheus_connection.go ← `mapj protheus connection *` commands
│   └── tdn.go            ← `mapj tdn search` command
│
├── auth/                 ← Credential storage (encrypted) + login/logout/status
│   ├── store.go          ← ServiceCreds struct, AES-256-GCM encryption, machine key derivation
│   ├── login.go          ← Login commands for each service
│   ├── logout.go         ← Logout command
│   └── status.go         ← Auth status display
│
├── errors/               ← Strongly typed errors + ExitCoder interface
│   └── codes.go
│
└── output/               ← JSON/TOON envelope builder + formatters
    ├── envelope.go       ← Envelope, ErrDetail structs + constructors
    ├── formatter.go      ← LLM (compact JSON) and Auto-detection factory
    └── toon_formatter.go ← TOON (Tabular Object Notation) logic

pkg/
├── confluence/           ← All Confluence API logic (no CLI deps)
│   ├── client.go         ← HTTP client with Auto-Healing (Exponential Backoff)
│   ├── url.go            ← URL parser + 3-level resolution cascade
│   ├── pages.go          ← GET single page, list children
│   ├── spaces.go         ← Space listing
│   ├── search.go         ← CQL search with Auto-Pagination
│   ├── search_pipeline.go ← Enrichment (childCount) and search-to-export orchestration
│   ├── export.go         ← Orchestrator: Concurrent Worker Pool (10 workers)
│   ├── markdown.go       ← Singleton Markdown converter (html-to-markdown/v2)
│   ├── attachments.go    ← Download binary attachments
│   ├── writer.go         ← Write files, manifest.jsonl, README index
│   ├── logger.go         ← Stderr progress and summary logger
│   └── errors.go         ← Confluence-specific error types
│
└── protheus/             ← Protheus SQL Server logic (no CLI deps)
    └── query.go          ← Client, Prefix-based Validation, Query (with Early Cursor Closure), Ping()

skills/                   ← LLM agent skill files (YAML+Markdown)
docs/                     ← User-facing guides
tests/fixtures/           ← Test HTML fixtures
```

### The rule: `pkg/` is pure domain, `internal/cli/` is the adapter

- `pkg/confluence` and `pkg/protheus` have **zero** knowledge of Cobra, CLI flags, or output formats
- `internal/cli/` translates CLI flags → calls `pkg/` → wraps result in JSON/TOON envelope
- **Never** put Cobra imports in `pkg/`. **Never** put business logic in `internal/cli/`

---

## 2. Key Data Flows

### Confluence Export Flow (Concurrent)

```
User: mapj confluence export <url> --output-path ./docs --with-descendants

internal/cli/confluence.go
  confluenceExportRun()
    │
    ├── getConfluenceClient()           ← reads auth from store
    │
    ├── resolvePageID(url)              ← pkg/confluence/url.go
    │
    └── client.ExportWithDescendants()   ← pkg/confluence/export.go
        ├── GetDescendants()            ← fetches full tree list
        └── ExportPages()               ← Worker Pool (10 goroutines)
            ├── fetchPage()             ← client.go (with Auto-Healing)
            ├── convertToMarkdown()     ← markdown.go (Singleton)
            └── writePageFile()         ← writer.go
```

### SQL Safety Tripwire (Protheus)

```
User: mapj protheus query "SELECT * FROM HugeTable"

internal/cli/protheus.go
  protheusQueryRun()
    │
    ├── client.Query(..., maxRows)      ← pkg/protheus/query.go
    │   ├── ValidateReadOnly()          ← Prefix check (SELECT/WITH/EXEC)
    │   └── rows.Next() loop            ← breaks at maxRows, CLOSES CURSOR EARLY
    │
    ├── If result.Count > 500 && --output-file == "":
    │   ├── Divert output to temp .toon file
    │   └── Warn via stderr
    │
    └── Format summary/result to stdout
```

---

## 3. Exit Codes (internal/errors/codes.go)

The CLI uses an `ExitCoder` interface for machine-readable errors:

| Code | Type | Meaning |
|---|---|---|
| 0 | Success | Operation completed |
| 1 | GeneralError | Internal error or unexpected failure |
| 2 | UsageError | Bad arguments, syntax, or forbidden SQL |
| 3 | AuthError | Missing or invalid credentials |
| 4 | RetryableError | Rate limit (429) or Server error (50x) |

---

## 4. Credential Store

File: `~/.config/mapj/credentials.enc` (AES-256-GCM)

**ServiceCreds schema:**

```go
type ServiceCreds struct {
    TDN        *TDNCreds
    Confluence *ConfluenceCreds
    ProtheusProfiles map[string]*ProtheusProfile
    ProtheusActive   string
}
```

---

## 5. Design Decisions

### ① Prefix-based SQL Validation
Instead of regex blacklists, we use a whitelist of allowed prefixes (`SELECT`, `WITH`, `EXEC`). This prevents bypasses via comments or unusual syntax.

### ② Early Cursor Closure
We stop reading from the DB driver as soon as the `--max-rows` limit is reached. Closing the rows cursor immediately signals the SQL Server to stop processing the query, saving significant resources.

### ③ TOON (Tabular Object Notation)
We use a YAML-like tabular format for arrays of objects. It anchors column names at the top and lists data below, reducing token usage by ~40% compared to JSON.

### ④ Auto-Healing Client
The HTTP client internally handles retries for transient errors. Agents shouldn't have to manage network instability.

---

## 6. Build & Test

```bash
# Full check before committing
go build ./... && go test ./... -race && go vet ./...
```
