---
name: preset-worker
description: Worker para implementación de CRUD de presets con parametrización en Go con TDD.
---

# Preset Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Usar para features que implementan:
- Estructuras de datos QueryPreset y ParamDef
- PresetStore con Load/Save operations
- Sistema de detección y validación de parámetros
- Escaping seguro y detección de SQL injection
- Comandos CLI para preset (add, list, run, show, edit, remove, use)
- Tests unitarios y de integración

## Required Skills

None - Este worker implementa código Go directamente.

## Work Procedure

### Fase 1: Test-Driven Development (RED → GREEN)

1. **Escribir test que falle primero**:
   - Crear archivo de test antes de implementación
   - Test debe capturar el comportamiento esperado
   - Verificar que test falla sin implementación
   - Commit del test

2. **Implementar para hacer pasar**:
   - Escribir código mínimo para pasar el test
   - Refactorizar si es necesario
   - Verificar que test pasa
   - Commit de la implementación

### Fase 2: Verificación

1. **Tests unitarios**:
   ```bash
   go test ./internal/preset/... -v -cover
   go test ./internal/cli/... -v -cover
   ```

2. **Build verificación**:
   ```bash
   go build ./...
   go build -o mapj.exe ./cmd/mapj
   ```

3. **Verificación manual**:
   - Ejecutar comandos CLI relevantes
   - Verificar JSON output structure
   - Verificar error messages

### Fase 3: Output Format Consistency

Todos los outputs deben usar el sistema de envelopes existente:

```go
// Éxito
env := output.NewEnvelope(cmd.CommandPath(), result)
fmt.Println(formatter.Format(env))

// Error
env := output.NewErrorEnvelope(cmd.CommandPath(), "ERROR_CODE", message, retryable)
fmt.Println(formatter.Format(env))

// Error con hint
env := output.NewErrorEnvelopeWithHint(cmd.CommandPath(), "ERROR_CODE", message, hint, retryable)
fmt.Println(formatter.Format(env))
```

### Fase 4: SQL Injection Prevention

Para features de escaping y validación:

1. **Implementar detección de patrones**:
   - `; DROP`, `; DELETE`, etc.
   - `OR 1=1`, `OR '1'='1'`
   - `UNION SELECT`
   - `--` comment injection

2. **Implementar escaping**:
   - `'` → `''` para strings
   - CSV → `'a', 'b', 'c'` para listas

3. **Tests exhaustivos de seguridad**:
   - Cada patrón debe tener test
   - Combinaciones de patrones deben detectarse

## Example Handoff

```json
{
  "salientSummary": "Implemented PresetStore with atomic writes, QueryPreset and ParamDef structures, and Load/Save operations. All 15 storage tests pass. File permissions set to 0600, atomic write via temp file + rename.",
  "whatWasImplemented": "Preset Storage Infrastructure: (1) internal/preset/store.go with PresetStore, QueryPreset, ParamDef, PresetFile structures, (2) Load() handles missing file and corrupted JSON gracefully, (3) Save() creates directory, uses atomic write, sets 0600 permissions, (4) JSON formatted with indentation for readability, (5) Active preset tracking persists between sessions.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {
        "command": "go test ./internal/preset/... -v -run TestPresetStore",
        "exitCode": 0,
        "observation": "All 15 storage tests pass including atomic write, permissions, error handling"
      },
      {
        "command": "go test ./internal/preset/... -cover",
        "exitCode": 0,
        "observation": "Coverage: 92.3%"
      },
      {
        "command": "go build -o mapj.exe ./cmd/mapj",
        "exitCode": 0,
        "observation": "Build successful, no errors"
      }
    ],
    "interactiveChecks": [
      {
        "action": "Manual test: Create preset and verify file",
        "observed": "File created at ~/.config/mapj/presets.json with correct JSON structure and 0600 permissions"
      }
    ]
  },
  "tests": {
    "added": [
      {
        "file": "internal/preset/store_test.go",
        "cases": [
          {"name": "TestNewPresetStore", "verifies": "Store creation with correct path"},
          {"name": "TestLoadMissingFile", "verifies": "Graceful handling of missing file"},
          {"name": "TestLoadCorruptedJSON", "verifies": "Error on corrupted JSON"},
          {"name": "TestSaveCreatesDirectory", "verifies": "Directory creation on first save"},
          {"name": "TestSaveAtomicWrite", "verifies": "Temp file + rename pattern"},
          {"name": "TestSavePermissions", "verifies": "0600 file permissions"},
          {"name": "TestQueryPresetStructure", "verifies": "All fields present"},
          {"name": "TestParamDefStructure", "verifies": "All fields present"},
          {"name": "TestActivePresetPersistence", "verifies": "Active preset survives sessions"}
        ]
      }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Tests existentes fallan después de implementación
- Dependencia en código que no existe
- Requerimientos ambiguos o contradictorios
- Error en infraestructura que no puedes resolver (ej. archivo de credenciales corrupto)
- Necesitas modificar archivos off-limits (ver AGENTS.md)
