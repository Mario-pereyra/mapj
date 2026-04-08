# Análisis de Parametrización de Queries en Presets de `mapj protheus`

**Fecha:** 2026-04-08  
**Autor:** Factory Droid (Worker Analysis)  
**Estado:** Análisis completado  
**Contexto:** Extensión del análisis `protheus-presets-analysis.md` para manejo de parámetros

---

## 1. Resumen Ejecutivo

**Recomendación principal:** Implementar **Opción A (Placeholders con sintaxis `:param`)** combinada con **interpolación segura con escaping**.

**Justificación:**
1. **Consistencia con la industria:** `psql -v` usa esta sintaxis desde hace décadas
2. **Agent-friendly:** Comando completo sin interacción, parseable, documentable
3. **Seguridad:** Permite escaping de valores antes de la interpolación
4. **Type-safety:** Los tipos se declaran en el preset y se validan antes de ejecutar

**Sintaxis propuesta:**
```bash
# Definición del preset con placeholders
mapj protheus preset add busca-cliente \
  --query "SELECT * FROM SA1010 WHERE A1_COD = :cliente AND A1_LOJA = :loja" \
  --param-def cliente:string \
  --param-def loja:string

# Ejecución con valores
mapj protheus preset run busca-cliente \
  --param cliente=000001 \
  --param loja=01
```

---

## 2. Análisis de Opciones de Parametrización

### 2.1 Opción A: Placeholders con sintaxis `:param` (psql-style)

**Sintaxis en preset:**
```sql
SELECT * FROM SA1010 WHERE A1_COD = :cliente AND A1_LOJA = :loja
```

**Comandos CLI:**
```bash
mapj protheus preset add busca-cliente \
  --query "SELECT * FROM SA1010 WHERE A1_COD = :cliente AND A1_LOJA = :loja"

mapj protheus preset run busca-cliente --param cliente=000001 --param loja=01
```

| Criterio | Evaluación |
|----------|------------|
| **Agent-friendly** | ✅ Comando completo, no requiere interacción |
| **Parseable** | ✅ Sintaxis clara `--param key=value` |
| **Documentable** | ✅ `preset show` lista los parámetros detectados |
| **Tipado** | ⚠️ Requiere `--param-def` adicional o auto-detección |
| **Industria** | ✅ Compatible con patrón psql `-v` |
| **SQL Injection** | ⚠️ Requiere escaping explícito |

**Veredicto:** ✅ **RECOMENDADO** - Mejor balance entre simplicidad y seguridad

---

### 2.2 Opción B: Template strings Go-style (`{{.param}}`)

**Sintaxis en preset:**
```sql
SELECT * FROM SA1010 WHERE A1_COD = '{{.cliente}}' AND A1_LOJA = '{{.loja}}'
```

**Comandos CLI:**
```bash
mapj protheus preset add busca-cliente \
  --query "SELECT * FROM SA1010 WHERE A1_COD = '{{.cliente}}' AND A1_LOJA = '{{.loja}}'"

mapj protheus preset run busca-cliente --var cliente=000001 --var loja=01
```

| Criterio | Evaluación |
|----------|------------|
| **Agent-friendly** | ✅ Comando completo |
| **Parseable** | ✅ Sintaxis clara |
| **Documentable** | ✅ Detectable via regex |
| **Tipado** | ✅ Compatible con Go templates (funciones de tipo) |
| **Industria** | ✅ Patrón usado en kubectl, Helm |
| **SQL Injection** | ❌ Los valores se interpolan como strings literales |
| **Complejidad** | ❌ Requiere parser de templates Go completo |

**Veredicto:** ⚠️ **NO RECOMENDADO** - Complejidad innecesaria y riesgo de SQL injection

**Problema crítico:** La sintaxis `{{.cliente}}` requiere que el usuario incluya las comillas en el template, lo que es propenso a errores y hace difícil el escaping.

---

### 2.3 Opción C: Interpolación con `$VAR` (env-style)

**Sintaxis en preset:**
```sql
SELECT * FROM SA1010 WHERE A1_COD = '$CLIENTE' AND A1_LOJA = '$LOJA'
```

**Comandos CLI:**
```bash
mapj protheus preset add busca-cliente \
  --query "SELECT * FROM SA1010 WHERE A1_COD = '\$CLIENTE' AND A1_LOJA = '\$LOJA'"

mapj protheus preset run busca-cliente --env CLIENTE=000001 LOJA=01
```

| Criterio | Evaluación |
|----------|------------|
| **Agent-friendly** | ✅ Comando completo |
| **Parseable** | ⚠️ Conflicto con shell variable expansion |
| **Documentable** | ✅ Convención de nombres UPPER_CASE |
| **Tipado** | ❌ Sin tipado nativo |
| **Industria** | ⚠️ Estándar para env vars, no para SQL |
| **SQL Injection** | ❌ Similar a Opción B |
| **Escaping** | ❌ Problemas con shell interpolation |

**Veredicto:** ❌ **NO RECOMENDADO** - Conflicto con shell variables, ambiguo

---

### 2.4 Opción D: Input interactivo

**Comandos CLI:**
```bash
mapj protheus preset run busca-cliente
# CLI prompt: "Enter value for cliente: "
# CLI prompt: "Enter value for loja: "
```

| Criterio | Evaluación |
|----------|------------|
| **Agent-friendly** | ❌ **CRÍTICO: NO compatible con LLMs** |
| **Parseable** | ❌ Requiere interacción |
| **Documentable** | ✅ Prompt muestra nombre del parámetro |
| **Tipado** | ⚠️ Difícil de validar en prompt |
| **Industria** | ✅ Usado en algunos CLIs |
| **SQL Injection** | ⚠️ Requiere validación post-input |

**Veredicto:** ❌ **NO RECOMENDADO** - Incompatible con el requisito agent-friendly

**Nota:** Podría implementarse como fallback cuando un LLM no proporciona los valores requeridos, pero no como mecanismo principal.

---

### 2.5 Opción E: URL-style query params

**Sintaxis:**
```bash
mapj protheus preset run busca-cliente?cliente=000001&loja=01
```

| Criterio | Evaluación |
|----------|------------|
| **Agent-friendly** | ✅ Comando completo |
| **Parseable** | ✅ Sintaxis URL estándar |
| **Documentable** | ⚠️ Menos intuitivo para CLI |
| **Tipado** | ❌ Sin tipado nativo |
| **Industria** | ⚠️ Patrón web, no CLI |
| **SQL Injection** | ❌ Sin protección nativa |
| **Escaping** | ⚠️ Problemas con caracteres especiales |

**Veredicto:** ❌ **NO RECOMENDADO** - Sintaxis no idiomática para CLI

---

## 3. Análisis desde Perspectiva Agent-Friendly

### 3.1 Requisitos para LLMs

| Requisito | Descripción | Opción A | Opción B |
|-----------|-------------|----------|----------|
| **Comando completo** | No requiere prompts interactivos | ✅ | ✅ |
| **Parseable sin ambigüedad** | Output JSON estructurado | ✅ | ✅ |
| **Parámetros documentados** | `preset show` lista params | ✅ | ✅ |
| **Validación previa** | Error si falta param antes de ejecutar | ✅ | ✅ |
| **Tipado explícito** | Previene errores de tipo | ✅ con `--param-def` | ⚠️ implícito |

### 3.2 Flujo de Invocación por un LLM

**Escenario:** Un agente quiere ejecutar un preset que requiere parámetros.

```bash
# Paso 1: Descubrir qué parámetros requiere el preset
mapj protheus preset show busca-cliente
```

**Output esperado:**
```json
{
  "ok": true,
  "command": "mapj protheus preset show",
  "result": {
    "name": "busca-cliente",
    "query": "SELECT * FROM SA1010 WHERE A1_COD = :cliente AND A1_LOJA = :loja",
    "parameters": [
      {"name": "cliente", "type": "string", "required": true},
      {"name": "loja", "type": "string", "required": true}
    ],
    "connection": null
  }
}
```

```bash
# Paso 2: Ejecutar con los parámetros
mapj protheus preset run busca-cliente --param cliente=000001 --param loja=01
```

**Output esperado:**
```json
{
  "ok": true,
  "command": "mapj protheus preset run",
  "result": {
    "columns": ["A1_COD", "A1_NOME", "A1_LOJA"],
    "rows": [["000001", "CLIENTE TESTE", "01"]],
    "count": 1
  }
}
```

### 3.3 Manejo de Errores Agent-Friendly

**Error: Parámetro faltante**
```json
{
  "ok": false,
  "command": "mapj protheus preset run",
  "error": {
    "code": "MISSING_PARAMETER",
    "message": "preset 'busca-cliente' requires parameter 'cliente' which was not provided",
    "retryable": true,
    "hint": "Run 'mapj protheus preset show busca-cliente' to see required parameters"
  }
}
```

**Error: Parámetro con valor inválido**
```json
{
  "ok": false,
  "command": "mapj protheus preset run",
  "error": {
    "code": "INVALID_PARAMETER_VALUE",
    "message": "parameter 'fecha_inicio' expected date format YYYY-MM-DD, got 'ayer'",
    "retryable": true,
    "hint": "Use format: --param fecha_inicio=2026-04-01"
  }
}
```

---

## 4. Diseño de Flags CLI para Parámetros

### 4.1 Flags para Definición de Preset

```bash
mapj protheus preset add <name> \
  --query "SQL con :placeholders" \
  --param-def <name>:<type>[:<default>] \
  --param-desc <name>="descripción"
```

**Tipos soportados:**

| Tipo | Formato de valor | Ejemplo |
|------|------------------|---------|
| `string` | Cualquier string | `--param cliente=000001` |
| `int` | Entero válido | `--param limite=100` |
| `date` | ISO 8601 (YYYY-MM-DD) | `--param fecha=2026-04-01` |
| `datetime` | ISO 8601 extendido | `--param timestamp=2026-04-01T10:30:00` |
| `bool` | true/false, 1/0 | `--param activo=true` |
| `list` | Valores separados por coma | `--param codigos=001,002,003` |

**Ejemplo completo:**
```bash
mapj protheus preset add pedidos-rango \
  --query "SELECT * FROM SC5010 WHERE C5_EMISSAO BETWEEN :fecha_inicio AND :fecha_fin" \
  --param-def fecha_inicio:date \
  --param-def fecha_fin:date \
  --param-desc fecha_inicio="Fecha inicial del rango" \
  --param-desc fecha_fin="Fecha final del rango" \
  --description "Pedidos en un rango de fechas"
```

### 4.2 Flags para Ejecución de Preset

```bash
mapj protheus preset run <name> \
  --param <name>=<value> \
  [--connection <profile>] \
  [--max-rows <n>] \
  [--output-file <path>]
```

**Múltiples parámetros:**
```bash
mapj protheus preset run busca-cliente \
  --param cliente=000001 \
  --param loja=01
```

**Formato alternativo (JSON):**
```bash
mapj protheus preset run busca-cliente \
  --params-json '{"cliente":"000001","loja":"01"}'
```

**Ventajas de `--params-json`:**
- Útil cuando hay muchos parámetros
- Fácil de generar programáticamente
- Compatible con output de otros comandos

### 4.3 Validación de Parámetros

**Antes de la ejecución:**
1. Detectar todos los `:param` en la query
2. Verificar que todos los parámetros requeridos tienen valor
3. Validar tipos de datos según `--param-def`
4. Aplicar escaping de seguridad

**Output de validación fallida:**
```json
{
  "ok": false,
  "error": {
    "code": "PARAMETER_VALIDATION_ERROR",
    "message": "parameter validation failed",
    "details": [
      {"param": "cliente", "error": "required parameter not provided"},
      {"param": "fecha_inicio", "error": "expected date, got 'invalid'"}
    ]
  }
}
```

---

## 5. Metadatos de Preset para Documentar Parámetros

### 5.1 Estructura de Datos Extendida

```go
type QueryPreset struct {
    Name        string                  `json:"name"`
    Description string                  `json:"description,omitempty"`
    Query       string                  `json:"query"`
    Connection  string                  `json:"connection,omitempty"`
    MaxRows     int                     `json:"maxRows,omitempty"`
    CreatedAt   string                  `json:"createdAt,omitempty"`
    UpdatedAt   string                  `json:"updatedAt,omitempty"`
    Tags        []string                `json:"tags,omitempty"`
    
    // NUEVOS CAMPOS para parametrización
    Parameters  map[string]*ParamDef    `json:"parameters,omitempty"`
}

type ParamDef struct {
    Type        string `json:"type"`                  // string, int, date, datetime, bool, list
    Required    bool   `json:"required"`              // default: true
    Default     string `json:"default,omitempty"`     // valor por defecto si no se provee
    Description string `json:"description,omitempty"` // descripción humana
    Pattern     string `json:"pattern,omitempty"`     // regex de validación (opcional)
}
```

### 5.2 Ejemplo de Archivo presets.json

```json
{
  "presets": {
    "busca-cliente": {
      "name": "busca-cliente",
      "description": "Busca un cliente por código y loja",
      "query": "SELECT * FROM SA1010 WHERE A1_COD = :cliente AND A1_LOJA = :loja",
      "parameters": {
        "cliente": {
          "type": "string",
          "required": true,
          "description": "Código del cliente (A1_COD)"
        },
        "loja": {
          "type": "string",
          "required": true,
          "description": "Loja/sucursal del cliente (A1_LOJA)"
        }
      },
      "createdAt": "2026-04-08T10:00:00Z"
    },
    "pedidos-rango": {
      "name": "pedidos-rango",
      "description": "Pedidos en un rango de fechas",
      "query": "SELECT * FROM SC5010 WHERE C5_EMISSAO BETWEEN :fecha_inicio AND :fecha_fin",
      "parameters": {
        "fecha_inicio": {
          "type": "date",
          "required": true,
          "description": "Fecha inicial (YYYY-MM-DD)"
        },
        "fecha_fin": {
          "type": "date",
          "required": true,
          "description": "Fecha final (YYYY-MM-DD)"
        }
      },
      "connection": "TOTALPEC_PRD"
    },
    "clientes-por-estado": {
      "name": "clientes-por-estado",
      "description": "Clientes de estados específicos",
      "query": "SELECT * FROM SA1010 WHERE A1_EST IN (:estados)",
      "parameters": {
        "estados": {
          "type": "list",
          "required": false,
          "default": "SP,RJ",
          "description": "Lista de estados separados por coma"
        }
      }
    }
  },
  "activePreset": "busca-cliente"
}
```

### 5.3 Detección Automática de Parámetros

Si el usuario no define `--param-def`, el sistema debe:

1. **Detectar placeholders:** Regex `:[a-zA-Z_][a-zA-Z0-9_]*`
2. **Inferir tipo básico:** Todos como `string` por defecto
3. **Marcar como requeridos:** Todos los detectados son required=true

```go
// placeholderRegex detecta :param en queries SQL
var placeholderRegex = regexp.MustCompile(`:[a-zA-Z_][a-zA-Z0-9_]*`)

func DetectParameters(query string) []string {
    matches := placeholderRegex.FindAllString(query, -1)
    params := make([]string, len(matches))
    for i, m := range matches {
        params[i] = strings.TrimPrefix(m, ":")
    }
    return params
}
```

---

## 6. Funciones de Validación y Escaping

### 6.1 Principio de Seguridad

**⚠️ IMPORTANTE:** OWASP desalienta el "escaping de todos los inputs" como defensa primaria. Sin embargo, para una CLI que interpola valores en SQL, el escaping es **una capa de defensa** combinada con:

1. **Validación estricta de tipos**
2. **Whitelist de caracteres permitidos**
3. **Prevención de SQL injection patterns**
4. **Audit logging de queries ejecutadas**

### 6.2 Función de Interpolación Segura

```go
package preset

import (
    "fmt"
    "regexp"
    "strings"
)

// SQLValueEscaper escapa valores para interpolación segura en SQL
// NOTA: Esto es una capa de defensa, NO una solución completa de SQL injection
type SQLValueEscaper struct {
    // dangerousPatterns detecta intentos de SQL injection
    dangerousPatterns []*regexp.Regexp
}

func NewSQLValueEscaper() *SQLValueEscaper {
    return &SQLValueEscaper{
        dangerousPatterns: []*regexp.Regexp{
            regexp.MustCompile(`(?i)(\bOR\b|\bAND\b)\s+['"]?\d+['"]?\s*=\s*['"]?\d+['"]?`), // OR 1=1
            regexp.MustCompile(`(?i);\s*(DROP|DELETE|UPDATE|INSERT|EXEC)`),                  // ; DROP
            regexp.MustCompile(`(?i)UNION\s+SELECT`),                                         // UNION SELECT
            regexp.MustCompile(`(?i)--\s*$`),                                                // comment injection
            regexp.MustCompile(`(?i)/\*.*\*/`),                                              // block comment
            regexp.MustCompile(`'(\s*OR\s*'|')`),                                            // string escape injection
        },
    }
}

// EscapeValue aplica escaping y validación a un valor de parámetro
func (e *SQLValueEscaper) EscapeValue(value string, paramType string) (string, error) {
    // 1. Detectar patrones peligrosos
    for _, pattern := range e.dangerousPatterns {
        if pattern.MatchString(value) {
            return "", fmt.Errorf("potential SQL injection detected in parameter value")
        }
    }

    // 2. Aplicar escaping según el tipo
    switch paramType {
    case "string":
        return escapeStringValue(value), nil
    case "int":
        if !isValidInt(value) {
            return "", fmt.Errorf("invalid integer value: %s", value)
        }
        return value, nil
    case "date":
        if !isValidDate(value) {
            return "", fmt.Errorf("invalid date format, expected YYYY-MM-DD: %s", value)
        }
        return fmt.Sprintf("'%s'", value), nil
    case "datetime":
        if !isValidDateTime(value) {
            return "", fmt.Errorf("invalid datetime format, expected ISO 8601: %s", value)
        }
        return fmt.Sprintf("'%s'", value), nil
    case "bool":
        return escapeBoolValue(value), nil
    case "list":
        return escapeListValue(value), nil
    default:
        return escapeStringValue(value), nil
    }
}

// escapeStringValue escapa comillas y caracteres especiales
func escapeStringValue(value string) string {
    // Duplicar comillas simples (SQL Server standard)
    escaped := strings.ReplaceAll(value, "'", "''")
    return fmt.Sprintf("'%s'", escaped)
}

// escapeListValue convierte una lista CSV a formato IN clause
func escapeListValue(value string) string {
    items := strings.Split(value, ",")
    escaped := make([]string, len(items))
    for i, item := range items {
        item = strings.TrimSpace(item)
        escaped[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''"))
    }
    return strings.Join(escaped, ", ")
}

func isValidInt(value string) bool {
    matched, _ := regexp.MatchString(`^-?\d+$`, value)
    return matched
}

func isValidDate(value string) bool {
    matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value)
    return matched
}

func isValidDateTime(value string) bool {
    matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, value)
    return matched
}

func escapeBoolValue(value string) string {
    lower := strings.ToLower(value)
    if lower == "true" || lower == "1" || lower == "yes" {
        return "1"
    }
    return "0"
}
```

### 6.3 Función de Interpolación de Query

```go
// InterpolateQuery reemplaza placeholders con valores escapados
func InterpolateQuery(query string, params map[string]string, paramDefs map[string]*ParamDef) (string, error) {
    escaper := NewSQLValueEscaper()
    
    // Detectar placeholders no provistos
    detectedParams := DetectParameters(query)
    for _, p := range detectedParams {
        if _, ok := params[p]; !ok {
            // Verificar si tiene default
            if def, hasDef := paramDefs[p]; hasDef && def.Default != "" {
                params[p] = def.Default
            } else {
                return "", fmt.Errorf("missing required parameter: %s", p)
            }
        }
    }
    
    // Interpolar cada parámetro
    result := query
    for name, value := range params {
        placeholder := ":" + name
        
        // Determinar tipo
        paramType := "string"
        if def, ok := paramDefs[name]; ok {
            paramType = def.Type
        }
        
        // Escapar valor
        escapedValue, err := escaper.EscapeValue(value, paramType)
        if err != nil {
            return "", fmt.Errorf("parameter '%s': %w", name, err)
        }
        
        // Reemplazar placeholder
        result = strings.ReplaceAll(result, placeholder, escapedValue)
    }
    
    return result, nil
}
```

---

## 7. Edge Cases y Soluciones

### 7.1 Parámetros Opcionales (WHERE con condiciones dinámicas)

**Problema:** Query con filtro opcional
```sql
SELECT * FROM SA1010 
WHERE D_E_L_E_T_ = '' 
  AND (:estado IS NULL OR A1_EST = :estado)
```

**Solución 1: Default especial `NULL`**
```bash
mapj protheus preset run clientes \
  --param estado=NULL  # Genera "WHERE (:estado IS NULL OR A1_EST = NULL)"
```

**Solución 2: Parámetro con default vacío**
```bash
mapj protheus preset add clientes \
  --query "SELECT * FROM SA1010 WHERE D_E_L_E_T_ = '' AND (:estado = '' OR A1_EST = :estado)" \
  --param-def estado:string:default=""
```

**Solución 3 (RECOMENDADA): Templates condicionales**

Para casos complejos, permitir sintaxis condicional:
```sql
SELECT * FROM SA1010 
WHERE D_E_L_E_T_ = '' 
{{if .estado}}AND A1_EST = :estado{{end}}
```

Esto requiere implementar un subset de Go templates para condicionales.

### 7.2 Listas de Valores (IN clause)

**Problema:** `WHERE A1_COD IN ('001', '002', '003')`

**Solución con tipo `list`:**
```bash
mapj protheus preset add clientes-lista \
  --query "SELECT * FROM SA1010 WHERE A1_COD IN (:codigos)" \
  --param-def codigos:list

mapj protheus preset run clientes-lista --param codigos=001,002,003
```

**Query interpolada:**
```sql
SELECT * FROM SA1010 WHERE A1_COD IN ('001', '002', '003')
```

### 7.3 Fechas con Formatos Específicos

**Problema:** Protheus usa formatos de fecha específicos (YYMMDD, YYYYMMDD)

**Solución: Tipo de dato personalizado**
```go
type DateFormat string

const (
    DateFormatISO     DateFormat = "2006-01-02"      // YYYY-MM-DD
    DateFormatProtheus DateFormat = "20060102"       // YYYYMMDD
    DateFormatShort   DateFormat = "060102"         // YYMMDD
)

func FormatDate(value string, format DateFormat) (string, error) {
    t, err := time.Parse("2006-01-02", value)
    if err != nil {
        return "", err
    }
    return t.Format(string(format)), nil
}
```

**CLI:**
```bash
mapj protheus preset add pedidos-protheus \
  --query "SELECT * FROM SC5010 WHERE C5_EMISSAO = :fecha" \
  --param-def fecha:date:format=protheus

mapj protheus preset run pedidos-protheus --param fecha=2026-04-01
# Interpola como: WHERE C5_EMISSAO = '20260401'
```

### 7.4 Caracteres Especiales y Escaping

**Problema:** Valor contiene `'` o caracteres especiales

**Ejemplo:** Cliente con nombre `O'Brian`

```bash
mapj protheus preset run busca-nombre --param nombre=O'Brian
```

**Solución:** El escaper duplica las comillas:
```sql
WHERE A1_NOME = 'O''Brian'
```

### 7.5 SQL Injection a través de Parámetros

**Intento de inyección:**
```bash
mapj protheus preset run busca-cliente \
  --param cliente="000001'; DROP TABLE SA1010; --"
```

**Defensa multicapa:**

1. **Detección de patrones peligrosos:**
   - `;` seguido de `DROP`, `DELETE`, etc.
   - `OR 1=1`
   - `UNION SELECT`
   - `--` al final

2. **Escaping de comillas:**
   ```sql
   WHERE A1_COD = '000001''; DROP TABLE SA1010; --'
   ```
   El valor completo se trata como string literal.

3. **Validación existente (`ValidateReadOnly`):**
   - El query resultante pasa por la validación de prefijo
   - Palabras clave prohibidas son detectadas

### 7.6 Validación de Tipos

**Error de tipo:**
```bash
mapj protheus preset run pedidos \
  --param fecha_inicio=ayer  # Esperaba date
```

**Output:**
```json
{
  "ok": false,
  "error": {
    "code": "PARAMETER_TYPE_ERROR",
    "message": "parameter 'fecha_inicio' expects date format YYYY-MM-DD, got 'ayer'",
    "hint": "Use format: --param fecha_inicio=2026-04-01"
  }
}
```

---

## 8. Comparación con Patrones de la Industria

### 8.1 psql (PostgreSQL)

**Sintaxis:**
```bash
psql -v cliente="000001" -v loja="01" -c "SELECT * FROM SA1010 WHERE A1_COD = :cliente"
```

| Aspecto | psql | Propuesta mapj |
|---------|------|----------------|
| Sintaxis placeholder | `:var` | `:var` (compatible) |
| Flag para variables | `-v` | `--param` |
| Tipado | No | Sí (`--param-def`) |
| Documentación | Manual | Auto-detectado en `preset show` |
| Escaping | Automático | Explícito con validación |
| Presets | Archivos `.sql` con `\set` | JSON estructurado |

**Conclusión:** Compatible con sintaxis psql pero con mejoras en tipado y documentación.

### 8.2 Helm (Kubernetes)

**Sintaxis:**
```bash
helm install mychart --set cliente=000001 --set loja=01
```

| Aspecto | Helm | Propuesta mapj |
|---------|------|----------------|
| Sintaxis placeholder | `{{.var}}` | `:var` |
| Flag para valores | `--set` | `--param` |
| Archivo de valores | `values.yaml` | `presets.json` |
| Tipado | Go templates | Declarado con `--param-def` |
| Condicionales | `{{if}}` | Futuro: posible soporte |

**Conclusión:** Helm usa templates más complejos. `mapj` opta por simplicidad.

### 8.3 AWS CLI (CloudFormation)

**Sintaxis:**
```bash
aws cloudformation deploy \
  --parameter-overrides ParameterKey=Cliente,ParameterValue=000001
```

| Aspecto | AWS CLI | Propuesta mapj |
|---------|---------|----------------|
| Sintaxis | Verbosa | Compacta (`--param key=value`) |
| Tipado | En template | En preset |
| Validación | En deploy | Antes de ejecutar |

**Conclusión:** AWS es más verboso. `mapj` prioriza ergonomía para uso diario.

### 8.4 kubectl

**Sintaxis:**
```bash
kubectl get pods -o go-template='{{.metadata.name}}'
```

**Conclusión:** kubectl usa templates solo para output formatting, no para inputs de queries.

### 8.5 Resumen Comparativo

| CLI | Sintaxis placeholder | Flag parámetros | Tipado | Escaping |
|-----|---------------------|-----------------|--------|----------|
| **psql** | `:var` | `-v` | No | Automático |
| **mysql** | `@var` | No dedicado | No | Manual |
| **Helm** | `{{.var}}` | `--set` | Implícito | Template engine |
| **AWS** | N/A | `--parameter-overrides` | En template | Validación |
| **mapj (propuesto)** | `:var` | `--param` | Explícito | Validado + escapado |

---

## 9. Recomendación Final de Implementación

### 9.1 Resumen de Decisiones

| Decisión | Opción | Justificación |
|----------|--------|---------------|
| **Sintaxis placeholder** | `:param` | Compatible con psql, simple |
| **Flag CLI** | `--param key=value` | Ergonómico, parseable |
| **Tipado** | `--param-def` opcional | Valida tipos, documentación |
| **Escaping** | Validación + escaping multicapa | Defensa contra SQL injection |
| **Detección auto** | Regex en `preset add` | Descubre placeholders automáticamente |
| **Documentación** | `preset show` lista params | Agent-friendly |

### 9.2 Ejemplos de Uso Completos

#### Ejemplo 1: Query simple con un parámetro

```bash
# Definir preset
mapj protheus preset add cliente-by-codigo \
  --query "SELECT * FROM SA1010 WHERE A1_COD = :codigo" \
  --description "Busca cliente por código"

# Ejecutar
mapj protheus preset run cliente-by-codigo --param codigo=000001
```

#### Ejemplo 2: Query con múltiples parámetros y tipos

```bash
# Definir preset con tipos
mapj protheus preset add pedidos-rango \
  --query "SELECT C5_NUM, C5_EMISSAO FROM SC5010 WHERE C5_EMISSAO BETWEEN :fecha_inicio AND :fecha_fin AND C5_VEND = :vendedor" \
  --param-def fecha_inicio:date \
  --param-def fecha_fin:date \
  --param-def vendedor:string \
  --description "Pedidos por rango de fechas y vendedor"

# Ejecutar
mapj protheus preset run pedidos-rango \
  --param fecha_inicio=2026-01-01 \
  --param fecha_fin=2026-03-31 \
  --param vendedor=000001
```

#### Ejemplo 3: Query con lista de valores

```bash
# Definir preset con lista
mapj protheus preset add clientes-estados \
  --query "SELECT A1_COD, A1_NOME, A1_EST FROM SA1010 WHERE A1_EST IN (:estados)" \
  --param-def estados:list \
  --param-desc estados="Estados separados por coma (ej: SP,RJ,MG)"

# Ejecutar
mapj protheus preset run clientes-estados --param estados=SP,RJ,MG
```

#### Ejemplo 4: Uso por un LLM (agent-friendly)

```bash
# LLM descubre parámetros
$ mapj protheus preset show pedidos-rango
{
  "ok": true,
  "result": {
    "name": "pedidos-rango",
    "query": "SELECT C5_NUM, C5_EMISSAO FROM SC5010 WHERE C5_EMISSAO BETWEEN :fecha_inicio AND :fecha_fin",
    "parameters": [
      {"name": "fecha_inicio", "type": "date", "required": true, "description": ""},
      {"name": "fecha_fin", "type": "date", "required": true, "description": ""}
    ]
  }
}

# LLM ejecuta con parámetros
$ mapj protheus preset run pedidos-rango \
    --param fecha_inicio=2026-01-01 \
    --param fecha_fin=2026-03-31 \
    --output-file pedidos_q1.toon
{
  "ok": true,
  "result": {
    "rows": 1523,
    "columns": 2,
    "format": "toon",
    "output_file": "pedidos_q1.toon"
  }
}
```

### 9.3 Orden de Implementación

1. **Fase 1: Básico**
   - Detección de placeholders (`:param`)
   - Flag `--param` para ejecución
   - Interpolación con escaping básico

2. **Fase 2: Tipado**
   - `--param-def` con tipos básicos (string, int, date, bool)
   - Validación de tipos antes de ejecutar
   - Error messages claros

3. **Fase 3: Avanzado**
   - Tipo `list` para IN clauses
   - Parámetros con defaults
   - Formatos de fecha personalizados

4. **Fase 4: Opcional**
   - Templates condicionales (`{{if}}`)
   - JSON input para parámetros
   - Audit logging

---

## 10. Referencias

### 10.1 Documentación de la Industria

- **PostgreSQL psql:** https://www.postgresql.org/docs/current/app-psql.html
- **Helm Values:** https://helm.sh/docs/chart_best_practices/values/
- **AWS CloudFormation Parameters:** https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/parameters-section-structure.html
- **OWASP SQL Injection Prevention:** https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html

### 10.2 Archivos del Proyecto mapj

- `.factory/research/protheus-presets-analysis.md` - Análisis base de presets
- `internal/cli/protheus.go` - Implementación de comandos query
- `pkg/protheus/query.go` - Validación de queries (`ValidateReadOnly`)
- `skills/mapj-protheus-query/SKILL.md` - Skill agent-friendly

---

**Fin del documento**
