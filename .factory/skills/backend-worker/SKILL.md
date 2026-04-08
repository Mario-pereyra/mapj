---
name: backend-worker
description: Worker para implementación de fixes y features en Go con TDD
---

# Backend Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Usar para features que requieren:
- Implementación de fixes en código Go
- Modificación de lógica de negocio
- Escritura de tests unitarios
- Refactoring de código existente

## Required Skills

None - Este worker implementa código Go directamente.

## Work Procedure

### Para fixes de bugs:

1. **Escribir test que falle (RED)** - Antes de implementar:
   - Crear o modificar archivo de test
   - Test debe capturar el bug (ej. SQL injection no detectado)
   - Verificar que test falla con código actual
   - Commit del test

2. **Implementar fix (GREEN)** - Hacer que test pase:
   - Modificar código de producción
   - Verificar que test ahora pasa
   - Commit del fix

3. **Verificar tests existentes**:
   - `go test ./... -cover` debe pasar
   - `go test -race ./...` para concurrencia

4. **Verificación manual**:
   - Ejecutar comando CLI relevante
   - Verificar comportamiento esperado

### Para cambios de documentación:

1. **Leer documentación existente**
2. **Identificar cambios requeridos**
3. **Actualizar archivo**
4. **Verificar formato y completitud**

## Example Handoff

```json
{
  "salientSummary": "Fixed SQL validation regression in pkg/protheus/query.go. Added detection of dangerous keywords in any position and semicolon separator. Tests pass: 5 new SQL injection tests, all existing tests pass.",
  "whatWasImplemented": "Fix SQL Validation: (1) Restored dangerous keywords list (INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, TRUNCATE, EXEC, EXECUTE, MERGE, GRANT, REVOKE), (2) Added semicolon detection as statement separator, (3) Added 5 new tests in protheus_validation_test.go for SQL injection scenarios.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {
        "command": "go test ./pkg/protheus/... -v -run TestValidate",
        "exitCode": 0,
        "observation": "All 8 validation tests pass including 5 new SQL injection tests"
      },
      {
        "command": "go test ./... -cover",
        "exitCode": 0,
        "observation": "All packages tested successfully"
      }
    ],
    "interactiveChecks": [
      {
        "action": "Manual SQL injection test: mapj protheus query --sql 'SELECT * FROM users; DROP TABLE users;'",
        "observed": "Error: query contains forbidden keyword: DROP"
      }
    ]
  },
  "tests": {
    "added": [
      {
        "file": "pkg/protheus/protheus_validation_test.go",
        "cases": [
          {"name": "TestSQLInjection_DropTable", "verifies": "DROP TABLE rejected"},
          {"name": "TestSQLInjection_InsertStatement", "verifies": "INSERT rejected"},
          {"name": "TestSQLInjection_SemicolonSeparator", "verifies": "Semicolon as separator detected"},
          {"name": "TestSQLInjection_DeleteInQuery", "verifies": "DELETE in middle rejected"},
          {"name": "TestValidQuery_WithSelect", "verifies": "Valid SELECT accepted"}
        ]
      }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Tests existentes fallan después de fix
- Fix causa side effects inesperados
- Requerimientos ambiguos o contradictorios
- Dependencia en código que no existe
