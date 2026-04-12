# Backend Worker Skill

## Overview
Implementa features de logging, health checks, y observabilidad para CLI mapj.

## Procedures

### 1. Setup
1. Lee `mission.md` del missionDir
2. Lee `AGENTS.md` del missionDir
3. Ejecuta baseline tests: `go test ./...`
4. Verifica que el build funciona: `go build ./...`

### 2. Implementación
Para cada feature:
1. Crear archivo en directorio apropiado
2. Implementar código siguiendo convenciones en AGENTS.md
3. Agregar tests unitarios
4. Verificar con `go test ./...`

### 3. Verificación
- Todos los tests pasan
- `go vet ./...` sin errores
- `go build ./...` compila sin errores

## Conventions

### Logging con zap
```go
logger, _ := zap.NewProduction()
defer logger.Sync()
logger.Info("message", zap.String("key", value))
```

### Errores enriquecidos
```go
type EnrichedError struct {
    Code      string
    Message   string
    Hint      string
    Retryable bool
    TraceId   string
}
```

### Cobra commands
- Usar `PersistentPreRunE` para middleware
- No modificar commands existentes sin necesidad
- Mantener backward compatibility

## Output
Al completar, returns:
- Handoff con feature implementado
- Tests agregados
- Verificación de build
