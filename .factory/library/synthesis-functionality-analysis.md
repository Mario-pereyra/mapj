# Síntesis: Funcionalidad y Refactorings — main vs gem

## Resumen

**gem introduce 7 features nuevas significativas** con valor agregado variable. El refactor de dominios presenta mejoras reales pero con **3 bugs críticos** que comprometen la estabilidad. El valor total de gem es alto, pero requiere fixes antes de merge.

---

## VAL-FUNC-001: Features Nuevas en gem

| Feature | Descripción | Valor | Justificación |
|---------|-------------|-------|---------------|
| **TOON Formatter** | Output format ~40% más token-efficient que JSON | ⭐⭐⭐⭐⭐ ALTO | 486 líneas nuevas + 452 tests. Crítico para consumo por LLMs. Reducción significativa de tokens en respuestas. |
| **Auto-retry con Backoff** | Reintento automático (3 intentos) con exponential backoff + jitter | ⭐⭐⭐⭐ ALTO | Mejora resilencia sin intervención del cliente. Implementado en `pkg/confluence/client.go`. |
| **Auto-paginación** | Loop automático en `Search()` para obtener todos los resultados | ⭐⭐⭐⭐ ALTO | Simplifica drásticamente el consumo de API. Antes: cliente debía manejar paginación manualmente. |
| **Exit Codes Typesafe** | Interfaz `ExitCoder` con tipos concretos (`AuthError`, `UsageError`, etc.) | ⭐⭐⭐ MEDIO | Más robusto que string matching. Elimina false positives en detección de errores. |
| **Cursor Closure + maxRows** | `defer rows.Close()` correcto + early termination con maxRows | ⭐⭐⭐ MEDIO | Prevención de leaks de recursos. Útil para queries grandes. |
| **Concurrent Exports** | Worker pool (10 goroutines) para exports paralelos | ⭐⭐⭐ MEDIO | Acelera exports masivos, pero **BUG CRÍTICO** en mutex anula el beneficio. |
| **--version flag** | Nueva flag para mostrar versión del CLI | ⭐ BAJO | Feature cosmética, menor valor funcional. |

### Evaluación General de Features

- **Total features nuevas**: 7
- **Features con valor ALTO**: 3 (TOON, Auto-retry, Auto-paginación)
- **Features con valor MEDIO**: 3 (Exit codes, Cursor closure, Concurrent exports)
- **Features con valor BAJO**: 1 (--version)

---

## VAL-FUNC-002: Evaluación de Refactorings por Dominio

### Dominio 1: Confluence (`pkg/confluence/`)

| Refactoring | Tipo | Evidencia | Evaluación |
|-------------|------|-----------|------------|
| **Auto-retry** | ✅ Mejora Real | `client.go:78-114` - Loop con backoff exponencial | Reduce errores transitorios sin intervención del cliente |
| **Auto-paginación** | ✅ Mejora Real | `search.go:82-130` - Loop automático con batchSize 100 | Simplifica API, elimina código cliente para paginación |
| **Payload optimization** | ✅ Mejora Real | `search.go` - `expand: "space"` vs expand completo | Reduce tamaño de respuesta, mejora performance |
| **PageRef simplificado** | ✅ Mejora Real | `search.go:29-37` - Campos eliminados: space, labels, ancestors, version, lastUpdated | Token savings significativo para LLMs |
| **Concurrent exports** | ⚠️ Mejora con BUG | `export.go:117-140` - Worker pool con semaphore | **BUG: Mutex local no sincroniza** |
| **Pipeline extraction** | ✅ Mejora Real | `search_pipeline.go` nuevo (86 líneas) | Separa concerns, más mantenible |

**Veredicto Confluence**: **Mejora Real con Bug Crítico** - 5 mejoras reales, 1 bug crítico.

---

### Dominio 2: Protheus (`pkg/protheus/`)

| Refactoring | Tipo | Evidencia | Evaluación |
|-------------|------|-----------|------------|
| **Cursor closure** | ✅ Mejora Real | `query.go:92-94` - `defer rows.Close()` | Previene leaks de conexión DB |
| **maxRows parameter** | ✅ Mejora Real | `query.go:95-98` - Early break con contador | Control de memoria para queries grandes |
| **SQL Validation** | 🔴 REGRESIÓN CRÍTICA | `query.go:44-58` - Solo verifica prefijo | **No detecta SQL injection** |

**Veredicto Protheus**: **REGRESIÓN** - 2 mejoras reales vs 1 bug crítico de seguridad.

**Detalle SQL Validation REGRESIÓN**:
```go
// main: Validación robusta con keywords peligrosos
dangerous := []string{
    "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
    "TRUNCATE", "EXEC", "EXECUTE", "MERGE", "INTO", "REPLACE",
    "GRANT", "REVOKE", "DENY", "BACKUP", "RESTORE",
}

// gem: Solo verifica prefijo - NO detecta inyección
if !strings.HasPrefix(normalized, "SELECT") && 
   !strings.HasPrefix(normalized, "WITH") && 
   !strings.HasPrefix(normalized, "EXEC") {
    return fmt.Errorf("query must start with SELECT, WITH, or EXEC")
}
// ⚠️ Pasa: "SELECT * FROM users; DROP TABLE users;"
```

---

### Dominio 3: TDN (`internal/cli/tdn.go`)

| Refactoring | Tipo | Evidencia | Evaluación |
|-------------|------|-----------|------------|
| **Pipeline decoupling** | ✅ Mejora Real | `tdn.go` -112 líneas, lógica movida a `search_pipeline.go` | Código más modular y mantenible |
| **Flag rename** | ⚠️ Cosmético | `--limit` → `--max-results` | Mejor semántica pero breaking change |
| **EnrichWithChildCount** | ✅ Mejora Real | Movido de CLI a Client, con timeout estricto (2s) | Mejor organización, timeout previene hangs |

**Veredicto TDN**: **Mejora Real** - Simplificación significativa sin bugs introducidos.

---

## VAL-FUNC-003: Bugs Críticos Identificados

### 🔴 Bug 1: SQL Validation REGRESIÓN

| Campo | Valor |
|-------|-------|
| **Severidad** | 🔴 CRÍTICO |
| **Archivo** | `pkg/protheus/query.go:44-58` |
| **Descripción** | Validación solo verifica que query empiece con SELECT/WITH/EXEC. No detecta SQL injection inline. |
| **Código Afectado** | `ValidateReadOnly()` - Eliminó lista de keywords peligrosos |
| **Impacto** | Seguridad - Posible inyección SQL vía `; DROP TABLE` |
| **Ejemplo Exploit** | `SELECT * FROM users WHERE id = 1; DROP TABLE users; --` |
| **Recomendación** | Restaurar validación de keywords peligrosos de main. Agregar detección de `;` como separador de statements. |

---

### 🔴 Bug 2: Mutex Bug en Concurrent Exports

| Campo | Valor |
|-------|-------|
| **Severidad** | 🔴 CRÍTICO |
| **Archivo** | `pkg/confluence/export.go:214-217` |
| **Descripción** | Mutex declarado como variable local dentro de goroutine. Cada goroutine tiene su propia instancia, no sincroniza nada. |
| **Código Afectado** |
```go
// BUG: Mutex local no sincroniza entre goroutines
mu := sync.Mutex{}  // ← Debería ser campo de Client o parámetro
mu.Lock()
_ = WriteManifest(opts.OutputPath, entry)
mu.Unlock()
```
| **Impacto** | Race condition en escritura de manifest, posible corrupción de datos |
| **Recomendación** | Usar mutex compartido (campo de `Client` o parámetro de función) |

---

### 🟡 Bug 3: Breaking Changes sin Migración

| Campo | Valor |
|-------|-------|
| **Severidad** | 🟡 ALTO |
| **Archivos** | `internal/cli/confluence.go`, `internal/cli/protheus.go`, `internal/cli/tdn.go` |
| **Descripción** | Eliminación de funcionalidad sin documentar migración ni alternativas |
| **Cambios** |
| - Formato `csv` eliminado | Sin alternativa documentada |
| - Formato `human` eliminado | Sin alternativa documentada |
| - Comando `retry-failed` removido | Sin reemplazo |
| - Flag `--limit` → `--max-results` | Breaking change para scripts existentes |
| **Impacto** | Ruptura de compatibilidad para usuarios existentes |
| **Recomendación** | Documentar migración en CHANGELOG o restaurar funcionalidad con deprecation warning |

---

## Resumen de Refactorings

| Dominio | Mejoras Reales | Bugs Críticos | Veredicto |
|---------|----------------|---------------|-----------|
| **Confluence** | 5 | 1 (mutex) | ⚠️ Mejora con bug |
| **Protheus** | 2 | 1 (SQL injection) | 🔴 REGRESIÓN |
| **TDN** | 2 | 0 | ✅ Mejora limpia |
| **Core** | 3 (TOON, exit codes, version) | 0 | ✅ Mejora limpia |
| **Total** | **12** | **2** | ⚠️ **Requiere fixes** |

---

## Conclusión

**gem tiene valor significativo pero comprometido por bugs críticos:**

1. ✅ **12 mejoras reales** que aportan valor funcional
2. ✅ **3 features nuevas de alto valor** (TOON, auto-retry, auto-paginación)
3. ⚠️ **2 bugs críticos de seguridad/estabilidad** (SQL injection, mutex)
4. ⚠️ **1 breaking change sin migración** (CSV/human eliminados)

**Recomendación**: **Fix-and-merge** — Los 2 bugs críticos deben fixearse antes de merge. El breaking change requiere documentación de migración.

### Prioridad de Fixes

1. **URGENTE**: SQL Validation — Restaurar validación de keywords peligrosos
2. **URGENTE**: Mutex Bug — Usar mutex compartido para concurrent exports
3. **IMPORTANTE**: Breaking changes — Documentar migración o agregar deprecation warnings
