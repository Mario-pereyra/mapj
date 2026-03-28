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
│   ├── confluence_retry.go ← `mapj confluence retry-failed` command
│   ├── protheus.go       ← `mapj protheus query` command
│   ├── protheus_connection.go ← `mapj protheus connection *` commands
│   └── tdn.go            ← `mapj tdn search` command
│
├── auth/                 ← Credential storage (encrypted) + login/logout/status
│   ├── store.go          ← ServiceCreds struct, AES-256-GCM encryption, profile helpers
│   ├── login.go          ← Login commands for each service
│   ├── logout.go         ← Logout command
│   └── status.go         ← Auth status display
│
├── errors/               ← Exit code constants
│   └── codes.go
│
└── output/               ← JSON envelope builder + formatters
    ├── envelope.go       ← Envelope, ErrDetail structs + constructors
    └── formatter.go      ← JSON/table formatter

pkg/
├── confluence/           ← All Confluence API logic (no CLI deps)
│   ├── client.go         ← HTTP client with Bearer/Basic auth selection
│   ├── url.go            ← URL parser + 3-level resolution cascade
│   ├── pages.go          ← GET single page, list children
│   ├── spaces.go         ← Space listing
│   ├── search.go         ← CQL search
│   ├── export.go         ← Orchestrator: single/recursive/space export
│   ├── markdown.go       ← HTML→Markdown converter (html-to-markdown/v2)
│   ├── attachments.go    ← Download binary attachments
│   ├── writer.go         ← Write files, manifest.jsonl, errors.jsonl, README index
│   ├── logger.go         ← Structured error logger (export-errors.jsonl)
│   ├── errors.go         ← Confluence-specific error types
│   └── walker.go         ← Recursive page tree walker
│
└── protheus/             ← Protheus SQL Server logic (no CLI deps)
    └── query.go          ← Client, ValidateReadOnly(), Query(), Ping()

skills/                   ← LLM agent skill files (YAML+Markdown)
docs/                     ← User-facing guides
tests/fixtures/           ← Test HTML fixtures
```

### The rule: `pkg/` is pure domain, `internal/cli/` is the adapter

- `pkg/confluence` and `pkg/protheus` have **zero** knowledge of Cobra, CLI flags, or output formats
- `internal/cli/` translates CLI flags → calls `pkg/` → wraps result in JSON envelope
- **Never** put Cobra imports in `pkg/`. **Never** put business logic in `internal/cli/`

---

## 2. Key Data Flows

### Confluence Export Flow

```
User: mapj confluence export <url> --output-path ./docs --with-descendants

internal/cli/confluence.go
  confluenceExportRun()
    │
    ├── getConfluenceClient()           ← reads auth from store, applies auth type
    │
    ├── resolvePageID(url)              ← pkg/confluence/url.go
    │   ├── Try 1: extract ID from URL directly
    │   ├── Try 2: GET /rest/api/content?spaceKey=&title= (CQL)
    │   └── Try 3: HTML scrape (extract ajs-page-id meta tag — WAF bypass)
    │
    └── client.ExportWithOpts(ExportOpts{...})   ← pkg/confluence/export.go
        ├── fetchPage()                 ← pages.go: tries export_view, falls back to storage
        ├── convertToMarkdown()         ← markdown.go: html-to-markdown/v2
        ├── writePageFile()             ← writer.go: saves .md with YAML front matter
        ├── appendToManifest()          ← writer.go: appends to manifest.jsonl
        └── [if --with-descendants]
            └── walker.go: recursively fetch children, repeat for each
```

### Auth Type Resolution (the fixed bug)

```
mapj auth login confluence --url URL --token TOKEN [--username] [--auth-type]
  │
  ├── isCloudURL(url)?                  ← strings.Contains(url, "atlassian.net")
  │   ├── YES → authType = "basic"
  │   └── NO  → authType = "bearer"
  │
  ├── --auth-type flag? → override auto-detect
  │
  ├── authType == "bearer" && username present?
  │   └── WARN: username ignored, clear it
  │
  └── Store: ConfluenceCreds{AuthType: "bearer"|"basic", ...}

getConfluenceClient() in confluence.go:
  ├── AuthType == "basic" → client.SetBasicAuth(username, token)
  └── default ("bearer" or empty legacy) → token already set in NewClient()
```

### Protheus Multi-Profile Model

```
ServiceCreds (stored in credentials.enc)
├── Protheus: *ProtheusCreds          ← v1 legacy, kept for migration only
├── ProtheusProfiles: map[name]*ProtheusProfile   ← v2 named profiles
└── ProtheusActive: string            ← name of active profile

creds.ActiveProtheusProfile()
  ├── If ProtheusActive is set and profile exists → return it
  └── If legacy Protheus v1 exists → return as {Name: "default", ...} for migration
```

---

## 3. How to Add a New Command

### Pattern: `mapj <domain> <action> <args>`

**Step 1:** Create or open `internal/cli/<domain>.go`

```go
var myNewCmd = &cobra.Command{
    Use:   "action <required-arg>",
    Short: "One-line description",
    Args:  cobra.ExactArgs(1),
    RunE:  myNewRun,
}

var myNewFlag string

func init() {
    domainCmd.AddCommand(myNewCmd)
    myNewCmd.Flags().StringVar(&myNewFlag, "my-flag", "default", "Flag description")
}

func myNewRun(cmd *cobra.Command, args []string) error {
    formatter := GetFormatter()
    // ... call pkg/ logic ...
    env := output.NewEnvelope(cmd.CommandPath(), result)
    fmt.Println(formatter.Format(env))
    return nil
}
```

**Step 2:** If you need new domain logic, add it to `pkg/<domain>/`

**Step 3:** Wire errors properly:
```go
// Usage error (wrong args, forbidden operation)
env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", err.Error(), false)

// Auth error
env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", msg, false)

// Retryable network error
env := output.NewErrorEnvelope(cmd.CommandPath(), "QUERY_ERROR", err.Error(), true)
```

**Step 4:** Add a test in `pkg/<domain>/<file>_test.go`

**Step 5:** Update `skills/SKILL.md` commands table and relevant skill file.

---

## 4. Credential Store — How Encryption Works

File: `~/.config/mapj/credentials.enc`

```
Encryption: AES-256-GCM
Key source:
  1. MAPJ_ENCRYPTION_KEY env var (32 bytes exact) — use for testing/CI
  2. deriveMachineKey(): sha256(hostname + username + homeDir) — machine-bound

Format: [12-byte nonce][ciphertext]
Plaintext: JSON-marshaled ServiceCreds struct
```

**ServiceCreds schema (current):**

```go
type ServiceCreds struct {
    TDN        *TDNCreds
    Confluence *ConfluenceCreds  // AuthType: "bearer" | "basic" | ""(legacy=bearer)
    Protheus   *ProtheusCreds    // v1 legacy — DO NOT REMOVE (migration compat)
    ProtheusProfiles map[string]*ProtheusProfile  // v2 named profiles
    ProtheusActive   string       // active profile name
}
```

⚠️ **Migration contract:** If you add a new field to `ServiceCreds`, it must have `omitempty` so old credential files continue to deserialize correctly.

---

## 5. URL Resolution Cascade (url.go)

The `ResolvePageID(url, client)` function tries 3 strategies in order:

```
1. DIRECT — Extract ID from URL patterns:
   - ?pageId=12345
   - /pages/12345/
   - /wiki/spaces/KEY/pages/12345/
   If found → return ID immediately (no API call)

2. API — Parse space key + title from display URL, call:
   GET /rest/api/content?spaceKey=KEY&title=TITLE&type=page
   If found → return ID

3. SCRAPE — HTTP GET the URL as a browser, parse HTML for:
   <meta name="ajs-page-id" content="12345">
   This is the WAF bypass used for tdn.totvs.com which blocks API auth
   but serves public HTML fine.
```

If all 3 fail → return error `PAGE_NOT_FOUND`.

---

## 6. HTML → Markdown Conversion (markdown.go)

Uses `github.com/JohannesKaufmann/html-to-markdown/v2`.

**Confluence-specific handling:**
- Attempts `export_view` representation first (better macro rendering)
- Falls back to `storage` if `export_view` returns empty or fails
- Custom rules for Confluence macros: info/warning/note panels, code blocks, expand macros
- YAML front matter injected after conversion:
  ```yaml
  ---
  page_id: "123"
  title: "..."
  source_url: "..."
  space_key: "..."
  labels: [...]
  updated_at: "..."
  exported_at: "..."
  ---
  ```

---

## 7. Output System (internal/output/)

Every command wraps its result in an `Envelope`:

```go
// Success
output.NewEnvelope(cmd.CommandPath(), anySerializableResult)

// Error
output.NewErrorEnvelope(cmd.CommandPath(), "ERROR_CODE", "message", isRetryable)
```

The `Formatter` selects JSON rendering (default) or table. Currently only JSON is fully implemented. The `-o table` flag exists but defaults to JSON.

---

## 8. Testing

```bash
# Run all tests
go test ./...

# Run specific package
go test ./pkg/confluence/...
go test ./internal/auth/...

# Run with verbose output
go test -v ./pkg/protheus/...

# Run specific test
go test -v ./pkg/confluence/... -run TestValidateReadOnly

# Test with custom encryption key (avoid machine-bound key in CI)
MAPJ_ENCRYPTION_KEY="12345678901234567890123456789012" go test ./internal/auth/...
```

### Test structure

| File | What it tests |
|------|---------------|
| `pkg/confluence/confluence_url_test.go` | URL parsing for all URL formats |
| `pkg/confluence/markdown_test.go` | HTML→Markdown conversion rules |
| `pkg/confluence/export_test.go` | Export orchestration |
| `pkg/protheus/protheus_validation_test.go` | SELECT-only enforcement |
| `internal/auth/auth_store_test.go` | Credential encryption/decryption |
| `internal/output/output_test.go` | Envelope formatting |
| `internal/errors/codes_test.go` | Exit code mapping |

### Adding a test

```go
// Use testify/assert (already a dep)
import "github.com/stretchr/testify/assert"

func TestMyThing(t *testing.T) {
    result, err := MyFunction(input)
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

Test fixtures (HTML samples) go in `tests/fixtures/`.

---

## 9. Dependencies

| Package | Role |
|---------|------|
| `github.com/spf13/cobra` | CLI framework (commands, flags, help) |
| `github.com/JohannesKaufmann/html-to-markdown/v2` | HTML → Markdown conversion |
| `github.com/denisenkom/go-mssqldb` | SQL Server driver for Protheus |
| `github.com/stretchr/testify` | Test assertions |
| `golang.org/x/crypto` | AES-GCM encryption (via stdlib in practice) |

**Adding a dependency:**
```bash
go get github.com/new/package
go mod tidy
```

---

## 10. Build & Release

```bash
# Development run (no binary)
go run ./cmd/mapj <args>

# Build binary (current OS/arch)
go build -o mapj ./cmd/mapj

# Cross-compile for Windows (from Linux/Mac)
GOOS=windows GOARCH=amd64 go build -o mapj.exe ./cmd/mapj

# Cross-compile for Linux (from Windows)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o mapj ./cmd/mapj

# Lint + vet
go vet ./...

# Full check before committing
go build ./... && go test ./... && go vet ./...
```

---

## 11. Known Gotchas & Design Decisions

### ① Auth type stored, not inferred

Before the fix: `if username != "" → Basic Auth`. This broke PAT auth on Server/DC.  
**Now:** `AuthType` field in `ConfluenceCreds` is the source of truth. `getConfluenceClient()` reads it. Auto-detected on login, overridable with `--auth-type`.

### ② `ProtheusCreds` (v1) must stay

The legacy `ProtheusCreds` struct and `Protheus *ProtheusCreds` field in `ServiceCreds` cannot be removed. Users with old credential files would fail to decrypt them if the JSON schema changes. `ActiveProtheusProfile()` transparently migrates v1 → v2.

### ③ WAF bypass for tdn.totvs.com

The public TDN portal blocks API authentication but serves public HTML. Strategy 3 in `ResolvePageID()` GETs the URL as a browser and scrapes `<meta name="ajs-page-id">`. This replicates what the original Python GUI did.

### ④ `export_view` → `storage` fallback

Some Confluence pages return empty body in `export_view` representation (typically pages with unsupported macros). The `fetchPage()` function automatically retries with `storage` format and logs that it happened.

### ⑤ `INTO` is blocked in Protheus queries

The SELECT-only guard blocks the keyword `INTO` to prevent `SELECT INTO #temp`. This is intentional — it also blocks any `INSERT INTO` attempt. If future use requires temp tables, this would need a more surgical check.

### ⑥ CSV format does not escape commas

`protheusResultToCSV()` in `protheus.go` joins fields with commas without quoting. If a field value contains a comma, the CSV will be malformed. Known limitation — use JSON format for fields with potential commas.

### ⑦ `--max-rows` is client-side

The Protheus `--max-rows` flag truncates the result slice **after** the DB returns all rows. It does NOT add `TOP N` to the SQL. For large tables, always use `TOP N` in the SQL itself for true DB-side limiting.

### ⑧ Credentials file is machine-bound by default

Without `MAPJ_ENCRYPTION_KEY`, the key is derived from `sha256(hostname + username + homeDir)`. This means:
- Credentials **cannot be shared** between machines
- If username or hostname changes, credentials become unreadable
- **For CI/CD:** Always set `MAPJ_ENCRYPTION_KEY` explicitly

---

## 12. How to Add a New Service (e.g., `mapj jira`)

1. **Add credentials struct** to `internal/auth/store.go`:
   ```go
   type JiraCreds struct {
       BaseURL  string `json:"baseURL"`
       Username string `json:"username,omitempty"`
       Token    string `json:"token"`
   }
   // Add to ServiceCreds:
   Jira *JiraCreds `json:"jira,omitempty"`
   ```

2. **Add login command** to `internal/auth/login.go` (follow Confluence pattern)

3. **Add status line** to `internal/auth/status.go`

4. **Add logout case** to `internal/auth/logout.go`

5. **Create domain package** `pkg/jira/client.go` + business logic files

6. **Create CLI commands** `internal/cli/jira.go`

7. **Wire to rootCmd** in `internal/cli/root.go`:
   ```go
   rootCmd.AddCommand(tdnCmd, confluenceCmd, protheusCmd, jiraCmd)
   ```

8. **Create skill file** `skills/mapj-jira.md`

9. **Update** `skills/SKILL.md` commands table

---

## 13. File Naming Conventions

| Pattern | Example | Purpose |
|---------|---------|---------|
| `pkg/<domain>/client.go` | `pkg/confluence/client.go` | HTTP client struct |
| `pkg/<domain>/<entity>.go` | `pkg/confluence/pages.go` | Domain entity operations |
| `pkg/<domain>/<file>_test.go` | `pkg/confluence/markdown_test.go` | Tests alongside implementation |
| `internal/cli/<domain>.go` | `internal/cli/confluence.go` | CLI commands for a domain |
| `internal/cli/<domain>_<feature>.go` | `internal/cli/confluence_retry.go` | CLI extension for specific feature |
| `skills/mapj-<domain>.md` | `skills/mapj-confluence-export.md` | Agent skill file |
| `docs/<domain>-guide.md` | `docs/protheus-guide.md` | User guide |
