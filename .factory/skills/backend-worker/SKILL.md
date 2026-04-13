# Backend Worker Skill

skillName: backend-worker

## Overview
Implementa features de logging, health checks, y observabilidad para CLI mapj en Go.

## Procedures

### 1. Setup
1. Lee `mission.md` del missionDir
2. Lee `AGENTS.md` del missionDir  
3. Lee `features.json` para ver la feature a implementar
4. Ejecuta baseline tests: `go test ./...`
5. Verifica que el build funciona: `go build ./...`

### 2. Implementation
Para cada feature:
1. Leer la description de la feature en features.json
2. Crear/modificar archivos necesarios en `internal/` o `cmd/`
3. Implementar código siguiendo convenciones en AGENTS.md
4. Agregar tests unitarios en archivos `*_test.go`
5. Verificar con `go test ./...`

### 3. Verification
- `go build ./...` compila sin errores
- `go test ./...` pasa (o falla gracefully con justificación)
- `go vet ./...` sin errores

## Conventions

### Logging con zap
```go
import "go.uber.org/zap"

var logger *zap.Logger

func init() {
    var err error
    logger, err = zap.NewProduction()
    if err != nil {
        logger = zap.NewNop()
    }
}
defer logger.Sync()

logger.Info("message",
    zap.String("key", value),
    zap.String("traceId", GetTraceID()),
)
```

### Cobra PersistentPreRunE
```go
var rootCmd = &cobra.Command{
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Logging middleware
        return nil
    },
}
```

### Errores enriquecidos
```go
func NewErrorWithTrace(errMsg string) * EnrichedError {
    return &EnrichedError{
        Message: errMsg,
        TraceId: GetTraceID(),
    }
}
```

## Output
Al completar, returns handoff con:
- Feature implementada
- Tests agregados/actualizados
- Commit con cambios
