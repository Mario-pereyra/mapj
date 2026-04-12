# User Testing - Observabilidad CLI

## Validation Surface

**Tool:** CLI terminal (Windows cmd/PowerShell)

## Test Commands

### Logging Tests
```bash
# Basic logging
mapj --log-level=debug tdn search "test" 2>&1 | jq .

# Verify JSON validity
mapj --log-level=info confluence export 235312129 --output-path /tmp/test 2>&1 | jq -e '.'

# traceId verification
mapj tdn search "AdvPL" 2>&1 | grep traceId
```

### Health Check Tests
```bash
# All services
mapj health

# Single service
mapj health --service=tdn
mapj health --service=protheus
mapj health --service=tds

# Invalid service
mapj health --service=invalid 2>&1 | jq '.error.code'
```

### Observability Plugin Tests
```bash
# Enable observability
set MAPJ_OBSERVE=1
mapj tdn search "test" 2>&1 | jq .

# Metrics
mapj observability metrics
```

## Resource Cost

CLI commands son livianos:
- Memory: ~20MB por comando
- CPU: negligible overhead
- Max concurrent: 5 validators

## Preconditions

Para testing completo:
- Credenciales de TDN/Confluence configuradas
- Profile de Protheus activo
- Profile de TDS activo
- Conexión a red TOTVS (VPN si necesario)

Si servicios no disponibles, tests de conectividad retornarán errores esperados.
