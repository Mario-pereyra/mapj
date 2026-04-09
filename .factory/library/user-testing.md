# User Testing Guide for mapj CLI

## Validation Surface

The mapj CLI is tested through its **CLI interface** only. There is no web UI or GUI.

### Testing Tools Required
- **Shell/CLI test runner**: Execute commands and capture output
- **JSON parser**: Validate envelope structure
- **File system inspection**: Verify exported files

### No External Services Required for Unit Tests
Most tests can run without real Confluence/SQL Server connections:
- Output formatters: Use mock data
- SQL validation: Use mock queries
- Parameter interpolation: Use mock databases
- Preset CRUD: Use temp files

### Integration Tests with Real Services
Some tests require real connections:
- TDN search: Requires network to tdn.totvs.com
- Confluence export: Requires valid Confluence instance
- Protheus query: Requires SQL Server connection

**When real services unavailable**: Use mock servers or skip with `--ignored` flag.

## Validation Concurrency

**Max concurrent validators: 1**

This is a CLI tool. Testing is sequential:
1. Run `cargo test` to verify all unit tests pass
2. Run integration tests manually if network available
3. Verify binary builds successfully

Each validation run is lightweight - just compiles and runs test suite.

## Testing Checklist

### Foundation
- [ ] `mapj --version` returns "0.2.0-agentic"
- [ ] `mapj --help` shows all commands
- [ ] `mapj auth --help` shows auth subcommands
- [ ] `mapj tdn --help` shows TDN subcommands
- [ ] `mapj confluence --help` shows Confluence subcommands
- [ ] `mapj protheus --help` shows Protheus subcommands

### Output Formats
- [ ] `--output json` produces compact JSON
- [ ] `--output toon` produces tabular format
- [ ] `--output auto` auto-detects format
- [ ] `--json` takes precedence over `--output`
- [ ] `--verbose` adds schemaVersion and timestamp
- [ ] Error responses always JSON regardless of --output

### Auth Commands
- [ ] `mapj auth status` shows all services
- [ ] Credentials stored encrypted (verify file not plaintext)
- [ ] `mapj auth logout` removes credentials

### TDN Search
- [ ] `mapj tdn search "test"` returns results
- [ ] `mapj tdn spaces list` returns spaces
- [ ] `--max-results` limits results
- [ ] `--check-children` adds childCount

### Confluence Export
- [ ] Numeric page ID exports correctly
- [ ] URL parsing works for Cloud/Server formats
- [ ] Markdown has YAML front matter
- [ ] `--with-descendants` exports tree
- [ ] Progress logged to stderr

### Protheus Query
- [ ] SELECT query executes
- [ ] Invalid prefix rejected
- [ ] `--max-rows` limits results
- [ ] Safety Tripwire for >500 rows
- [ ] `--output-file` writes to file

### Preset System
- [ ] `preset add` creates preset with params
- [ ] `preset list` shows all presets
- [ ] `preset show` displays details
- [ ] `preset run` interpolates and executes
- [ ] SQL injection in params rejected
- [ ] `preset edit` updates fields
- [ ] `preset remove` deletes preset
- [ ] `preset use` sets active preset

## Running Tests

```bash
# Unit tests (no network required)
cargo test

# Integration tests (may need network)
cargo test --test integration

# Specific test
cargo test toon_formatter

# Run with output
cargo test -- --nocapture
```

## Resource Cost Classification

This is a **CLI tool** - testing is CPU and memory lightweight:
- Compilation: ~1-2 GB RAM
- Test execution: ~100-200 MB RAM
- No GPU required

Max concurrent test runs: 1 (sequential testing is standard for CLI)
