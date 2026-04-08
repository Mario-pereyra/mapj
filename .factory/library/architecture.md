# Architecture

How the preset system works - components, relationships, data flows.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Layer                                │
│  protheus_preset.go - Cobra commands (add/list/run/show/etc)   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                     Business Layer                              │
│  preset/store.go - Persistence (Load/Save)                      │
│  preset/params.go - Detection & Validation                      │
│  preset/escape.go - SQL Security                                │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                     Data Layer                                  │
│  ~/.config/mapj/presets.json - JSON file storage               │
│  QueryPreset, ParamDef, PresetFile structures                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Structures

### QueryPreset
```go
type QueryPreset struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description,omitempty"`
    Query       string                 `json:"query"`
    Connection  string                 `json:"connection,omitempty"`
    MaxRows     int                    `json:"maxRows,omitempty"`
    Parameters  map[string]*ParamDef   `json:"parameters,omitempty"`
    Tags        []string               `json:"tags,omitempty"`
    CreatedAt   string                 `json:"createdAt,omitempty"`
    UpdatedAt   string                 `json:"updatedAt,omitempty"`
}
```

### ParamDef
```go
type ParamDef struct {
    Type        string `json:"type"`                  // string, int, date, datetime, bool, list
    Required    bool   `json:"required"`              // default: true
    Default     string `json:"default,omitempty"`     // fallback value
    Description string `json:"description,omitempty"` // human-readable
    Pattern     string `json:"pattern,omitempty"`     // regex validation
}
```

### PresetFile
```go
type PresetFile struct {
    Presets       map[string]*QueryPreset `json:"presets"`
    ActivePreset  string                  `json:"activePreset,omitempty"`
}
```

---

## Data Flows

### Add Preset Flow
```
CLI add command
    │
    ├── Parse flags (--query, --param-def, --tags, etc.)
    │
    ├── DetectParameters(query) → ["param1", "param2"]
    │
    ├── Build QueryPreset with timestamps
    │
    └── PresetStore.Save()
            │
            ├── Create ~/.config/mapj/ if needed
            ├── Write to temp file
            ├── Rename to presets.json (atomic)
            └── Set permissions 0600
```

### Run Preset Flow
```
CLI run command
    │
    ├── PresetStore.Load() → QueryPreset
    │
    ├── Parse --param flags
    │
    ├── Validate all required params present
    │
    ├── For each param:
    │       ├── ValidateParamType(value, type)
    │       └── DetectSQLInjection(value)
    │
    ├── InterpolateQuery(query, params, paramDefs)
    │       ├── EscapeStringValue() for strings
    │       ├── EscapeListValue() for lists
    │       └── Replace :placeholders with escaped values
    │
    ├── ValidateReadOnly(interpolatedQuery) ← existing check
    │
    └── protheus.Query() → QueryResult
```

---

## Invariants

1. **Atomic Writes**: presets.json is never in partial state
2. **Type Safety**: All parameter values validated before interpolation
3. **SQL Injection Defense**: Multiple detection layers before execution
4. **Connection Priority**: CLI flag > preset saved > active profile > error
5. **Timestamp Integrity**: createdAt set once, updatedAt refreshed on edits
6. **Active Preset Cleanup**: Deleting active preset clears reference

---

## Integration Points

### With Existing Code
- `pkg/protheus/query.go`: `ValidateReadOnly()` called post-interpolation
- `pkg/protheus/query.go`: `Query()` used for execution
- `internal/auth/store.go`: Connection profiles referenced by name
- `internal/output/envelope.go`: All outputs use envelope format

### Storage Location
- Path: `~/.config/mapj/presets.json`
- Permissions: `0600` (owner read/write only)
- Format: JSON with indentation (human-readable)

---

## Security Model

### SQL Injection Prevention (Defense in Depth)

1. **Detection Layer**: Regex patterns detect:
   - `; DROP`, `; DELETE`, etc.
   - `OR 1=1`, `OR '1'='1'`
   - `UNION SELECT`
   - `--` comment injection

2. **Escaping Layer**: Values escaped for SQL Server:
   - `'` → `''` (quote doubling)
   - List values individually escaped

3. **Validation Layer**: Post-interpolation:
   - `ValidateReadOnly()` ensures SELECT/WITH/EXEC prefix
   - Forbidden keywords checked anywhere in query

### Error Handling
- All errors have structured JSON format
- Error codes are UPPER_SNAKE_CASE
- Hints are actionable for agents and humans
- `retryable` flag indicates recovery possible
