# 📘 Guía de Usuario — `mapj protheus`

> **Para:** Mario (usuario del CLI `mapj`)  
> **Versión:** 3.2 — Post-refactorización agentic  
> **Fecha:** Abril 2026

---

## ¿Qué es esto?

`mapj protheus` permite ejecutar consultas SELECT sobre bases de datos SQL Server de Protheus de forma segura y optimizada para agentes IA.

- **Prefix Validation:** Seguridad robusta basada en prefijos (SELECT, WITH, EXEC).
- **Early Cursor Closure:** El flag `--max-rows` aborta la descarga de datos en el servidor, no solo en el cliente.
- **Safety Tripwire:** Si el resultado es muy grande (> 500 filas), se guarda automáticamente en un archivo `.toon` para no inundar el contexto de la IA.
- **Schema Discovery:** Comando integrado para ver la estructura de tablas al instante.

---

## 1. Gestión de perfiles (conexiones)

```bash
mapj protheus connection list                    # ver perfiles (* = activo)
mapj protheus connection use NOMBRE              # cambiar activo
mapj protheus connection ping                    # probar conectividad + hint VPN
mapj protheus connection add ... [--use]         # registrar nuevo
```

---

## 2. Descubrimiento de Tablas (Schema)

Antes de hacer una query, consultá las columnas para evitar errores de nombres:

```bash
mapj protheus schema SA1010
```

Salida (formato TOON):
```yaml
ok: true
result[N]{CHARACTER_MAXIMUM_LENGTH,COLUMN_NAME,DATA_TYPE}:
  6,A1_COD,varchar
  40,A1_NOME,varchar
  ...
```

---

## 3. Ejecutar queries

### Sintaxis

```bash
mapj protheus query "<SQL>" [-o toon|llm] [--max-rows N] [--connection NOMBRE]
```

### Flags principales

| Flag | Default | Descripción |
|------|---------|-------------|
| `-o`, `--output` | `auto` | `auto` (TOON para tablas), `llm` (JSON compacto), `toon` (YAML tabular) |
| `--max-rows` | `10000` | Límite de filas. Cierra el cursor de DB temprano para ahorrar ancho de banda. |
| `--connection` | (activo) | Ejecutar en un perfil específico sin cambiar el activo. |
| `--output-file` | stdout | Redirigir el volcado a un archivo local. |

### Ejemplos

```bash
# Query estándar (formato TOON automático)
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"

# Comparar con PRD
mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD

# Usar CTEs
mapj protheus query "WITH cte AS (SELECT A1_COD FROM SA1010) SELECT * FROM cte"
```

---

## 4. Protección del Context Window (Tripwire)

Si el agente se olvida de poner un `TOP` o un `--output-file` y la consulta trae más de **500 filas**, la CLI hará un **auto-fallback**:

1. Intercepta la salida masiva.
2. La guarda en un archivo temporal (ej: `mapj_overflow_12345.toon`).
3. Avisa por `stderr`.
4. Devuelve un resumen compacto por `stdout` para que el agente sepa dónde están los datos.

---

## 5. Restricciones de Seguridad

Solo se permiten queries que empiecen estrictamente con:
- `SELECT`
- `WITH`
- `EXEC`

Keywords bloqueados: `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `CREATE`, `TRUNCATE`, `MERGE`, `INTO` (incluyendo SELECT INTO).

---

## 6. Errores Comunes

| Error | Solución |
|-------|----------|
| `validation error` | La query no es de solo lectura o tiene un prefijo inválido. |
| `i/o timeout` | Probablemente falta activar la VPN (TOTALPEC o UNION). |
| `login error` | Credenciales incorrectas en el perfil. Usá `connection show` para verificar. |
