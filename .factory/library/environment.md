# Environment

Environment variables, external dependencies, and setup notes.

---

## What belongs here:
Required env vars, external dependencies, platform-specific notes.

## What does NOT belong here:
Service ports/commands (use `.factory/services.yaml`).

---

## Project: mapj CLI

### Language & Runtime
- **Go**: 1.26.1
- **Build target**: Windows (mapj.exe)

### Key Dependencies
```
github.com/spf13/cobra v1.10.2      - CLI framework
github.com/stretchr/testify v1.11.1 - Testing assertions
github.com/denisenkom/go-mssqldb    - SQL Server driver
```

### No External Environment Variables Required
The preset system uses local file storage only.

### Platform Notes
- **Windows**: File permissions may not enforce 0600 strictly
- **Config directory**: `~/.config/mapj/` (or `%APPDATA%\mapj\` on Windows)

### No External Services
- Presets stored in local JSON file
- No network services required for preset management
- Database connection only needed for `preset run` with real queries

---

## Setup for Development

```bash
# Download dependencies
go mod download

# Run tests
go test ./... -cover

# Build
go build -o mapj.exe ./cmd/mapj
```

---

## Directory Structure

```
internal/
├── preset/          # NEW: Preset storage and parameter system
│   ├── store.go     # Load/Save, QueryPreset, ParamDef
│   ├── params.go    # Detection and validation
│   └── escape.go    # SQL escaping and injection detection
├── cli/
│   ├── protheus.go           # Existing: query commands
│   ├── protheus_connection.go # Existing: connection CRUD
│   └── protheus_preset.go    # NEW: preset CRUD commands
└── auth/
    └── store.go     # Existing: credential storage (reference only)

pkg/
└── protheus/
    └── query.go     # Existing: Query execution (use, don't modify)
```
