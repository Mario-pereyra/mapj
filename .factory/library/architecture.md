# Arquitectura de Observabilidad para mapj CLI

## Overview

Sistema de observabilidad diseñado para ser:
- **Escalable**: Arquitectura plugin que permite agregar nuevos comandos con observabilidad mínima
- **Performante**: Logging con zap (zero-allocation), <5ms overhead
- **Extensible**: Preparado para OTel traces y Prometheus metrics

## Componentes

### 1. Logging (`internal/logging/`)
- **zap** como librería de logging
- Output JSON estructurado a stderr
- Niveles: debug, info, warn, error
- traceId (UUID v4) por sesión

### 2. Observability Plugin (`internal/cli/observability.go`)
- **ObservableCommand interface** para opt-in
- **Middleware** PersistentPreRunE para logging automático
- Métricas: counter y histogram

### 3. Health Checks (`internal/cli/health.go`)
- Verificación de conectividad por servicio
- Soporta: TDN, Confluence, Protheus, TDS
- Reporta latencia cuando disponible

## Data Flow

```
User Command
    ↓
rootCmd.PersistentPreRunE (middleware)
    ↓
Generate traceId (UUID v4)
    ↓
Execute command
    ↓
Log entries with traceId
    ↓
Return result/error with traceId
    ↓
Middleware logs completion
```

## Integración con Commands Existentes

| Command | Integración |
|---------|-------------|
| `tdn` | No cambios necesarios |
| `confluence` | Logging automático via middleware |
| `protheus` | Logging automático + health check |
| `auth` | Logging automático |

## Flags

| Flag | Descripción | Default |
|------|-------------|---------|
| `--log-level` | Nivel de logging | info |
| `--observe` | Habilitar observabilidad | off |
| `--service` | Servicio para health check | all |

## Environment Variables

| Variable | Descripción |
|----------|-------------|
| `MAPJ_TRACE_ID` | Override traceId |
| `MAPJ_OBSERVE` | Enable observability (0/1) |
