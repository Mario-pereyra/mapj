# Architecture

**What belongs here:** Descripción de la arquitectura del sistema para referencia de workers.

---

## Estructura del Proyecto mapj_cli

```
mapj_cli/
├── cmd/mapj/main.go          # Entry point (minimal)
├── internal/
│   ├── auth/                 # Autenticación y credential store
│   ├── cli/                  # Comandos CLI (cobra)
│   ├── errors/               # Códigos de error centralizados
│   └── output/               # Formateadores de salida
└── pkg/
    ├── confluence/           # Cliente Confluence API + export
    └── protheus/             # Cliente SQL Server Protheus
```

## Dominios

### Confluence
- Cliente HTTP para API de Confluence
- Export de páginas a Markdown con front matter YAML
- Búsqueda CQL con paginación
- Auto-detección de auth type (Bearer vs Basic)

### TDN
- Reutiliza `pkg/confluence` como cliente
- Autenticación opcional (público funciona sin token)
- Search→Export pipeline con `--export-to`

### Protheus
- Cliente SQL Server para queries read-only
- Multi-perfil con migración v1→v2
- Validación SQL estricta (SELECT-only)
- VPN hints contextuales por IP range

## Sistema de Output

### Formateadores (Strategy Pattern)
- `LLMFormatter`: JSON compacto para agentes
- `HumanFormatter`: JSON pretty con timestamp
- `CSVFormatter`: RFC 4180 compliant
- `TOONFormatter`: Token-efficient (nuevo en gem)

### Envelope
Todas las respuestas siguen: `{ok, command, result, error}`

## Exit Codes

| Code | Significado |
|------|-------------|
| 0 | Success |
| 1 | Error genérico |
| 2 | Usage error |
| 3 | Auth error |
| 4 | Retryable error |
| 5 | Conflict error |

## Diferencias entre Ramas

### main (8.5/10 código)
- Arquitectura más limpia
- Menor complejidad ciclomática
- Sin TOON formatter

### gem (8.1/10 código, mejor tests/docs)
- Introduce TOON formatter
- +339 líneas de tests
- Documentación significativamente mejor
- **3 bugs críticos**: SQL validation, mutex, breaking changes
