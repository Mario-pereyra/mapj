# Síntesis: Documentación — main vs gem

## Resumen

**Ganador: gem (9.0/10 vs 7.5/10)**

La rama `gem` presenta documentación significativamente superior a `main`. Las mejoras incluyen: formato TOON como default, Agentic Features bien documentadas, y guías más concisas y enfocadas. Sin embargo, `main` tiene ventajas en troubleshooting detallado y documentación de formatos eliminados (CSV/Human).

---

## VAL-DOCS-001: README Comparado

### Tabla Comparativa por Aspecto

| Aspecto | main | gem | Ganador | Justificación |
|--------|------|-----|---------|---------------|
| **Claridad** | 8.0/10 | 8.5/10 | **gem** | Estructura más limpia, tablas más concisas, menos verbosidad |
| **Completitud** | 8.0/10 | 9.0/10 | **gem** | Agrega Agentic Features (TOON, Safety Tripwire, Auto-Healing, High Concurrency, Prefix Validation, Early Cursor Closure) |
| **Ejemplos** | 8.5/10 | 8.5/10 | **empate** | Ambos tienen ejemplos claros y comandos concretos |
| **Troubleshooting** | 9.0/10 | 7.0/10 | **main** | Tabla de troubleshooting extensiva (6 problemas vs implícito en gem) |
| **Instalación** | 9.0/10 | 7.0/10 | **main** | Prerequisites documentadas, opción A (pre-built) y opción B (source), verificación |
| **Output formats** | 8.0/10 | 9.0/10 | **gem** | Mejor organización, TOON optimizado para tokens, auto-detection |
| **Exit codes** | 8.0/10 | 9.0/10 | **gem** | Tabla más concisa con columna "Agent Action" clara |

### Análisis Detallado

#### README.md — main (Puntuación: 7.5/10)

**Fortalezas**:
- Sección de instalación completa con prerequisitos y opciones múltiples
- Tabla de troubleshooting con 6 problemas comunes y soluciones
- Output format con `-o csv` documentado para spreadsheets
- Estructura de proyecto detallada
- Auth con auto-detection explicada

**Debilidades**:
- Verbosidad excesiva en algunas secciones
- Falta documentación de features agentic modernos
- Output default (`llm`) menos eficiente que TOON

**Evidencia específica**:
```markdown
## Troubleshooting
| Symptom | Cause | Fix |
|---------|-------|-----|
| `401` on tdninterno | Old credentials with wrong auth type | `mapj auth login confluence --url ... --token TOKEN` |
| `PAGE_NOT_FOUND` | Wrong URL or private page | Try `pageId=` URL or check access |
| `i/o timeout` | VPN not connected | Connect TOTALPEC VPN... |
```

#### README.md — gem (Puntuación: 9.0/10)

**Fortalezas**:
- Sección "Agentic Features (CLI v0.2.0)" con 6 features nuevas bien documentadas
- Output format con TOON como default optimizado para tokens
- Tabla de exit codes con "Agent Action" - accionable para LLMs
- Estructura más limpia y enfocada

**Debilidades**:
- Sección de instalación simplificada (pierde prerequisitos)
- Troubleshooting menos explícito
- Formatos CSV/Human eliminados sin migración documentada

**Evidencia específica**:
```markdown
## Agentic Features (CLI v0.2.0)

- **TOON Format**: Native support for Tabular Object Notation. ~40% token savings.
- **Safety Tripwire**: Protheus queries > 500 rows → auto-diverted to temp `.toon` file
- **Auto-Healing**: Confluence client handles 429/50x with exponential backoff
- **High Concurrency**: Worker pools parallelize heavy exports (5-10x faster)
- **Prefix Validation**: Queries must start with SELECT/WITH/EXEC
- **Early Cursor Closure**: `--max-rows` aborts DB processing at server level
```

**Ganador README**: **gem** — Mejor organización, features modernas documentadas, optimizado para consumo por LLMs.

---

## VAL-DOCS-002: Skills Comparados

### Tabla de Skills por Archivo

| Skill | main | gem | Ganador | Diferencias Clave |
|-------|------|-----|---------|-------------------|
| **mapj/SKILL.md** | 2.1.0 | 2.1.0 | **gem** | TOON default, output modes simplificados |
| **mapj-tdn-search/SKILL.md** | 2.0.0 | 2.1.0 | **gem** | Auto-Pagination, `--max-results` flag, schema optimizado |
| **mapj-confluence-export/SKILL.md** | 2.0.0 | 2.1.0 | **gem** | Auto-Healing, Worker Pool, retry-failed removido (internal) |
| **mapj-protheus-query/SKILL.md** | 3.0.0 | 3.2.0 | **gem** | Schema Discovery, Safety Tripwire, Prefix Validation, Early Cursor Closure |

### Análisis por Skill

#### mapj/SKILL.md (Main Orchestrator)

| Aspecto | main | gem |
|---------|------|-----|
| **Output modes** | `llm` (default), `json`, `csv` | `auto` (default), `llm`, `toon`, `json` |
| **TOON format** | ❌ No documentado | ✅ Documentado con ejemplos |
| **Error envelope** | Con `hint` | Con `hint` (igual) |
| **Sub-skill index** | 4 skills | 4 skills (igual) |

**Ganador**: **gem** — TOON como default optimiza tokens para LLMs.

---

#### mapj-tdn-search/SKILL.md

| Aspecto | main | gem |
|---------|------|-----|
| **Versión** | 2.0.0 | 2.1.0 |
| **Auto-Pagination** | ❌ No documentado | ✅ "Native Auto-Pagination to reach desired result count" |
| **Flag name** | `--limit` | `--max-results` (mejor semántica) |
| **Output schema** | Campos: id, type, title, url, space, labels, ancestors, version, lastUpdated, lastUpdatedBy, childCount | Campos: id, title, url, childCount (optimizado para tokens) |
| **Check-children warning** | ⚠️ Básico | ✅ "TRAMPA CRÍTICA — childCount ≠ total de páginas del árbol" con ejemplo detallado |

**Evidencia gem**:
```markdown
> ⚠️ **TRAMPA CRÍTICA — `childCount` ≠ total de páginas del árbol**
> `childCount` cuenta solo los hijos **directos** (un nivel).
> Una página con `childCount: 1` puede tener **171 páginas** en total
```

**Ganador**: **gem** — Auto-Pagination, output schema optimizado, warnings más explícitos.

---

#### mapj-confluence-export/SKILL.md

| Aspecto | main | gem |
|---------|------|-----|
| **Versión** | 2.0.0 | 2.1.0 |
| **retry-failed** | ✅ Documentado | ❌ Removido (auto-retry interno) |
| **Auto-Healing** | ❌ | ✅ "Native exponential backoff for 429/50x" |
| **Concurrent Worker Pool** | ❌ | ✅ "Worker pools parallelize heavy exports, processing trees 5-10x faster" |
| **Debug flags** | `--debug`, `--dump-debug` | No documentados (eliminados) |

**Cambio filosófico**: main documenta retry manual; gem hace retry automático y no expone el comando.

**Ganador**: **gem** — Auto-Healing y Worker Pool son mejoras reales, aunque pierde flags de debug.

---

#### mapj-protheus-query/SKILL.md

| Aspecto | main | gem |
|---------|------|-----|
| **Versión** | 3.0.0 | 3.2.0 |
| **Schema Discovery** | ❌ | ✅ Comando `mapj protheus schema <table>` documentado |
| **Safety Tripwire** | ❌ | ✅ "If result > 500 rows → auto-save to temp .toon file" |
| **Prefix Validation** | "DML/DDL is blocked" | "Prefix-based validation (SELECT, WITH, EXEC)" |
| **Early Cursor Closure** | ❌ | ✅ "`--max-rows` aborts DB processing at server level" |
| **Formatos output** | `json`, `csv` | `auto`, `llm`, `toon` (CSV eliminado) |

**Evidencia gem**:
```markdown
## Schema Discovery

AI Agents should always check the table structure before querying to avoid hallucinations:

\`\`\`bash
# Get columns and types for a table
mapj protheus schema SA1010
\`\`\`
```

**Ganador**: **gem** — Schema Discovery, Safety Tripwire, y Early Cursor Closure son features valiosas bien documentadas.

---

### Resumen Skills

| Skill | Ganador | Razón principal |
|-------|---------|-----------------|
| mapj/SKILL.md | **gem** | TOON default, output optimizado |
| mapj-tdn-search/SKILL.md | **gem** | Auto-Pagination, schema optimizado, warnings mejorados |
| mapj-confluence-export/SKILL.md | **gem** | Auto-Healing, Worker Pool (aunque pierde debug flags) |
| mapj-protheus-query/SKILL.md | **gem** | Schema Discovery, Safety Tripwire, Early Cursor Closure |

**Ganador Skills**: **gem** (4/4 skills superiores)

---

## VAL-DOCS-003: Documentación Técnica Comparada

### Tabla Comparativa por Archivo

| Archivo | main | gem | Ganador | Diferencias Clave |
|---------|------|-----|---------|-------------------|
| **CONTRIBUTING.md** | 8.5/10 | 8.0/10 | **main** | Known Gotchas extensivo (11 vs 6), troubleshooting detallado |
| **confluence-export-guide.md** | 7.5/10 | 8.5/10 | **gem** | Más conciso, Auto-Healing documentado, retry-failed removido |
| **protheus-guide.md** | 7.5/10 | 9.0/10 | **gem** | Schema Discovery, Safety Tripwire, Early Cursor Closure |

### Análisis Detallado

#### CONTRIBUTING.md

| Aspecto | main | gem | Ganador |
|---------|------|-----|---------|
| **Architecture Overview** | Detallado con todos los archivos | Simplificado | **empate** |
| **Known Gotchas** | 11 items extensivos | 6 items (Design Decisions) | **main** |
| **Data Flows** | 3 diagramas detallados | 2 diagramas simplificados | **main** |
| **Design Decisions** | ❌ | ✅ 4 decisions documentadas | **gem** |
| **How to Add Commands** | Extensivo | Simplificado | **main** |
| **Testing** | Extensivo con estructura de tests | Simplificado | **main** |

**Gotchas en main que gem no tiene**:
1. CSV format does not escape commas
2. `--max-rows` is client-side (gem: server-side Early Cursor Closure)
3. Credentials file is machine-bound
4. `export_view` → `storage` fallback
5. WAF bypass for tdn.totvs.com

**Design Decisions en gem que main no tiene**:
1. Prefix-based SQL Validation
2. Early Cursor Closure
3. TOON (Tabular Object Notation)
4. Auto-Healing Client

**Ganador CONTRIBUTING**: **main** — Known Gotchas extensivo es invaluable para debugging, aunque gem tiene mejor Design Decisions.

---

#### confluence-export-guide.md

| Aspecto | main | gem |
|---------|------|-----|
| **Verbosidad** | Alta (más texto) | Media (más conciso) |
| **retry-failed** | ✅ Sección completa | ❌ Removido |
| **Auto-Healing** | ❌ | ✅ "Ya no existe retry-failed. La CLI reintenta automáticamente" |
| **Worker Pool** | ❌ | ✅ "Concurrent Worker Pool (10 workers)" |
| **Debug flags** | `--debug`, `--dump-debug` documentados | No documentados |

**Cambio clave**: gem explica que retry-failed ya no existe porque Auto-Healing lo hace internamente.

**Ganador**: **gem** — Documentación más enfocada, Auto-Healing y Worker Pool son mejoras reales.

---

#### protheus-guide.md

| Aspecto | main | gem |
|---------|------|-----|
| **Schema Discovery** | ❌ | ✅ Sección nueva con `mapj protheus schema` |
| **Safety Tripwire** | ❌ | ✅ Sección "Protección del Context Window" |
| **SQL Validation** | Keywords bloqueados listados | Prefix-based validation explicado |
| **`--max-rows`** | "Client-side row cap" | "Cierra el cursor de DB temprano" |
| **Output file** | Documentado | Documentado + Safety Tripwire automático |

**Evidencia gem**:
```markdown
## 4. Protección del Context Window (Tripwire)

Si el agente se olvida de poner un `TOP` y la consulta trae más de **500 filas**, 
la CLI hará un **auto-fallback**:

1. Intercepta la salida masiva
2. La guarda en un archivo temporal
3. Avisa por `stderr`
4. Devuelve un resumen compacto por `stdout`
```

**Ganador**: **gem** — Schema Discovery, Safety Tripwire, y Early Cursor Closure son features críticas bien documentadas.

---

### Ganador Global Documentación Técnica

| Criterio | main | gem | Ganador |
|----------|------|-----|---------|
| **CONTRIBUTING.md** | 8.5/10 | 8.0/10 | **main** |
| **confluence-export-guide.md** | 7.5/10 | 8.5/10 | **gem** |
| **protheus-guide.md** | 7.5/10 | 9.0/10 | **gem** |
| **Promedio** | 7.8/10 | 8.5/10 | **gem** |

**Ganador**: **gem** — 2/3 guías superiores, promedio más alto.

---

## Breaking Changes en Documentación (gem)

| Cambio | Impacto | Documentación |
|--------|---------|---------------|
| CSV format eliminado | 🟡 Usuarios existentes pierden opción | ❌ No documentado en README/guías |
| Human format eliminado | 🟡 Usuarios existentes pierden opción | ❌ No documentado |
| retry-failed removido | 🟡 Scripts existentes rompen | ✅ Explicado en guide (auto-retry interno) |
| `--limit` → `--max-results` | 🟡 Breaking change en flag | ❌ No documentado como breaking |
| `--debug` / `--dump-debug` removidos | 🟢 Menor | ❌ No documentado |

**Recomendación**: Documentar breaking changes en CHANGELOG con guía de migración.

---

## Ganador Global para Documentación

### Puntuación Consolidada

| Criterio | main | gem | Ganador |
|----------|------|-----|---------|
| **README.md** | 7.5/10 | 9.0/10 | **gem** |
| **Skills (4 archivos)** | 7.0/10 | 9.0/10 | **gem** |
| **CONTRIBUTING.md** | 8.5/10 | 8.0/10 | **main** |
| **confluence-export-guide.md** | 7.5/10 | 8.5/10 | **gem** |
| **protheus-guide.md** | 7.5/10 | 9.0/10 | **gem** |
| **Promedio Ponderado** | **7.5/10** | **9.0/10** | **gem** |

### 🏆 **Ganador: gem**

**Justificación cuantitativa**:
- README: +1.5 puntos (Agentic Features, TOON)
- Skills: +2.0 puntos (Auto-Pagination, Auto-Healing, Schema Discovery, Safety Tripwire)
- protheus-guide: +1.5 puntos (Schema Discovery, Safety Tripwire, Early Cursor Closure)
- Promedio total: **gem 9.0/10 vs main 7.5/10**

**Justificación cualitativa**:
- ✅ Documentación optimizada para consumo por LLMs (TOON, output schema reducido)
- ✅ Features modernas bien documentadas (Auto-Healing, Worker Pool, Safety Tripwire)
- ✅ Estructura más limpia y enfocada
- ✅ Design Decisions documentadas
- ⚠️ Pierde troubleshooting extensivo de main
- ⚠️ Pierde Known Gotchas detallados
- ⚠️ Breaking changes sin documentación de migración

---

## Recomendaciones

1. **URGENTE**: Documentar breaking changes (CSV/Human eliminados, retry-failed removido, `--limit` → `--max-results`)
2. **IMPORTANTE**: Mover Known Gotchas de main a gem CONTRIBUTING.md (es invaluable para debugging)
3. **SUGERIDO**: Agregar tabla de troubleshooting a README de gem (main la tiene)

---

## Evidencia Específica por Archivo

### README.md
```
main: 173 líneas
gem: 140 líneas (-33 líneas, más conciso)

Agregado en gem: Sección "Agentic Features" (6 features)
Eliminado en gem: Troubleshooting table, Installation prerequisitos
```

### CONTRIBUTING.md
```
main: ~400 líneas con 11 Known Gotchas
gem: ~200 líneas con 6 Design Decisions

Agregado en gem: Design Decisions section
Eliminado en gem: Gotchas detallados, How to Add extensivo, Testing detallado
```

### Skills
```
mapj-protheus-query/SKILL.md:
  main: 3.0.0, sin Schema Discovery, sin Safety Tripwire
  gem: 3.2.0, +Schema Discovery section, +Safety Tripwire section, +Early Cursor Closure
```

### Guides
```
protheus-guide.md:
  main: Sin Schema Discovery, --max-rows client-side
  gem: +Schema Discovery section, +Safety Tripwire section, Early Cursor Closure
```
