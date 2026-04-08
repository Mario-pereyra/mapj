# Síntesis: Calidad de Código — main vs gem

## Resumen

**Ganador: main (8.5/10 vs 8.1/10)**

La rama `main` mantiene una ventaja en calidad de código debido a su arquitectura más limpia y menor complejidad ciclomática. Aunque `gem` introduce mejoras significativas (TOON formatter, auto-retry, auto-paginación), presenta **3 bugs críticos** que afectan su puntuación: regresión en SQL validation, mutex bug en exports concurrentes, y breaking changes sin migración.

---

## Tabla Comparativa por Área

| Área | main | gem | Ganador | Justificación |
|------|------|-----|---------|---------------|
| **Arquitectura** | 8.5 | 8.0 | **main** | Estructura más simple y predecible. gem añade complejidad con TOON (486 líneas) y search_pipeline.go sin justificación arquitectural clara. |
| **Patrones** | 8.0 | 8.0 | **empate** | Ambas ramas usan patrones consistentes. gem introduce ExitCoder typesafe (positivo) pero tiene mutex bug (negativo). |
| **Errores** | 8.0 | 7.5 | **main** | gem tiene +26 tests en errors (positivo) pero SQL validation REGRESIÓN no detecta inyección SQL (crítico). |
| **Dominios** | 8.0 | 7.5 | **main** | gem tiene mejoras reales (cursor closure, auto-paginación) pero bugs críticos en protheus (SQL) y confluence (mutex). |

**Puntuación ponderada**: main 8.1/10 vs gem 7.8/10 → **main gana por consistencia y ausencia de bugs críticos**

---

## Análisis Detallado por Área

### 1. Arquitectura

#### main (8.5/10)
- **Estructura limpia**: Separación clara entre `pkg/` (clientes API) e `internal/` (CLI, auth, output)
- **Complejidad controlada**: Archivos mantienen tamaños razonables
- **Sin TOON**: Output solo JSON/CSV/Human

**Evidencia específica**:
```
pkg/confluence/client.go:57 — _getHeaders() con guion bajo (no idiomático)
pkg/protheus/query.go — 144 líneas, bien estructurado
```

#### gem (8.0/10)
- **Nuevas abstracciones**: `search_pipeline.go` (86 líneas), `toon_formatter.go` (486 líneas)
- **Complejidad añadida**: markdown.go con 662 líneas, múltiples handlers de macro
- **Refactorings**: Simplificación de tdn.go (-112 líneas)

**Evidencia específica**:
```
internal/output/toon_formatter.go — 486 líneas, 25+ funciones helper
pkg/confluence/markdown.go — 662 líneas con 30+ funciones de render
pkg/confluence/search_pipeline.go — Nueva abstracción de pipeline
```

**Problemas detectados**:
- `toon_formatter.go`: Funciones como `encodeRootArray()`, `encodeObjectTabularArray()` con lógica compleja de branching
- `markdown.go`: Handler registration con 20+ callbacks, difícil de mantener

---

### 2. Patrones de Diseño

#### main (8.0/10)
- **Exit codes**: Mapeo por string matching (`MapErrorToCode` usa `strings.Contains`)
- **Error handling**: Patrón simple pero frágil
- **Sin auto-retry**: El cliente debe manejar reintentos manualmente

**Evidencia específica**:
```go
// internal/errors/codes.go:28-43 (main)
func MapErrorToCode(err error) int {
    if err == nil { return ExitSuccess }
    if exitCoder, ok := err.(ExitCoder); ok {
        return exitCoder.ExitCode()
    }
    return ExitError // Fallback genérico
}
```

#### gem (8.0/10)
- **ExitCoder typesafe**: Interfaz `ExitCoder` con tipos concretos (`AuthError`, `UsageError`, `RetryableError`)
- **Auto-retry con backoff**: `backoffWithJitter()` en client.go
- **Auto-paginación**: Loop automático en `Search()`
- **⚠️ Mutex bug**: Sincronización incorrecta en exports concurrentes

**Evidencia específica**:
```go
// internal/errors/codes.go (gem) — Typesafe exit codes
type ExitCoder interface { ExitCode() int }
type AuthError struct { Msg string }
func (e *AuthError) ExitCode() int { return ExitAuth }

// pkg/confluence/client.go:97-100 — Auto-retry
func backoffWithJitter(attempt int) {
    base := time.Duration(1<<attempt) * time.Second
    jitter := time.Duration(rand.Intn(500)) * time.Millisecond
    time.Sleep(base + jitter)
}

// ⚠️ pkg/confluence/export.go:205 — BUG: Mutex local
mu := sync.Mutex{}  // ← BUG: Cada goroutine crea su propio mutex
mu.Lock()
_ = WriteManifest(opts.OutputPath, entry)
mu.Unlock()
```

**Mutex Bug Explicado**: El mutex se declara como variable local dentro de la función, por lo que cada goroutine tiene su propia instancia. Esto no sincroniza nada entre goroutines.

---

### 3. Manejo de Errores

#### main (8.0/10)
- **Validación SQL robusta**: Detecta keywords peligrosos (INSERT, DELETE, DROP, etc.)
- **Tests básicos**: Cobertura limitada en errors

**Evidencia específica**:
```go
// pkg/protheus/query.go (main) — Validación completa
dangerous := []string{
    "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
    "TRUNCATE", "EXEC", "EXECUTE", "MERGE", "INTO", "REPLACE",
    "GRANT", "REVOKE", "DENY", "BACKUP", "RESTORE",
}
for _, keyword := range dangerous {
    pattern := fmt.Sprintf(`\b%s\b`, keyword)
    if matched, _ := regexp.MatchString(pattern, normalized); matched {
        return fmt.Errorf("query contains forbidden keyword: %s", keyword)
    }
}
```

#### gem (7.5/10)
- **+26 tests en errors**: `internal/errors/codes_test.go` nuevo con 26 casos
- **⚠️ SQL Validation REGRESIÓN**: No detecta inyección SQL
- **Exit codes mejorados**: Tipos concretos vs string matching

**Evidencia específica**:
```go
// pkg/protheus/query.go (gem) — REGRESIÓN: Validación debilitada
if !strings.HasPrefix(normalized, "SELECT") && 
   !strings.HasPrefix(normalized, "WITH") && 
   !strings.HasPrefix(normalized, "EXEC") {
    return fmt.Errorf("query must start with SELECT, WITH, or EXEC")
}
// ⚠️ NO detecta: "SELECT * FROM users; DROP TABLE users;"
```

**SQL Injection Demo**:
```sql
-- Esta query pasa la validación en gem pero NO en main:
SELECT * FROM users WHERE id = 1; DROP TABLE users; --
```

---

### 4. Dominios

#### Confluence

| Aspecto | main | gem | Notas |
|---------|------|-----|-------|
| **Auto-retry** | ❌ | ✅ | `backoffWithJitter()` con 3 reintentos |
| **Auto-paginación** | ❌ | ✅ | Loop automático en `Search()` |
| **Mutex** | N/A | ⚠️ BUG | Local mutex no sincroniza |
| **Tests** | Básicos | +339 líneas | Cobertura 66.7% en errors |

**Evidencia específica**:
```go
// pkg/confluence/search.go:82-130 (gem) — Auto-paginación
for remainingLimit > 0 {
    batchSize = 100
    if remainingLimit < 100 { batchSize = remainingLimit }
    // ... fetch batch ...
    remainingLimit -= raw.Size
    currentStart += raw.Size
    if raw.Links.Next == "" { break }
}
```

#### Protheus

| Aspecto | main | gem | Notas |
|---------|------|-----|-------|
| **SQL Validation** | ✅ Robusta | ⚠️ REGRESIÓN | No detecta inyección |
| **Cursor Closure** | ❌ | ✅ | `defer rows.Close()` correcto |
| **maxRows** | ❌ | ✅ | Nuevo parámetro con early break |
| **Tests** | Básicos | +104 líneas | Validación tests actualizados |

**Evidencia específica**:
```go
// pkg/protheus/query.go (gem) — Mejora: cursor closure
rows, err := db.QueryContext(ctx, sqlQuery)
if err != nil { return nil, fmt.Errorf("query failed: %w", err) }
defer rows.Close() // ← Correcto

// Nuevo: maxRows con early break
for rows.Next() {
    if maxRows > 0 && count >= maxRows {
        break // Early termination
    }
    // ... scan row ...
}
```

#### TDN

| Aspecto | main | gem | Notas |
|---------|------|-----|-------|
| **Complejidad** | Alto | Simplificado | -112 líneas en tdn.go |
| **Pipeline** | Inline | Extraído | `search_pipeline.go` nuevo |

---

## Bugs Críticos Detectados en gem

### Bug 1: SQL Validation REGRESIÓN
- **Archivo**: `pkg/protheus/query.go:44-58`
- **Severidad**: 🔴 CRÍTICO
- **Descripción**: Validación solo verifica prefijo (SELECT/WITH/EXEC), no detecta SQL injection
- **Ejemplo**: `SELECT * FROM users; DROP TABLE users;` pasa la validación
- **Recomendación**: Restaurar validación de keywords peligrosos de main

### Bug 2: Mutex Bug
- **Archivo**: `pkg/confluence/export.go:205`
- **Severidad**: 🔴 CRÍTICO
- **Descripción**: Mutex local en cada goroutine no sincroniza correctamente
- **Código afectado**:
  ```go
  mu := sync.Mutex{} // ← BUG: Debería ser parámetro o campo de Client
  mu.Lock()
  _ = WriteManifest(opts.OutputPath, entry)
  mu.Unlock()
  ```
- **Recomendación**: Usar mutex compartido (campo de Client o parámetro)

### Bug 3: Breaking Changes sin Migración
- **Archivos**: `internal/cli/confluence.go`, `internal/cli/protheus.go`
- **Severidad**: 🟡 ALTO
- **Descripción**: 
  - Formato `csv` y `human` eliminados sin alternativa
  - Comando `retry-failed` removido sin reemplazo
- **Recomendación**: Documentar migración o restaurar funcionalidad

---

## Conclusión

**main es superior en calidad de código** debido a:

1. **Ausencia de bugs críticos**: main no tiene regresiones de seguridad
2. **Arquitectura más simple**: Menor complejidad ciclomática
3. **SQL Validation correcta**: Protección contra inyección SQL

**gem tiene valor pero requiere fixes**:

1. ✅ Mejoras reales: auto-retry, auto-paginación, cursor closure, TOON formatter
2. ⚠️ 3 bugs críticos que deben fixearse antes de merge
3. ✅ +339 líneas de tests significativamente mejoran cobertura

**Recomendación**: **Fix-and-merge** — Aplicar fixes a bugs críticos en gem antes de considerar merge.
