# Environment

**What belongs here:** Variables de entorno, dependencias externas, notas de setup.

---

## Dependencias

- **Go 1.21+**: Runtime principal
- **Cobra**: Framework CLI
- **stretchr/testify**: Testing framework
- **go-mssqldb**: Driver SQL Server (Protheus)
- **go-resty**: Cliente HTTP (Confluence)

## Variables de Entorno

| Variable | Descripción |
|----------|-------------|
| `MAPJ_CONFIG_DIR` | Directorio de configuración (default: ~/.config/mapj) |
| `MAPJ_DEBUG` | Habilita debug logging |
| `CONFLUENCE_TOKEN` | Token de autenticación Confluence |
| `CONFLUENCE_URL` | URL base de Confluence |
| `PROTHEUS_DSN` | DSN de conexión SQL Server |

## Credenciales

Almacenadas en `~/.config/mapj/credentials.json` (encriptadas con age).

## No hay servicios externos requeridos para análisis

Esta misión es de análisis documental, no requiere servicios ejecutándose.
