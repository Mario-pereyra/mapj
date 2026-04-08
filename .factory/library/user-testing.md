# User Testing

Testing surface, required skills, and resource classification for preset system.

---

## Validation Surface

**Primary Surface:** CLI (Terminal)

**Tools:** bash (command execution), jq (JSON validation)

**No browser required** - Pure CLI testing

---

## Required Testing Skills/Tools

| Tool | Purpose |
|------|---------|
| `bash` | Execute CLI commands |
| `jq` | Parse and validate JSON output |
| `diff` | Compare expected vs actual output |

---

## Resource Cost Classification

### Per Validator Instance

| Resource | Usage | Notes |
|----------|-------|-------|
| Memory | ~50 MB | CLI process only |
| CPU | Minimal | No heavy computation |
| Processes | 1 | Single CLI invocation |

### Max Concurrent Validators: 5

CLI testing is lightweight. Each validator only runs CLI commands and parses JSON output. No database connections required for preset management tests.

---

## Test Prerequisites

1. **CLI Built**: `go build -o mapj.exe ./cmd/mapj`
2. **Config Directory**: Will be auto-created at `~/.config/mapj/`
3. **Clean State**: Delete `presets.json` between test runs for isolation

---

## Test Isolation Strategy

### Per-Assertion Isolation

- Each test run uses unique preset names (timestamp-prefixed)
- Tests clean up created presets after execution
- File operations use temp directories where possible

### Parallel Execution

- Safe to run multiple CLI tests in parallel
- Each test uses different preset names
- No shared mutable state (file is written atomically)

---

## Key Test Flows

### Flow 1: Basic CRUD
```
preset add → preset list → preset show → preset remove
```

### Flow 2: Parameter Execution
```
preset add (with params) → preset show → preset run (with values) → verify output
```

### Flow 3: Security Validation
```
preset run (with injection attempt) → verify error → verify no execution
```

### Flow 4: Connection Integration
```
preset add (with connection) → preset run → preset run --connection override
```

---

## Expected Output Format

### Success Response
```json
{
  "ok": true,
  "command": "mapj protheus preset <cmd>",
  "result": { ... }
}
```

### Error Response
```json
{
  "ok": false,
  "command": "mapj protheus preset <cmd>",
  "error": {
    "code": "ERROR_CODE_HERE",
    "message": "Human readable message",
    "hint": "Actionable suggestion",
    "retryable": true
  }
}
```

---

## Cleanup Commands

```bash
# Remove all presets
rm ~/.config/mapj/presets.json

# Or use CLI
mapj protheus preset list --json | jq -r '.presets[].name' | xargs -I{} mapj protheus preset remove {} --force
```

---

## Flow Validator Guidance: CLI

### Isolation Rules

- **Unique Preset Names**: Each flow validator must use preset names prefixed with their group ID (e.g., `group1-test1`, `group2-preset-a`)
- **No Shared Mutable State**: All preset operations use atomic file writes, so concurrent validators won't corrupt data
- **Independent Testing**: Each validator tests its assigned assertions independently

### Boundaries

| Boundary | Rule |
|----------|------|
| Preset Names | Use format: `<group-id>-<test-name>` (e.g., `g1-basic`, `g2-security`) |
| Config Directory | Use default `~/.config/mapj/` - no need to override |
| Clean State | Start by checking/removing conflicting preset names |
| Cleanup | Remove created presets after testing (use `--force` flag) |

### CLI Binary Path

```
D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\mapj.exe
```

### Working Directory

```
D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli
```

### Evidence Collection

- Save command outputs to: `<missionDir>/evidence/<milestone>/<group-id>/`
- Use descriptive file names: `<assertion-id>-output.txt`
- Include both success and failure outputs

### Assertion Testing Pattern

1. Clean preset with same name if exists
2. Run the CLI command under test
3. Capture exit code and output
4. Verify output against assertion specification
5. Record result (pass/fail) with evidence
6. Cleanup created preset
