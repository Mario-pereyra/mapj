---
name: mapj
description: >
  CLI tool for AI agents to interact with TOTVS enterprise systems (TDN, Confluence, Protheus ERP).
  Use when: searching TDN documentation, exporting Confluence pages to Markdown (single, recursive,
  or full space), downloading attachments, retrying failed exports, executing SELECT queries on
  Protheus ERP database, or managing Protheus connection profiles (add, list, switch, ping, remove).
  Do NOT use when: writing to Confluence, modifying Protheus data, DML/DDL SQL operations,
  or accessing non-TOTVS systems.
  Triggers: "search TDN", "export confluence", "export to markdown", "export with descendants",
  "export space", "query Protheus", "SELECT Protheus", "switch database", "ping Protheus",
  "list connections", "TOTVS documentation", "look up ADVPL", "export attachments".
compatibility: Requires mapj binary at PATH (Go 1.23+). VPN for internal servers. See sub-skills for specific requirements.
metadata:
  version: 2.0.0
  language: en
  author: Mario Pereira
  license: MIT
  tags:
    - totvs
    - protheus
    - confluence
    - tdn
    - erp
    - agentic
    - cli
  capabilities:
    - search
    - export
    - query
    - connection-management
    - authentication
related:
  - mapj-tdn-search
  - mapj-confluence-export
  - mapj-protheus-query
allowed-tools: Bash
---

# mapj — Agentic CLI for TOTVS Ecosystem

`mapj` connects AI agents to TOTVS enterprise systems. All output is **JSON** with a
consistent envelope. All operations are **read-only** — no data is modified.

> ⚠️ **Documentation mandate:** Any change to commands, flags, or behavior MUST update
> the corresponding sub-skill file, the relevant `docs/` guide, and `CONTRIBUTING.md`.
> Code and docs must always be in sync.

---

## Step 1 — Verify Binary is Available

```bash
mapj --help
mapj auth status
```

If `mapj` is not found: build from source in the project root:
```bash
go build -o mapj ./cmd/mapj
# Then add to PATH
```

---

## Step 2 — Route to the Right Sub-Skill

Load the specific sub-skill for your task. Don't try to do everything from this file.

```
What do you need to do?
│
├─ Search TDN by keyword, label, date, or topic
│   → Load: mapj-tdn-search/SKILL.md
│
├─ Search TDN AND export all found pages
│   → Load: mapj-tdn-search/SKILL.md  (use --export-to flag)
│
├─ List all available TDN spaces
│   → mapj tdn spaces list  (no sub-skill needed)
│
├─ Export a specific Confluence page to Markdown
│   → Load: mapj-confluence-export/SKILL.md
│
├─ Query Protheus ERP database (SELECT only)
│   → Load: mapj-protheus-query/SKILL.md
│
├─ Manage Protheus connection profiles (add/list/switch/ping/remove)
│   → Load: mapj-protheus-query/SKILL.md
│
└─ Check auth status / logout
    → Use auth commands below (no sub-skill needed)
```

---

## Auth Commands (Global — no sub-skill needed)

```bash
# Check all services at once
mapj auth status
# Output: shows TDN, Confluence, Protheus status + active Protheus profile

# Logout a service (removes stored credentials)
mapj auth logout confluence
mapj auth logout protheus
mapj auth logout tdn
```

### First-time auth setup

```
Your Confluence URL?
├─ contains "atlassian.net"
│   → mapj auth login confluence --url URL --username EMAIL --token API_TOKEN
└─ does NOT contain "atlassian.net" (Server/DC like tdninterno.totvs.com)
    → mapj auth login confluence --url URL --token PAT_TOKEN
    → ⚠️ NEVER add --username here → causes 401
```

```bash
# TDN (same server, same PAT as Confluence)
mapj auth login tdn --url https://tdninterno.totvs.com --token YOUR_PAT

# Protheus — register named profile (full setup in mapj-protheus-query/SKILL.md)
mapj protheus connection add TOTALPEC_BIB \
  --server 192.168.99.102 --port 1433 \
  --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB --use
```

---

## Output Schema (All Commands)

### Success
```json
{
  "ok": true,
  "command": "mapj tdn search \"REST API\"",
  "result": { "...": "..." },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T22:00:00Z"
}
```

### Error
```json
{
  "ok": false,
  "command": "mapj protheus query \"INSERT INTO...\"",
  "error": {
    "code": "USAGE_ERROR",
    "message": "query contains forbidden keyword: INSERT",
    "retryable": false
  },
  "schemaVersion": "1.0",
  "timestamp": "2026-03-28T22:00:00Z"
}
```

---

## Exit Codes

| Code | Meaning | Agent Action |
|---|---|---|
| 0 | Success | Parse `result` from JSON |
| 1 | General error | Read `error.message`, fix issue |
| 2 | Usage error (wrong args, forbidden SQL) | Fix the command |
| 3 | Auth error | Run `mapj auth login <service>` |
| 4 | Retryable (timeout, rate limit) | Wait 2s, retry up to 3× |

---

## Global Error Handling Pattern

```bash
# Always check exit code
result=$(mapj tdn search "query")
if [ $? -ne 0 ]; then
  code=$(echo "$result" | jq -r '.error.code')
  retryable=$(echo "$result" | jq -r '.error.retryable')
  # if retryable == true → wait and retry
  # if code == "AUTH_ERROR" → re-authenticate
fi
```

---

## Limitations (Non-Negotiable)

- **Protheus**: SELECT-only. INSERT/UPDATE/DELETE/EXEC/INTO are blocked.
- **Rate limits**: TDN/Confluence may 429 — implement 1s delay between bulk requests.
- **Credentials**: Machine-bound encryption. Use `MAPJ_ENCRYPTION_KEY` env var for CI/CD.
- **VPN**: Internal servers (`192.168.99.x`, `192.168.7.x`) require VPN. Ping shows which VPN.

---

## Sub-Skill Index

| Skill | Location | What it covers |
|---|---|---|
| TDN Search | `mapj-tdn-search/SKILL.md` | CQL search, --since, --ancestor, --label, --export-to pipeline |
| ↳ CQL reference | `mapj-tdn-search/references/cql-reference.md` | All operators, fields, date functions |
| Confluence Export | `mapj-confluence-export/SKILL.md` | Auth types, URL formats, export modes |
| ↳ Auth detail | `mapj-confluence-export/references/auth.md` | 401 fix, Cloud vs Server/DC |
| Protheus Query | `mapj-protheus-query/SKILL.md` | Query workflow, connection management |
| ↳ Known connections | `mapj-protheus-query/references/connections.md` | All 7 profiles + re-register script |
| ↳ Security rules | `mapj-protheus-query/references/security.md` | Blocked keywords + workarounds |
