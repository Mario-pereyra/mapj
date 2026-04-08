# Análisis de Viabilidad: CRUD de Presets de Queries para `mapj protheus query`

**Fecha:** 2026-04-08  
**Autor:** Factory Droid (Worker Analysis)  
**Estado:** Análisis completado

---

## 1. Resumen Ejecutivo

La implementación de un CRUD de presets de queries es **viável y recomendada**. El proyecto ya cuenta con los patrones de diseño necesarios (CRUD de conexiones, almacenamiento encriptado, sistema de envelopes para output) que pueden replicarse directamente para presets.

**Recomendación final:** Implementar presets en un archivo separado (`~/.config/mapj/presets.json`) SIN encriptación, ya que las queries no contienen credenciales. Esto simplifica la implementación, permite edición manual y mantiene la separación de responsabilidades.

**Estimación de complejidad:** Media (2-3 días de desarrollo con tests)

---

## 2. Análisis de Arquitectura Actual

### 2.1 Estructura de Archivos Clave

| Archivo | Propósito | Patrones Relevantes |
|---------|-----------|---------------------|
| `internal/auth/store.go` | Almacenamiento encriptado de credenciales | `ServiceCreds` struct, mapas de perfiles, AES-GCM |
| `internal/cli/protheus_connection.go` | CRUD de conexiones | Subcomandos Cobra, flags, envelopes de output |
| `internal/cli/protheus.go` | Ejecución de queries | `--connection`, `--max-rows`, `--output-file` |
| `pkg/protheus/query.go` | Lógica de queries | `ValidateReadOnly()`, `QueryResult` |
| `internal/output/envelope.go` | Formato de salida | `Envelope`, `ErrDetail`, `NewEnvelope()` |

### 2.2 Patrones de Diseño Existentes

#### 2.2.1 Estructura de Datos para Perfiles (`store.go`)

```go
type ServiceCreds struct {
    TDN             *TDNCreds                   `json:"tdn,omitempty"`
    Confluence      *ConfluenceCreds            `json:"confluence,omitempty"`
    ProtheusProfiles map[string]*ProtheusProfile `json:"protheusProfiles,omitempty"`
    ProtheusActive   string                      `json:"protheusActive,omitempty"`
}

type ProtheusProfile struct {
    Name     string `json:"name"`
    Server   string `json:"server"`
    Port     int    `json:"port"`
    Database string `json:"database"`
    User     string `json:"user"`
    Password string `json:"password"`
}
```

**Patrones clave:**
- Uso de mapas `map[string]*Type` para almacenar entidades por nombre
- Campo "Active" para la entidad actualmente seleccionada
- Métodos helper: `ActiveProtheusProfile()`, `ProtheusProfileNames()`, `HasProtheusProfiles()`

#### 2.2.2 Patrón de Subcomandos CRUD (`protheus_connection.go`)

```
protheus connection
├── add <name>      --server, --database, --user, --password, --port, --use
├── list            (sin argumentos)
├── use <name>      (switch active)
├── remove <name>   (delete)
├── show [name]     (password masked, defaults to active)
└── ping [name]     (test connectivity)
```

**Patrones de output:**
- Éxito: `output.NewEnvelope(cmd.CommandPath(), result)`
- Error: `output.NewErrorEnvelope(cmd.CommandPath(), code, message, retryable)`
- Error con hint: `output.NewErrorEnvelopeWithHint(...)`

#### 2.2.3 Sistema de Flags para Query (`protheus.go`)

```go
protheusQueryCmd.Flags().IntVar(&protheusMaxRows, "max-rows", 10000, "...")
protheusQueryCmd.Flags().StringVar(&protheusConnection, "connection", "", "...")
protheusQueryCmd.Flags().StringVar(&protheusOutputFile, "output-file", "", "...")
```

---

## 3. Estructura de Datos Propuesta

### 3.1 QueryPreset Struct

```go
// QueryPreset representa una query guardada con metadatos
type QueryPreset struct {
    Name        string `json:"name"`                  // Identificador único
    Description string `json:"description,omitempty"` // Descripción humana
    Query       string `json:"query"`                 // SQL query completa
    Connection  string `json:"connection,omitempty"` // Perfil de conexión preferido (opcional)
    MaxRows     int    `json:"maxRows,omitempty"`     // Límite de filas (opcional, default: 10000)
    CreatedAt   string `json:"createdAt,omitempty"`   // ISO 8601 timestamp
    UpdatedAt   string `json:"updatedAt,omitempty"`   // ISO 8601 timestamp
    Tags        []string `json:"tags,omitempty"`       // Etiquetas para organización
}
```

### 3.2 PresetStore (nuevo archivo: `internal/preset/store.go`)

```go
type PresetStore struct {
    path string  // ~/.config/mapj/presets.json
}

type PresetsFile struct {
    Presets       map[string]*QueryPreset `json:"presets"`
    ActivePreset  string                  `json:"activePreset,omitempty"`
}
```

### 3.3 Métodos Helper (siguiendo el patrón de ServiceCreds)

```go
// ActivePreset returns the current active preset
func (p *PresetFile) ActivePreset() *QueryPreset

// SetPreset adds or updates a named preset
func (p *PresetFile) SetPreset(preset *QueryPreset, setActive bool)

// PresetNames returns sorted preset names
func (p *PresetFile) PresetNames() []string

// HasPresets returns true if there is at least one preset
func (p *PresetFile) HasPresets() bool
```

---

## 4. Comandos CLI Propuestos

### 4.1 Estructura de Comandos

```
protheus preset
├── add <name>       --query, --description, --connection, --max-rows, --tags, --use
├── list             [--tag, --connection]
├── run <name>       [--connection, --max-rows, --output-file]
├── show [name]      (defaults to active)
├── edit <name>      --query, --description, --connection, --max-rows, --tags
├── remove <name>    (delete)
└── use <name>       (set as active for quick run)
```

### 4.2 Detalle de Comandos

#### `preset add <name>`

```bash
mapj protheus preset add clientes-top10 \
  --query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010 WHERE D_E_L_E_T_ = ''" \
  --description "Top 10 clientes por código" \
  --connection TOTALPEC_BIB \
  --max-rows 100 \
  --tags "clientes,reportes" \
  --use
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset add",
  "result": {
    "name": "clientes-top10",
    "connection": "TOTALPEC_BIB",
    "setActive": true
  }
}
```

#### `preset list`

```bash
mapj protheus preset list
mapj protheus preset list --tag reportes
mapj protheus preset list --connection TOTALPEC_BIB
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset list",
  "result": {
    "presets": [
      {"name": "clientes-top10", "description": "...", "connection": "TOTALPEC_BIB", "active": true},
      {"name": "pedidos-hoy", "description": "...", "connection": null, "active": false}
    ],
    "count": 2,
    "active": "clientes-top10"
  }
}
```

#### `preset run <name>`

```bash
# Usa la conexión configurada en el preset (si existe) o la activa
mapj protheus preset run clientes-top10

# Override de conexión
mapj protheus preset run clientes-top10 --connection TOTALPEC_PRD

# Override de max-rows y output
mapj protheus preset run clientes-top10 --max-rows 500 --output-file ./report.toon
```

**Comportamiento:**
1. Carga el preset por nombre
2. Resuelve conexión (preset → flag → activa → error)
3. Ejecuta la query con los parámetros del preset
4. Permite overrides via flags

**Output:** Igual que `protheus query` (rows, columns, count o file summary)

#### `preset show [name]`

```bash
mapj protheus preset show               # Muestra el preset activo
mapj protheus preset show clientes-top10  # Muestra preset específico
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset show",
  "result": {
    "name": "clientes-top10",
    "description": "Top 10 clientes por código",
    "query": "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010 WHERE D_E_L_E_T_ = ''",
    "connection": "TOTALPEC_BIB",
    "maxRows": 100,
    "tags": ["clientes", "reportes"],
    "active": true
  }
}
```

#### `preset edit <name>`

```bash
mapj protheus preset edit clientes-top10 --max-rows 200
mapj protheus preset edit clientes-top10 --query "SELECT TOP 20 ..."
mapj protheus preset edit clientes-top10 --connection TOTALPEC_PRD
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset edit",
  "result": {
    "name": "clientes-top10",
    "updated": ["maxRows"]
  }
}
```

#### `preset remove <name>`

```bash
mapj protheus preset remove clientes-top10
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset remove",
  "result": {
    "removed": "clientes-top10",
    "wasActive": true,
    "newActive": "pedidos-hoy"
  }
}
```

#### `preset use <name>`

```bash
mapj protheus preset use clientes-top10
```

**Output:**
```json
{
  "ok": true,
  "command": "mapj protheus preset use",
  "result": {
    "previous": "pedidos-hoy",
    "active": "clientes-top10"
  }
}
```

---

## 5. Opciones de Almacenamiento

### 5.1 Opción A: Mismo archivo encriptado (`credentials.enc`)

**Implementación:**
```go
type ServiceCreds struct {
    // ... existentes ...
    QueryPresets      map[string]*QueryPreset `json:"queryPresets,omitempty"`
    QueryPresetActive string                   `json:"queryPresetActive,omitempty"`
}
```

| Ventajas | Desventajas |
|----------|-------------|
| No requiere nuevo archivo | Mezcla datos de diferente naturaleza |
| Reutiliza infraestructura de encriptación | Las queries no son sensibles → encriptación innecesaria |
| Atomicidad en operaciones | Dificulta edición manual/debugging |
| Consistencia con patrón existente |Archivo crece con cada preset |

**Veredicto:** No recomendado. La encriptación es excesiva para queries.

---

### 5.2 Opción B: Archivo JSON separado (`presets.json`)

**Implementación:**
```go
// Nuevo archivo: internal/preset/store.go
type PresetStore struct {
    path string  // ~/.config/mapj/presets.json
}

func NewPresetStore() (*PresetStore, error) {
    home, _ := os.UserHomeDir()
    configDir := filepath.Join(home, ".config", "mapj")
    return &PresetStore{
        path: filepath.Join(configDir, "presets.json"),
    }, nil
}
```

| Ventajas | Desventajas |
|----------|-------------|
| Separación de responsabilidades | Requiere nuevo store |
| Sin encriptación innecesaria | Archivo adicional |
| Editable manualmente para debugging | Debe crear directorio si no existe |
| Menor acoplamiento | Puede necesitar migración futura |
| Formato legible | |

**Veredicto:** **Recomendado.** Simplifica la implementación y mantiene la separación de preocupaciones.

---

### 5.3 Opción C: YAML con comentarios (`presets.yaml`)

| Ventajas | Desventajas |
|----------|-------------|
| Permite comentarios documentales | Librería YAML adicional |
| Más legible para humanos | No sigue patrón JSON existente |
| | Parsing más lento |

**Veredicto:** No recomendado. Rompe la consistencia con el patrón JSON existente.

---

## 6. Consideraciones de Seguridad

### 6.1 Análisis de Sensibilidad de Datos

| Componente | ¿Contiene datos sensibles? | Recomendación |
|------------|---------------------------|---------------|
| Query SQL | Potencialmente | NO encriptar, pero validar |
| Nombre de preset | No | Sin encriptación |
| Connection name | No (es una referencia) | Sin encriptación |
| Tags | No | Sin encriptación |
| MaxRows | No | Sin encriptación |

### 6.2 Riesgos de Information Disclosure en Queries

**Escenario de riesgo:** Una query puede contener:
- Valores hardcodeados: `WHERE A1_CGC = '12345678000199'` (CNPJ/CPF)
- Nombres de tablas sensibles: `FROM SA1010` (clientes)
- Condiciones que revelan lógica de negocio

**Mitigación:**
1. **Validación de seguridad en `preset add`:** Warning si la query contiene patrones sensibles
2. **Permisos de archivo:** `0644` (lectura solo para owner, grupo/otros sin acceso)
3. **Documentación:** Avisar al usuario que los presets son texto plano

### 6.3 Validación de Queries en Presets

```go
// Reutilizar ValidateReadOnly() de pkg/protheus/query.go
func ValidatePresetQuery(query string) error {
    if err := ValidateReadOnly(query); err != nil {
        return fmt.Errorf("preset query validation: %w", err)
    }
    return nil
}
```

### 6.4 Recomendación Final de Seguridad

- **NO encriptar presets** - Las queries no contienen credenciales
- **Usar permisos restrictivos** - `0600` para el archivo `presets.json`
- **Reutilizar validación existente** - `ValidateReadOnly()` para todas las queries
- **Warning en `add`** - Informar si la query contiene patrones potencialmente sensibles

---

## 7. Edge Cases y Consideraciones

### 7.1 Edge Cases Identificados

| Edge Case | Solución |
|-----------|----------|
| Preset con conexión inexistente | Validar en `run`: error con hint de `connection list` |
| Preset activo eliminado | Auto-seleccionar otro preset o limpiar `activePreset` |
| Query inválida en preset | Validar en `add` y `edit` con `ValidateReadOnly()` |
| Nombre duplicado en `add` | Error: "preset already exists, use `edit` to modify" |
| Flags de override en conflicto | Documentar precedencia: flag CLI > preset > default |
| Archivo presets.json corrupto | Backup automático antes de write, error graceful |
| Directorio `.config/mapj` no existe | Crear en `NewPresetStore()` |
| Preset sin conexión configurada | Usar conexión activa, error si no hay activa |

### 7.2 Precedencia de Configuración en `preset run`

```
1. Flag --connection (CLI)       → Máxima prioridad
2. Preset.Connection (guardado)  → Media prioridad  
3. Active connection             → Default
4. Error: NO_CONNECTION          → Si no hay ninguna
```

```
1. Flag --max-rows (CLI)         → Máxima prioridad
2. Preset.MaxRows (guardado)     → Media prioridad
3. Default 10000                 → Fallback
```

### 7.3 Interacción con Sistema Existente

- `protheus query` permanece inalterado
- `preset run` es un wrapper que llama internamente a la lógica de `query`
- El flag `--connection` funciona igual en ambos comandos

---

## 8. Estimación de Complejidad

### 8.1 Breakdown de Tareas

| Tarea | Estimación | Complejidad |
|-------|------------|-------------|
| Crear `internal/preset/store.go` | 2h | Baja |
| Crear `internal/cli/protheus_preset.go` | 4h | Media |
| Implementar comandos CRUD (6 subcomandos) | 4h | Media |
| Integrar `preset run` con `query.go` | 2h | Baja |
| Tests unitarios | 3h | Media |
| Documentación en Long descriptions | 1h | Baja |

**Total:** ~16 horas (2 días de trabajo)

### 8.2 Archivos Nuevos Requeridos

```
mapj_cli/
├── internal/
│   └── preset/
│       └── store.go          (nuevo, ~120 líneas)
└── internal/cli/
    └── protheus_preset.go     (nuevo, ~400 líneas)
```

### 8.3 Archivos a Modificar

```
mapj_cli/
└── internal/cli/
    └── protheus.go            (añadir registro de presetCmd en init())
```

---

## 9. Recomendación Final

### 9.1 Implementar con las siguientes decisiones:

1. **Almacenamiento:** Archivo separado `~/.config/mapj/presets.json` SIN encriptación
2. **Permisos:** `0600` para el archivo de presets
3. **Validación:** Reutilizar `ValidateReadOnly()` de `pkg/protheus/query.go`
4. **Comandos:** Implementar los 6 subcomandos propuestos
5. **Integración:** `preset run` como wrapper de `query`, sin modificar el comando original

### 9.2 Orden de Implementación Sugerido

1. Crear `internal/preset/store.go` con tests
2. Implementar `preset add/list/show` (básicos)
3. Implementar `preset run` (core functionality)
4. Implementar `preset remove/edit/use`
5. Añadir validaciones y edge cases
6. Documentación y tests finales

### 9.3 Próximos Pasos

Si se aprueba este análisis, el siguiente paso sería la implementación siguiendo el orden sugerido en 9.2, con TDD como metodología (tests antes de implementación).

---

## 10. Referencias

- `internal/auth/store.go` - Patrón de almacenamiento y estructura de datos
- `internal/cli/protheus_connection.go` - Patrón de comandos CRUD
- `internal/cli/protheus.go` - Patrón de ejecución de queries
- `pkg/protheus/query.go` - Validación de queries
- `internal/output/envelope.go` - Formato de output
