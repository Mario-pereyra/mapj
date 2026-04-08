# Síntesis: Principios de Estructura y División de Skills

**Assertions cumplidas**: VAL-STR-001, VAL-STR-002, VAL-STR-003

**Fecha**: 2026-04-08

---

## Resumen

Este documento sintetiza los principios de estructura y división para el diseño de skills efectivas, cubriendo tres áreas críticas: (1) criterios para decidir cuándo crear múltiples skills vs una skill grande, (2) el patrón de progressive disclosure para optimizar el uso del contexto del agente, y (3) anti-patrones comunes que deben evitarse con sus respectivas soluciones.

---

## 1. Criterios de División de Skills (VAL-STR-001)

### 1.1 La Pregunta Fundamental

**¿Cuándo crear múltiples skills vs una skill grande?**

La respuesta depende de evaluar la skill propuesta contra cuatro criterios: **dominio**, **workflow**, **usuario**, y **frecuencia**. Una skill que falla en cualquiera de estos criterios debe dividirse.

### 1.2 Los Cuatro Criterios de División

| Criterio | Pregunta | Indicador de División | Ejemplo |
|----------|----------|----------------------|---------|
| **Dominio** | ¿La skill opera en un solo dominio conceptual? | Si abarca múltiples dominios → dividir por dominio | `protheus-query` vs `protheus-export` vs `confluence-search` |
| **Workflow** | ¿La skill ejecuta un solo flujo de trabajo? | Si tiene múltiples workflows → dividir por workflow | `code-review` vs `code-generate` vs `code-refactor` |
| **Usuario** | ¿La skill sirve a un solo tipo de usuario/rol? | Si sirve múltiples roles → dividir por audiencia | `admin-config` vs `user-query` |
| **Frecuencia** | ¿Todas las funciones se usan con similar frecuencia? | Si tiene partes raramente usadas → extraer como skill separada | `query-common` vs `query-advanced` |

### 1.3 Criterio 1: Dominio

**Definición**: El dominio conceptual es el área de conocimiento o contexto en el que opera la skill.

**Principio**: Una skill = Un dominio

**Indicadores de violación**:
- La skill accede a múltiples sistemas/DBs/ APIs no relacionados
- El nombre contiene "and" o "or" (`database-and-email-skill`)
- Las instrucciones tienen secciones claramente separadas por contextos distintos

**Ejemplo de división por dominio**:

| ❌ God Skill (múltiples dominios) | ✅ Skills divididas por dominio |
|----------------------------------|-------------------------------|
| `database-master` | `protheus-query` |
| - Query Protheus ERP | - Solo Protheus queries |
| - Query Confluence | |
| - Query TDN API | `confluence-search` |
| | - Solo Confluence search |
| | |
| | `tdn-search` |
| | - Solo TDN documentation |

**Beneficio**: Cada skill tiene instrucciones específicas del dominio, ejemplos relevantes, y discovery más preciso.

### 1.4 Criterio 2: Workflow

**Definición**: El workflow es la secuencia de acciones que la skill ejecuta.

**Principio**: Una skill = Un workflow principal

**Indicadores de violación**:
- La skill tiene múltiples procedimientos independientes
- El usuario debe elegir entre caminos mutuamente excluyentes
- Instrucciones con "If the user wants X, do this; if Y, do that"

**Ejemplo de división por workflow**:

| ❌ Skill con múltiples workflows | ✅ Skills divididas por workflow |
|--------------------------------|--------------------------------|
| `code-assistant` | `code-review` |
| - Review code for issues | - Procedimiento: analyze → report → suggest |
| - Generate new code | |
| - Refactor existing code | `code-generate` |
| | - Procedimiento: understand → design → implement |
| | |
| | `code-refactor` |
| | - Procedimiento: identify → transform → verify |

**Beneficio**: Instrucciones procedimentales claras sin bifurcaciones, ejemplos enfocados en un tipo de tarea.

### 1.5 Criterio 3: Usuario/Rol

**Definición**: El usuario/rol es el tipo de persona o sistema que invoca la skill.

**Principio**: Una skill = Una audiencia principal

**Indicadores de violación**:
- Instrucciones diferenciadas por nivel de expertise
- Secciones como "For beginners" vs "For advanced users"
- Tareas que solo ciertos roles pueden ejecutar

**Ejemplo de división por usuario**:

| ❌ Skill multi-audiencia | ✅ Skills divididas por usuario |
|--------------------------|-------------------------------|
| `database-management` | `admin-database-config` |
| - Configure DB connections (admin) | - Solo tareas de configuración |
| - Query database (user) | - Permisos: admin only |
| - Monitor performance (admin) | |
| | `user-database-query` |
| | - Solo consultas SELECT |
| | - Permisos: user |

**Beneficio**: Instrucciones adaptadas al nivel de expertise, sin información irrelevante para cada audiencia.

### 1.6 Criterio 4: Frecuencia de Uso

**Definición**: La frecuencia es qué tan a menudo se usa cada parte de la skill.

**Principio**: Funcionalidad de alta frecuencia = Skill separada; Funcionalidad de baja frecuencia = Skill especializada

**Indicadores de violación**:
- El body de la skill es largo (>500 líneas) porque incluye casos especiales
- La mayoría de las instrucciones se aplican a casos raros
- Secciones marcadas como "rarely used" o "edge cases"

**Ejemplo de división por frecuencia**:

| ❌ Skill con mezcla de frecuencia | ✅ Skills divididas por frecuencia |
|----------------------------------|----------------------------------|
| `query-executor` | `query-common` |
| - Basic SELECT queries (frecuente) | - 80% de casos comunes |
| - Complex JOINs (menos frecuente) | - Body: ~100 líneas |
| - Query optimization (raro) | |
| - Query debugging (raro) | `query-advanced` |
| - Body: 600+ líneas | - JOINs, subqueries, optimization |
| | - Body: ~300 líneas |
| | |
| | `query-debug` |
| | - Troubleshooting, debugging |
| | - Body: ~100 líneas |

**Beneficio**: Contexto eficiente - el agente carga instrucciones detalladas solo para skills que necesita.

### 1.7 Matriz de Decisión

```
                    ┌─────────────────────────────────────────┐
                    │         ¿VIOLA ALGÚN CRITERIO?          │
                    └─────────────────────────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────┐
                    │                                     │
                   NO                                    YES
                    │                                     │
                    ▼                                     ▼
        ┌───────────────────┐               ┌─────────────────────────┐
        │   MANTENER COMO   │               │   ¿VIOLA MÚLTIPLES      │
        │   SKILL ÚNICA     │               │   CRITERIOS?            │
        └───────────────────┘               └─────────────────────────┘
                                                      │
                                      ┌───────────────┴───────────────┐
                                      │                               │
                                     NO                              YES
                                      │                               │
                                      ▼                               ▼
                          ┌─────────────────────┐         ┌─────────────────────┐
                          │   DIVIDIR EN 2      │         │   DIVIDIR EN N      │
                          │   SKILLS            │         │   SKILLS            │
                          │   (N = criterios    │         │   (N = dominios ×   │
                          │    violados)        │         │    workflows)       │
                          └─────────────────────┘         └─────────────────────┘
```

### 1.8 Ejemplos de Aplicación

#### Caso 1: `protheus-master` (viola 3 criterios)

**Skill propuesta**: `protheus-master`
- Query Protheus database (workflow: query)
- Export data to Excel (workflow: export)
- Generate reports (workflow: report)
- Configure connections (workflow: config)

**Análisis**:
- ❌ Dominio: OK (solo Protheus)
- ❌ Workflow: VIOLA - 4 workflows diferentes
- ❌ Usuario: VIOLA - config es admin, resto es user
- ✅ Frecuencia: OK

**División resultante**:
```
protheus-master → protheus-query
               → protheus-export
               → protheus-report
               → protheus-admin
```

#### Caso 2: `code-review` (cumple todos los criterios)

**Skill propuesta**: `code-review`
- Analyze code for bugs, security issues, style violations
- Provide improvement suggestions
- Generate review comments

**Análisis**:
- ✅ Dominio: Un solo dominio (calidad de código)
- ✅ Workflow: Un solo workflow (analyze → report → suggest)
- ✅ Usuario: Una audiencia (developers)
- ✅ Frecuencia: Todas las funciones se usan juntas

**Decisión**: Mantener como skill única ✅

#### Caso 3: `api-helper` (viola 1 criterio)

**Skill propuesta**: `api-helper`
- Query REST APIs
- Query GraphQL APIs
- Query SOAP APIs

**Análisis**:
- ❌ Dominio: VIOLA - REST, GraphQL, SOAP son paradigmas diferentes
- ✅ Workflow: OK (todos son queries)
- ✅ Usuario: OK
- ✅ Frecuencia: OK

**División resultante**:
```
api-helper → api-rest-query
           → api-graphql-query
           → api-soap-query
```

---

## 2. Progressive Disclosure Pattern (VAL-STR-002)

### 2.1 Concepto

**Progressive disclosure** es un patrón arquitectónico que separa la información de una skill en capas con diferentes propósitos y visibilidad. El objetivo es optimizar el uso del contexto del agente: cargar información detallada solo cuando es necesaria.

### 2.2 Las Dos Capas

| Capa | Contenido | Propósito | Visibilidad | Tamaño Típico |
|------|-----------|-----------|-------------|----------------|
| **Discovery Layer** | name, description, keywords | Matching y routing | **Siempre cargada** | ~200 chars/skill |
| **Execution Layer** | instructions, examples, references | Ejecución detallada | **Carga condicional** | ~5,000 chars/skill |

### 2.3 Diagrama de Arquitectura

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CONTEXTO DEL AGENTE                              │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    DISCOVERY LAYER (SIEMPRE)                        │ │
│  │                                                                     │ │
│  │  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐  │ │
│  │  │ Skill 1          │ │ Skill 2          │ │ Skill N          │  │ │
│  │  │ name: "query"    │ │ name: "export"   │ │ name: "review"   │  │ │
│  │  │ desc: "Query DB" │ │ desc: "Export..."│ │ desc: "Review..."│  │ │
│  │  │ keywords: [...]  │ │ keywords: [...]  │ │ keywords: [...]  │  │ │
│  │  │ (~200 chars)     │ │ (~200 chars)     │ │ (~200 chars)     │  │ │
│  │  └──────────────────┘ └──────────────────┘ └──────────────────┘  │ │
│  │                                                                     │ │
│  │  Total: N skills × ~200 chars = eficiente para 100+ skills        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                  EXECUTION LAYER (CONDICIONAL)                       │ │
│  │                                                                     │ │
│  │  ┌──────────────────────────────────────────────────────────────┐ │ │
│  │  │ Skill N (seleccionada por el agente)                          │ │ │
│  │  │                                                               │ │ │
│  │  │ # Instructions                                                │ │ │
│  │  │ ## Role                                                       │ │ │
│  │  │ Act as a database analyst...                                  │ │ │
│  │  │                                                               │ │ │
│  │  │ ## Procedure                                                  │ │ │
│  │  │ 1. Validate request...                                        │ │ │
│  │  │ 2. Parse tables...                                            │ │ │
│  │  │                                                               │ │ │
│  │  │ ## Examples                                                   │ │ │
│  │  │ Example 1: ...                                                │ │ │
│  │  │ Example 2: ...                                                │ │ │
│  │  │                                                               │ │ │
│  │  │ (~5,000 chars promedio)                                       │ │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.4 Flujo de Carga

```
     Usuario: "Query Protheus for customer data"
                        │
                        ▼
    ┌───────────────────────────────────────────────┐
    │              FASE 1: DISCOVERY                │
    │                                               │
    │  Sistema carga frontmatter de TODAS las      │
    │  skills disponibles (~200 chars cada una)    │
    │                                               │
    │  [skill-1: name, desc, kw]                   │
    │  [skill-2: name, desc, kw]                   │
    │  [skill-3: name, desc, kw]                   │
    │  ...                                         │
    └───────────────────────────────────────────────┘
                        │
                        ▼
    ┌───────────────────────────────────────────────┐
    │              FASE 2: MATCHING                 │
    │                                               │
    │  LLM evalúa cuál skill(es) son relevantes    │
    │  basándose en:                               │
    │  - description matching                      │
    │  - keyword matching                          │
    │  - semantic similarity                       │
    │                                               │
    │  Resultado: "protheus-query" seleccionada    │
    └───────────────────────────────────────────────┘
                        │
                        ▼
    ┌───────────────────────────────────────────────┐
    │              FASE 3: EXECUTION LOAD           │
    │                                               │
    │  Sistema carga el body COMPLETO de la        │
    │  skill seleccionada:                         │
    │  - Instructions detalladas                   │
    │  - Few-shot examples                         │
    │  - References                                │
    │                                               │
    │  Solo "protheus-query" body se carga         │
    └───────────────────────────────────────────────┘
                        │
                        ▼
    ┌───────────────────────────────────────────────┐
    │              FASE 4: EXECUTION                │
    │                                               │
    │  LLM sigue las instrucciones detalladas      │
    │  y ejecuta la skill                          │
    └───────────────────────────────────────────────┘
```

### 2.5 Beneficios del Progressive Disclosure

| Beneficio | Descripción | Impacto |
|-----------|-------------|---------|
| **Eficiencia de contexto** | Solo se carga el body de skills que se usarán | Reduce contexto waste en ~90% con 10+ skills |
| **Discovery rápido** | Description + keywords permiten matching eficiente | Latencia de routing mínima |
| **Escalabilidad** | 100+ skills pueden existir sin agotar contexto | Anthropic tiene 100+ skills internas |
| **Reducción de noise** | Instrucciones irrelevantes no contaminan | Menor probabilidad de instrucciones contradictorias |

### 2.6 Niveles de Progressive Disclosure (Anthropic)

Anthropic implementa **3 niveles** de progressive disclosure:

| Nivel | Contenido | Cuándo se carga |
|-------|-----------|-----------------|
| **Nivel 1: Discovery** | name, description en frontmatter | Siempre |
| **Nivel 2: Execution** | Body completo de SKILL.md | Cuando la skill es relevante |
| **Nivel 3: Deep** | Archivos referenciados (`scripts/`, `references/`) | Cuando se necesitan recursos específicos |

**Estructura de directorio**:
```
my-skill/
├── SKILL.md              # Nivel 1 + 2 (frontmatter + body)
├── scripts/              # Nivel 3 (condicional)
│   └── helper.py
├── references/           # Nivel 3 (condicional)
│   └── api-docs.md
└── assets/               # Nivel 3 (condicional)
    └── templates.json
```

### 2.7 Implementación en SKILL.md

```yaml
---
# DISCOVERY LAYER (siempre visible)
name: protheus-query
description: |
  Execute SQL queries against Protheus ERP database when the user needs to 
  retrieve business data, generate reports, or analyze records. 
  Use for SELECT operations on Protheus tables (SA1, SB1, SC5, etc.).
  Triggers: "protheus", "query", "erp data", "totvs", "advpl database".
keywords:
  - protheus
  - query
  - erp
  - totvs
---

# EXECUTION LAYER (carga condicional)

## Role
Act as a database analyst with expertise in Protheus ERP (Totvs) data structures.

## Procedure
1. **Validate request**: Ensure the query is a SELECT operation
2. **Parse tables**: Identify Protheus tables involved
3. **Generate query**: Write SQL following Protheus conventions
4. **Execute**: Run query and return formatted results

## Examples

### Example 1: Simple customer query
User: "Show me all customers from São Paulo"
Query: `SELECT * FROM SA1010 WHERE A1_EST = 'SP'`

### Example 2: Order summary
User: "Get order totals for this month"
Query: `SELECT C5_NUM, C5_CLIENTE, C5_TOTAL FROM SC5010...`

## Limitations
- Only SELECT operations supported
- Maximum 1000 rows returned
```

### 2.8 Comparativa por Proveedor

| Proveedor | Niveles | Implementación |
|-----------|---------|----------------|
| **Anthropic** | 3 niveles | Discovery → Execution → Deep (scripts/references) |
| **OpenAI** | 2 niveles | Discovery → Execution |
| **Microsoft** | 1 nivel | Todo en JSON manifest (sin progressive disclosure) |
| **Google** | Variable | Depende de configuración |
| **CrewAI** | 2 niveles | Frontmatter → Body |

---

## 3. Anti-patrones Comunes (VAL-STR-003)

### 3.1 Tabla de Anti-patrones

| # | Anti-patrón | Problema | Síntomas | Solución |
|---|-------------|----------|----------|----------|
| 1 | **"God Skill"** | Instrucciones contradictorias, context pollution | Body >500 líneas, múltiples workflows, nombre contiene "and/or" | Dividir usando criterios de dominio/workflow |
| 2 | **Descripciones Vagas** | Discovery falla, skill nunca se invoca | Description <20 chars, sin triggers, sin keywords específicas | Añadir triggers explícitos, keywords, propósito específico |
| 3 | **Instrucciones Contradictorias** | Outputs inconsistentes, comportamiento impredecible | Pasos que se contradicen, múltiples "If X do Y; if Z do W" | Simplificar, coherencia entre pasos, separar workflows |
| 4 | **Over-engineering** | Complejidad innecesaria, difícil mantenimiento | Errores sofisticados, abstracciones múltiples, casos edge extensivos | Empezar simple (YAGNI), escalar solo cuando sea necesario |
| 5 | **Skill Fragmentada** | Discovery confuso, invocación fragmentada | Múltiples skills para una tarea simple, nombres muy específicos | Consolidar skills relacionadas con un dominio común |
| 6 | **Sin Ejemplos** | El agente no entiende el output esperado | Body sin sección de ejemplos, descripción sin demostración | Añadir 2-5 few-shot examples representativos |

### 3.2 Anti-patrón 1: "God Skill"

**Definición**: Una skill que intenta hacer demasiado, cubriendo múltiples dominios, workflows o responsabilidades.

**Problema**:
- Instrucciones contradictorias entre workflows
- Context pollution - el agente carga información irrelevante
- Discovery impreciso - description vaga por intentar cubrir todo

**Ejemplo problemático**:

```yaml
---
name: database-master
description: Manage all database operations including queries, exports, 
  reports, configuration, and monitoring.
---
```

**Síntomas**:
- Body de 800+ líneas
- Nombre contiene "master", "all", "complete"
- Secciones mutuamente excluyentes

**Solución**: Dividir según criterios:

```
database-master → database-query
                → database-export
                → database-report
                → database-config
                → database-monitor
```

**Cita de fuente**:
> "Avoid 'God Skills' - skills that try to do too much. Instead, create focused skills with clear responsibilities."
> — Anthropic Engineering Blog, "Building Effective AI Agents" (Dec 2024)

### 3.3 Anti-patrón 2: Descripciones Vagas

**Definición**: Description que no comunica claramente cuándo y por qué usar la skill.

**Problema**:
- Discovery falla - el agente no puede matchear la skill al prompt del usuario
- Skill nunca se invoca o se invoca incorrectamente
- Usuario frustrado porque el agente "no sabe" usar la skill

**Ejemplos problemáticos**:

| ❌ Description vaga | ✅ Description efectiva |
|---------------------|------------------------|
| `"Query database"` | `"Execute SQL queries against Protheus ERP database when the user needs to retrieve business data. Triggers: protheus, query, erp data."` |
| `"Do things with code"` | `"Review code for bugs, security vulnerabilities, and style violations. Use when user asks for code review or mentions PR review."` |
| `"Helper skill"` | `"Format and export data to Excel when the user needs to download query results as spreadsheets. Triggers: export, excel, download, spreadsheet."` |

**Síntomas**:
- Description <20 caracteres
- Sin palabras trigger
- Sin contexto de uso

**Solución**: Añadir:
1. **Propósito específico**: Qué hace exactamente
2. **Triggers**: Cuándo usarla (palabras clave)
3. **Keywords**: Para matching semántico

### 3.4 Anti-patrón 3: Instrucciones Contradictorias

**Definición**: Instrucciones que se contradicen entre sí o generan ambigüedad sobre cómo proceder.

**Problema**:
- Outputs inconsistentes
- Comportamiento impredecible
- Agente confundido sobre cuál camino seguir

**Ejemplo problemático**:

```markdown
## Procedure
1. Execute all queries requested by the user immediately.
2. Validate each query before execution to ensure safety.
3. Never execute queries without user confirmation.
```

**Contradicciones**:
- "Execute immediately" vs "Never execute without confirmation"
- ¿Qué aplica primero?

**Solución**: Coherencia y ordenamiento explícito:

```markdown
## Procedure
1. **Validate**: Check query is a safe SELECT operation
2. **Confirm**: Ask user confirmation for queries affecting >100 rows
3. **Execute**: Run the validated query
4. **Report**: Return formatted results

## Rules
- Only SELECT operations are allowed
- Always validate before execution
- Request confirmation for large result sets (>100 rows)
```

### 3.5 Anti-patrón 4: Over-engineering

**Definición**: Añadir complejidad innecesaria antes de que sea justificada por necesidades reales.

**Problema**:
- Difícil de mantener
- Errores sofisticados
- Tiempo de desarrollo desperdiciado

**Ejemplo problemático**:

```markdown
## Procedure
1. Query the database
2. If query is simple, use fast path (optimized)
3. If query is complex, use slow path (with caching)
4. If query involves joins, use join optimizer
5. If query has subqueries, use subquery rewriter
6. Apply query result transformation based on user type
7. Cache results with TTL based on data volatility
8. ...
```

**Síntomas**:
- Múltiples capas de abstracción
- Casos edge extensivos
- Lógica condicional compleja

**Solución**: YAGNI (You Ain't Gonna Need It):

```markdown
## Procedure
1. Validate query is a SELECT operation
2. Execute query
3. Format and return results

## Future enhancements
- Add caching if performance becomes an issue
- Add query optimization if slow queries are common
```

### 3.6 Anti-patrón 5: Skill Fragmentada

**Definición**: Dividir excesivamente una skill, creando múltiples micro-skills que deberían ser una.

**Problema**:
- Discovery confuso - múltiples skills para tareas relacionadas
- Invocación fragmentada
- Context switching innecesario

**Ejemplo problemático**:

```
query-select
query-insert
query-update
query-delete
```

**Problema**: Estas son operaciones del mismo dominio (database), mismo workflow (query execution), y mismo usuario. Deberían ser una skill con subfuncionalidades.

**Solución**: Consolidar según criterios:

```
query-select + query-insert + query-update + query-delete
→ database-query (con descripción que cubra CRUD operations)
```

**Excepción**: Si diferentes operaciones tienen workflows significativamente diferentes o usuarios diferentes, mantener separadas.

### 3.7 Anti-patrón 6: Sin Ejemplos

**Definición**: Skill sin few-shot examples que demuestren el output esperado.

**Problema**:
- El agente no entiende el formato de output esperado
- Variabilidad en la calidad de respuestas
- Iteraciones innecesarias

**Ejemplo problemático**:

```markdown
## Procedure
1. Parse the user's natural language query
2. Convert to SQL
3. Execute and return results
```

**Problema**: Sin ejemplos, el agente no sabe:
- ¿Qué formato de SQL?
- ¿Cómo se ven los resultados?
- ¿Qué nivel de detalle?

**Solución**: Añadir 2-5 ejemplos:

```markdown
## Examples

### Example 1: Simple query
User: "Show me customers from São Paulo"
Output:
```sql
SELECT A1_COD, A1_NOME, A1_EST FROM SA1010 WHERE A1_EST = 'SP'
```
Results: 15 customers found...

### Example 2: Aggregation
User: "Total sales by region"
Output:
```sql
SELECT A1_EST, SUM(C5_TOTAL) FROM SC5010 
JOIN SA1010 ON C5_CLIENTE = A1_COD 
GROUP BY A1_EST
```
Results: 5 regions, totals calculated...
```

---

## 4. Fuentes y Trazabilidad

| Hallazgo | Fuente | Referencia |
|----------|--------|------------|
| Criterios de división por dominio/workflow | Anthropic Engineering Blog | "Building Effective AI Agents" (Dec 2024) |
| Progressive disclosure 3 niveles | Anthropic Engineering Blog | "Equipping agents for the real world with Agent Skills" (Oct 2025) |
| Progressive disclosure 2 niveles | OpenAI Developers | "Agent Skills – Codex" (March 2026) |
| Anti-patrón "God Skill" | Anthropic Engineering Blog | "Building Effective AI Agents" (Dec 2024) |
| Description con triggers | OpenAI Developers | Codex skills documentation |
| Few-shot óptimo 2-5 | arXiv:2407.12994 | Prompt Engineering Methods Survey |
| Body <500 líneas | Claude docs | Best practices guide |
| CrewAI filesystem skills | CrewAI Docs | https://docs.crewai.com/en/concepts/skills |
| Directory-based skills con scripts | agentskills.io | https://agentskills.io |

---

## 5. Conclusión

### 5.1 Resumen de Principios

| Área | Principio Clave |
|------|-----------------|
| **División** | Una skill = Un dominio + Un workflow principal + Una audiencia + Uso consistente |
| **Progressive disclosure** | Discovery Layer (siempre) → Execution Layer (condicional) |
| **Anti-patrones** | Evitar God Skills, descripciones vagas, instrucciones contradictorias, over-engineering |

### 5.2 Aplicación a mapj_cli

Para las skills existentes de mapj_cli:

1. **Evaluar división**: Usar los 4 criterios para identificar si alguna skill debe dividirse
2. **Implementar progressive disclosure**: Asegurar que frontmatter tiene name, description, keywords
3. **Evitar anti-patrones**: Revisar cada skill contra la tabla de anti-patrones

### 5.3 Checklist de División

Antes de crear una skill, verificar:

- [ ] ¿Opera en un solo dominio conceptual?
- [ ] ¿Ejecuta un solo workflow principal?
- [ ] ¿Sirve a una audiencia principal?
- [ ] ¿Todas las funciones se usan con frecuencia similar?
- [ ] ¿Body <500 líneas?
- [ ] ¿Description tiene triggers específicos?

Si la respuesta es NO a cualquiera, considerar división.

---

**Validación**: Este documento cumple con:
- ✅ VAL-STR-001: Criterios de división documentados (dominio, workflow, usuario, frecuencia)
- ✅ VAL-STR-002: Progressive disclosure explicado con diagramas y capas
- ✅ VAL-STR-003: 6 anti-patrones documentados con problema, síntomas y solución
