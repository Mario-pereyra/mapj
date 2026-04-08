# Evaluación de Skills de mapj_cli

**Assertion cumplida**: VAL-APP-002

**Fecha**: 2026-04-08

---

## Resumen

Este documento evalúa las 4 skills actuales de mapj_cli contra las mejores prácticas documentadas en la guía práctica de diseño de skills. Cada skill es evaluada en dos dimensiones: **Discovery Layer** (frontmatter YAML) y **Execution Layer** (body de SKILL.md), con puntuaciones numéricas e identificación de gaps y mejoras.

---

## 1. Criterios de Evaluación

### 1.1 Discovery Layer (40 puntos máximo)

| Criterio | Puntuación | Límite | Justificación |
|----------|------------|--------|----------------|
| **name** | 0-10 | 3-64 chars, lowercase + hyphens | Identificador único para discovery |
| **description** | 0-10 | max 1024 chars, incluye QUÉ, CUÁNDO, TRIGGERS | Crítico para discovery automático |
| **keywords** | 0-5 | 5-10 items | Mejora matching semántico |
| **version** | 0-5 | semver | Tracking de cambios |
| **author** | 0-5 | texto | Responsabilidad |
| **metadata extra** | 0-5 | tags, capabilities, related | Información adicional útil |

### 1.2 Execution Layer (50 puntos máximo)

| Criterio | Puntuación | Límite | Justificación |
|----------|------------|--------|----------------|
| **Role** | 0-10 | 50-100 chars | Define expertise del agente |
| **Procedure** | 0-10 | 3-7 pasos concretos | Guía de ejecución |
| **Examples** | 0-10 | 2-5 ejemplos | Demuestra output esperado |
| **Success Criteria** | 0-10 | 3-5 criterios | Define cuándo es correcto |
| **Limitations** | 0-5 | Gestiona expectativas | Previene uso incorrecto |
| **Error Handling** | 0-5 | Casos de error | Recuperación ante fallos |
| **Body Length** | 0-10 | <500 líneas | Eficiencia de contexto |

### 1.3 Anti-patrones (10 puntos máximo)

| Criterio | Puntuación | Justificación |
|----------|------------|----------------|
| **No God Skill** | 0-3 | Una skill = un dominio/workflow |
| **Descripciones claras** | 0-3 | No vagas |
| **Sin contradicciones** | 0-2 | Instrucciones coherentes |
| **Sin over-engineering** | 0-2 | Complejidad justificada |

### 1.4 Escala de Puntuación Total

| Rango | Calificación | Interpretación |
|-------|---------------|----------------|
| 90-100 | ⭐⭐⭐⭐⭐ Excelente | Cumple todas las mejores prácticas |
| 75-89 | ⭐⭐⭐⭐ Muy Bueno | Cumple la mayoría, mejoras menores |
| 60-74 | ⭐⭐⭐ Bueno | Cumple lo esencial, gaps identificables |
| 45-59 | ⭐⭐ Mejorable | Gaps significativos, mejoras necesarias |
| 0-44 | ⭐ Insuficiente | No cumple estándares |

---

## 2. Evaluación de Skills

### 2.1 Skill: `mapj` (Orchestrator)

**Ubicación**: `skills/mapj/SKILL.md`

#### Discovery Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **name** | 10/10 | ✅ `mapj` - 4 chars, lowercase, cumple estándar |
| **description** | 9/10 | ✅ Completa: incluye QUÉ hace, CUÁNDO usar, Do NOT use, Triggers. Ligeramente larga pero informativa |
| **keywords** | 0/5 | ❌ No tiene campo `keywords`. Usa `metadata.tags` en su lugar |
| **version** | 5/5 | ✅ `2.1.0` - semver correcto |
| **author** | 5/5 | ✅ `Mario Pereira` |
| **metadata extra** | 5/5 | ✅ Excelente: incluye `compatibility`, `language`, `license`, `tags`, `capabilities`, `related`, `allowed-tools` |

**Subtotal Discovery**: 34/40

#### Execution Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **Role** | 0/10 | ❌ **AUSENTE**. No hay sección que defina el expertise del agente |
| **Procedure** | 6/10 | ⚠️ Tiene estructura de routing (Step 1, Step 2) pero no es un procedimiento de ejecución típico. Cumple su función como skill orchestrator |
| **Examples** | 0/10 | ❌ **AUSENTE**. No hay sección de ejemplos con input/output esperado |
| **Success Criteria** | 0/10 | ❌ **AUSENTE**. No define cuándo la ejecución es exitosa |
| **Limitations** | 5/5 | ✅ Sección "Limitations (Non-Negotiable)" clara y específica |
| **Error Handling** | 5/5 | ✅ Excelente: "Global Error Handling Pattern" y "Exit Codes" bien documentados |
| **Body Length** | 10/10 | ✅ ~180 líneas - bien dentro del límite de 500 |

**Subtotal Execution**: 26/50

#### Anti-patrones

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **No God Skill** | 3/3 | ✅ Es un router skill, no una God Skill. Delega a sub-skills |
| **Descripciones claras** | 3/3 | ✅ Description es específica y detallada |
| **Sin contradicciones** | 2/2 | ✅ Instrucciones coherentes |
| **Sin over-engineering** | 2/2 | ✅ Complejidad apropiada para CLI orchestration |

**Subtotal Anti-patrones**: 10/10

#### Puntuación Total

| Dimensión | Puntuación |
|-----------|------------|
| Discovery Layer | 34/40 |
| Execution Layer | 26/50 |
| Anti-patrones | 10/10 |
| **TOTAL** | **70/100** |

**Calificación**: ⭐⭐⭐ **Bueno**

#### Gaps Identificados

| Gap | Severidad | Impacto |
|-----|-----------|---------|
| Sin sección `Role` | ⚠️ MEDIO | Agente no tiene definición de expertise |
| Sin sección `Examples` | ⚠️ MEDIO | No hay few-shot examples para guiar output |
| Sin sección `Success Criteria` | ⚠️ MEDIO | No define cuándo la skill se ejecutó correctamente |
| Usa `tags` en vez de `keywords` | ⚠️ MENOR | Discovery puede ser menos eficiente |

#### Recomendaciones

1. **Añadir sección Role**: `Act as a TOTVS ecosystem integration specialist with expertise in Protheus ERP, Confluence, and TDN documentation.`
2. **Añadir 2-3 ejemplos**: Mostrar flujo típico de invocación
3. **Añadir Success Criteria**: Definir cuándo el routing fue exitoso
4. **Migrar `tags` a `keywords`**: Para cumplir estándar emergente

---

### 2.2 Skill: `mapj-confluence-export`

**Ubicación**: `skills/mapj-confluence-export/SKILL.md`

#### Discovery Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **name** | 10/10 | ✅ `mapj-confluence-export` - 21 chars, lowercase + hyphens |
| **description** | 9/10 | ✅ Completa: incluye QUÉ, CUÁNDO, Do NOT use, Triggers |
| **keywords** | 0/5 | ❌ No tiene campo `keywords`. Usa `metadata.tags` |
| **version** | 5/5 | ✅ `2.1.0` |
| **author** | 5/5 | ✅ `Mario Pereira` |
| **metadata extra** | 5/5 | ✅ Incluye `compatibility`, `tags`, `capabilities`, `related`, `allowed-tools` |

**Subtotal Discovery**: 34/40

#### Execution Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **Role** | 0/10 | ❌ **AUSENTE**. No hay definición de expertise |
| **Procedure** | 7/10 | ⚠️ Tiene "Export Workflow" con decisiones, pero no pasos numerados típicos |
| **Examples** | 0/10 | ❌ **AUSENTE**. No hay ejemplos de input/output |
| **Success Criteria** | 0/10 | ❌ **AUSENTE** |
| **Limitations** | 5/5 | ✅ Sección "What This Skill Will NOT Do" clara |
| **Error Handling** | 5/5 | ✅ "Error Reference" con tabla de errores y fixes |
| **Body Length** | 10/10 | ✅ ~130 líneas |

**Subtotal Execution**: 27/50

#### Anti-patrones

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **No God Skill** | 3/3 | ✅ Single domain (Confluence), single workflow (export) |
| **Descripciones claras** | 3/3 | ✅ Específica y detallada |
| **Sin contradicciones** | 2/2 | ✅ Instrucciones coherentes |
| **Sin over-engineering** | 2/2 | ✅ Complejidad apropiada |

**Subtotal Anti-patrones**: 10/10

#### Puntuación Total

| Dimensión | Puntuación |
|-----------|------------|
| Discovery Layer | 34/40 |
| Execution Layer | 27/50 |
| Anti-patrones | 10/10 |
| **TOTAL** | **71/100** |

**Calificación**: ⭐⭐⭐ **Bueno**

#### Gaps Identificados

| Gap | Severidad | Impacto |
|-----|-----------|---------|
| Sin sección `Role` | ⚠️ MEDIO | Agente no tiene expertise definido |
| Sin sección `Examples` | ⚠️ ALTO | Few-shot examples críticos para tasks de export |
| Sin sección `Success Criteria` | ⚠️ MEDIO | No define output correcto |
| Usa `tags` en vez de `keywords` | ⚠️ MENOR | Discovery menos estándar |

#### Recomendaciones

1. **Añadir Role**: `Act as a Confluence export specialist with expertise in Markdown conversion and document structure.`
2. **Añadir 3-4 ejemplos**: 
   - Single page export
   - Recursive export with descendants
   - Space export
   - Error case handling
3. **Añadir Success Criteria**: Output estructura correcta, archivos generados, manifest válido
4. **Migrar a `keywords`**

---

### 2.3 Skill: `mapj-protheus-query`

**Ubicación**: `skills/mapj-protheus-query/SKILL.md`

#### Discovery Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **name** | 10/10 | ✅ `mapj-protheus-query` - 19 chars, lowercase + hyphens |
| **description** | 9/10 | ✅ Muy completa: incluye QUÉ, CUÁNDO, Do NOT use, Triggers extensivos |
| **keywords** | 0/5 | ❌ No tiene campo `keywords`. Usa `metadata.tags` |
| **version** | 5/5 | ✅ `3.2.0` |
| **author** | 5/5 | ✅ `Mario Pereira` |
| **metadata extra** | 5/5 | ✅ Excelente: incluye `security` con políticas específicas |

**Subtotal Discovery**: 34/40

#### Execution Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **Role** | 0/10 | ❌ **AUSENTE** |
| **Procedure** | 8/10 | ✅ Tiene "Connection Management Workflow" y "Query Workflow" bien estructurados |
| **Examples** | 3/10 | ⚠️ Tiene "Essential queries" pero no son ejemplos con formato input/output esperado |
| **Success Criteria** | 0/10 | ❌ **AUSENTE** |
| **Limitations** | 5/5 | ✅ Sección de seguridad y limitaciones clara |
| **Error Handling** | 5/5 | ✅ "Error Reference" completo |
| **Body Length** | 10/10 | ✅ ~130 líneas |

**Subtotal Execution**: 31/50

#### Anti-patrones

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **No God Skill** | 3/3 | ✅ Single domain (Protheus), dos workflows relacionados (query + connection mgmt) |
| **Descripciones claras** | 3/3 | ✅ Muy específica |
| **Sin contradicciones** | 2/2 | ✅ Instrucciones coherentes |
| **Sin over-engineering** | 2/2 | ✅ Complejidad apropiada |

**Subtotal Anti-patrones**: 10/10

#### Puntuación Total

| Dimensión | Puntuación |
|-----------|------------|
| Discovery Layer | 34/40 |
| Execution Layer | 31/50 |
| Anti-patrones | 10/10 |
| **TOTAL** | **75/100** |

**Calificación**: ⭐⭐⭐⭐ **Muy Bueno**

#### Gaps Identificados

| Gap | Severidad | Impacto |
|-----|-----------|---------|
| Sin sección `Role` | ⚠️ MEDIO | Expertise no definido |
| Examples no estructurados | ⚠️ MEDIO | No siguen formato few-shot estándar |
| Sin Success Criteria | ⚠️ MEDIO | No define éxito |
| Usa `tags` vs `keywords` | ⚠️ MENOR | Discovery no estándar |

#### Recomendaciones

1. **Añadir Role**: `Act as a Protheus ERP database analyst with expertise in Totvs SQL Server schemas and business data retrieval.`
2. **Reformatear Examples**: Convertir "Essential queries" a formato few-shot con input/output:
   ```markdown
   ### Example 1: Simple customer query
   **User request:** "Show me customers from São Paulo"
   **Expected output:**
   ```sql
   SELECT A1_COD, A1_NOME, A1_EST FROM SA1010 WHERE A1_EST = 'SP'
   ```
   ```
3. **Añadir Success Criteria**: Query válido, resultado correcto, sin inyección SQL
4. **Migrar a `keywords`**

---

### 2.4 Skill: `mapj-tdn-search`

**Ubicación**: `skills/mapj-tdn-search/SKILL.md`

#### Discovery Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **name** | 10/10 | ✅ `mapj-tdn-search` - 15 chars, lowercase + hyphens |
| **description** | 9/10 | ✅ Completa: incluye QUÉ, CUÁNDO, Do NOT use, Triggers |
| **keywords** | 0/5 | ❌ No tiene campo `keywords`. Usa `metadata.tags` |
| **version** | 5/5 | ✅ `2.1.0` |
| **author** | 5/5 | ✅ `Mario Pereira` |
| **metadata extra** | 5/5 | ✅ Incluye `capabilities`, `related`, `allowed-tools` |

**Subtotal Discovery**: 34/40

#### Execution Layer

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **Role** | 0/10 | ❌ **AUSENTE** |
| **Procedure** | 7/10 | ⚠️ Tiene "Quick Routing" y "Core Commands" pero no es un procedimiento paso a paso |
| **Examples** | 4/10 | ⚠️ Tiene "Output Schema" que muestra formato, pero no ejemplos input/output |
| **Success Criteria** | 0/10 | ❌ **AUSENTE** |
| **Limitations** | 0/5 | ❌ **AUSENTE**. No hay sección de limitaciones |
| **Error Handling** | 5/5 | ✅ "Error Reference" presente |
| **Body Length** | 10/10 | ✅ ~120 líneas |

**Subtotal Execution**: 26/50

#### Anti-patrones

| Criterio | Puntuación | Hallazgo |
|----------|------------|----------|
| **No God Skill** | 3/3 | ✅ Single domain (TDN), single workflow (search) |
| **Descripciones claras** | 3/3 | ✅ Específica |
| **Sin contradicciones** | 2/2 | ✅ Coherente |
| **Sin over-engineering** | 2/2 | ✅ Apropiada |

**Subtotal Anti-patrones**: 10/10

#### Puntuación Total

| Dimensión | Puntuación |
|-----------|------------|
| Discovery Layer | 34/40 |
| Execution Layer | 26/50 |
| Anti-patrones | 10/10 |
| **TOTAL** | **70/100** |

**Calificación**: ⭐⭐⭐ **Bueno**

#### Gaps Identificados

| Gap | Severidad | Impacto |
|-----|-----------|---------|
| Sin sección `Role` | ⚠️ MEDIO | Expertise no definido |
| Sin Examples estructurados | ⚠️ ALTO | No hay few-shot para search tasks |
| Sin Success Criteria | ⚠️ MEDIO | Output correcto no definido |
| Sin Limitations | ⚠️ MEDIO | Expectativas no gestionadas |
| Usa `tags` vs `keywords` | ⚠️ MENOR | Discovery no estándar |

#### Recomendaciones

1. **Añadir Role**: `Act as a TDN documentation specialist with expertise in TOTVS knowledge base search and CQL queries.`
2. **Añadir 3-4 ejemplos**:
   - Búsqueda simple por keyword
   - Búsqueda con filtros (space + label)
   - Búsqueda con fecha (--since)
   - Search → Export pipeline
3. **Añadir Success Criteria**: Resultados relevantes, CQL válido, paginación correcta
4. **Añadir Limitations**: Rate limits, auth opcional, máximo resultados
5. **Migrar a `keywords`**

---

## 3. Tabla Comparativa de Skills

| Skill | Discovery | Execution | Anti-patrones | **Total** | Calificación |
|-------|-----------|-----------|---------------|-----------|--------------|
| mapj-protheus-query | 34/40 | 31/50 | 10/10 | **75/100** | ⭐⭐⭐⭐ Muy Bueno |
| mapj-confluence-export | 34/40 | 27/50 | 10/10 | **71/100** | ⭐⭐⭐ Bueno |
| mapj-tdn-search | 34/40 | 26/50 | 10/10 | **70/100** | ⭐⭐⭐ Bueno |
| mapj (orchestrator) | 34/40 | 26/50 | 10/10 | **70/100** | ⭐⭐⭐ Bueno |

### Fortalezas Comunes

1. **Discovery Layer sólido**: Todas tienen name, description, version, author correctos
2. **Sin anti-patrones**: Ninguna es God Skill, todas tienen descripciones claras
3. **Error handling**: Todas tienen secciones de error reference
4. **Body length eficiente**: Todas <200 líneas

### Gaps Comunes

1. **Sin sección Role**: 4/4 skills no tienen expertise definido
2. **Sin Success Criteria**: 4/4 skills no definen cuándo son exitosas
3. **Keywords vs tags**: 4/4 skills usan `metadata.tags` en vez del campo estándar `keywords`
4. **Examples inconsistentes**: Solo mapj-protheus-query tiene ejemplos parciales

---

## 4. Matriz de Mejoras Priorizadas

### Prioridad ALTA (impacta discovery y output quality)

| Skill | Mejora | Impacto | Esfuerzo |
|-------|--------|---------|----------|
| Todas | Añadir sección `Role` | Alto | Bajo |
| Todas | Migrar `tags` → `keywords` | Medio | Bajo |
| mapj-tdn-search | Añadir Examples few-shot | Alto | Medio |
| mapj-confluence-export | Añadir Examples few-shot | Alto | Medio |

### Prioridad MEDIA (impacta consistencia y mantenibilidad)

| Skill | Mejora | Impacto | Esfuerzo |
|-------|--------|---------|----------|
| Todas | Añadir Success Criteria | Medio | Bajo |
| mapj-tdn-search | Añadir Limitations | Medio | Bajo |
| mapj-protheus-query | Reformatear Examples | Medio | Bajo |

### Prioridad BAJA (nice-to-have)

| Skill | Mejora | Impacto | Esfuerzo |
|-------|--------|---------|----------|
| Todas | Añadir versión de Protheus tables en references | Bajo | Medio |

---

## 5. Checklist de Mejora Sugerida

### Template de Role (aplicar a todas)

```markdown
## Role

Act as a [DOMINIO] specialist with expertise in [TECNOLOGÍA/CONTEXTO].
```

**Ejemplos por skill**:
- `mapj`: "Act as a TOTVS ecosystem integration specialist with expertise in Protheus ERP, Confluence, and TDN documentation."
- `mapj-confluence-export`: "Act as a Confluence export specialist with expertise in Markdown conversion and document structure."
- `mapj-protheus-query`: "Act as a Protheus ERP database analyst with expertise in Totvs SQL Server schemas."
- `mapj-tdn-search`: "Act as a TDN documentation specialist with expertise in TOTVS knowledge base search."

### Template de Success Criteria (aplicar a todas)

```markdown
## Success Criteria

The skill execution is successful when:
- [CRITERIO 1 - output correcto]
- [CRITERIO 2 - validaciones]
- [CRITERIO 3 - manejo de errores]
```

### Template de Keywords (migrar de tags)

```yaml
---
name: mapj-protheus-query
description: ...
keywords:               # AÑADIR este campo
  - protheus
  - query
  - erp
  - totvs
  - database
  - sql
  - mssql
---
```

---

## 6. Conclusión

### Resumen de Evaluación

| Métrica | Valor |
|---------|-------|
| **Promedio total** | 71.5/100 |
| **Mejor skill** | mapj-protheus-query (75/100) |
| **Skill con más gaps** | mapj-tdn-search (sin Limitations) |
| **Gaps más comunes** | Role (100%), Success Criteria (100%), Keywords (100%) |

### Fortalezas del diseño actual

1. **Arquitectura modular**: El patrón de orchestrator + sub-skills es excelente
2. **Discovery Layer completo**: Descripciones detalladas con triggers
3. **Sin anti-patrones**: Ninguna skill es God Skill ni tiene instrucciones contradictorias
4. **Error handling robusto**: Todas manejan errores explícitamente

### Áreas de mejora críticas

1. **Sección Role**: Añadir a todas las skills para definir expertise
2. **Few-shot Examples**: Añadir 2-5 ejemplos estructurados a cada skill
3. **Success Criteria**: Definir qué constituye una ejecución exitosa
4. **Keywords estándar**: Migrar de `metadata.tags` a campo `keywords` de primer nivel

### Próximos pasos recomendados

1. **Inmediato**: Aplicar template de Role y migrar keywords (bajo esfuerzo, alto impacto)
2. **Corto plazo**: Añadir Success Criteria y reformatear Examples
3. **Medio plazo**: Añadir Limitations a skills que no tienen

---

**Validación**: Este documento cumple con VAL-APP-002:
- ✅ Evaluación de cada skill existente (4 skills evaluadas)
- ✅ Puntuaciones contra mejores prácticas (escala 0-100)
- ✅ Identificación de gaps y mejoras (tabla priorizada)
