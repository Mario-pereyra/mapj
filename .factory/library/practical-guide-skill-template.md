# Guía Práctica: Template de SKILL.md

**Assertion cumplida**: VAL-APP-001

**Fecha**: 2026-04-08

---

## Resumen

Este documento proporciona un template de SKILL.md listo para usar, basado en el estándar emergente documentado en las síntesis previas. El template sigue el formato **SKILL.md + YAML frontmatter** adoptado por Anthropic, OpenAI, Cursor y VS Code Copilot, implementando el patrón de progressive disclosure para optimizar el uso del contexto del agente.

---

## 1. Quick Reference Card

### Campos Requeridos vs Opcionales

| Campo | Requerido | Límite | Formato |
|-------|-----------|--------|---------|
| `name` | ✅ REQUERIDO | 3-64 chars | lowercase + hyphens |
| `description` | ✅ REQUERIDO | max 1024 chars | texto plano con triggers |
| **body** (instructions) | ✅ REQUERIDO | <500 líneas, <8,000 chars | Markdown |
| `keywords` | ⬜ OPCIONAL | 5-10 items | array de strings |
| `version` | ⬜ OPCIONAL | semver | string (ej. "1.0.0") |
| `author` | ⬜ OPCIONAL | texto | string |

### Secciones del Body

| Sección | Prioridad | Longitud Recomendada |
|---------|-----------|---------------------|
| **Role/Identidad** | ✅ CRÍTICO | 50-100 chars |
| **Procedure** | ✅ CRÍTICO | 1,000-4,000 chars |
| **Examples** | ✅ RECOMENDADO | 2-5 ejemplos (200-500 chars c/u) |
| **Success Criteria** | ✅ RECOMENDADO | 100-300 chars |
| **Limitations** | ⬜ OPCIONAL | 100-300 chars |

---

## 2. Template Completo (Copy-Paste Ready)

```markdown
---
# ============================================================================
# DISCOVERY LAYER (Siempre visible por el agente)
# ============================================================================

name: your-skill-name              # REQUERIDO: 3-64 chars, lowercase + hyphens
description: |                     # REQUERIDO: max 1024 chars
  [QUÉ hace la skill] when [CONTEXTO/CONDICIÓN].
  Use for [CASOS DE USO ESPECÍFICOS].
  Triggers: "[palabra1]", "[palabra2]", "[palabra3]".
  # Ejemplo: "Execute SQL queries against Protheus ERP when the user needs 
  # business data. Use for SELECT on tables SA1, SB1, SC5. 
  # Triggers: 'protheus', 'query', 'erp data'."

keywords:                           # OPCIONAL: 5-10 items para matching semántico
  - keyword1
  - keyword2
  - keyword3
  - keyword4
  - keyword5

version: "1.0.0"                    # OPCIONAL: semver para tracking de cambios
author: "Your Name or Team"         # OPCIONAL: responsable de la skill

# ============================================================================
# FIN DEL DISCOVERY LAYER
# ============================================================================
---

# Instructions

<!--
============================================================================
EXECUTION LAYER (Cargado condicionalmente cuando la skill es invocada)
============================================================================
-->

## Role

<!-- Define el rol/expertise del agente para esta skill (50-100 chars) -->
Act as a [ROL] with expertise in [DOMINIO/TECNOLOGÍA].

<!-- Ejemplo: "Act as a database analyst with expertise in Protheus ERP." -->

---

## Context

<!-- Opcional: Contexto adicional que el agente necesita saber -->
[INFORMACIÓN DE CONTEXTO RELEVANTE]

<!-- Ejemplo: "This skill operates on the Protheus ERP database (Totvs). 
Tables use prefix conventions: SA1 (customers), SB1 (products), SC5 (orders)." -->

---

## Procedure

<!-- Pasos procedimentales específicos. Cada paso debe ser accionable. -->

1. **[PASO 1]**: [ACCIÓN ESPECÍFICA]
   - [SUB-PASO O DETALLE]
   - [Criterio de validación si aplica]

2. **[PASO 2]**: [ACCIÓN ESPECÍFICA]
   - [SUB-PASO O DETALLE]

3. **[PASO 3]**: [ACCIÓN ESPECÍFICA]
   - [SUB-PASO O DETALLE]

4. **[PASO 4]**: [ACCIÓN ESPECÍFICA]
   - [SUB-PASO O DETALLE]

<!-- Ejemplo:
1. **Validate request**: Ensure the query is a SELECT operation (no INSERT/UPDATE/DELETE)
2. **Parse tables**: Identify Protheus tables involved (SA1, SB1, SC5, etc.)
3. **Generate query**: Write the SQL following Protheus conventions
4. **Execute**: Run the query and return results in a readable format
-->

---

## Examples

<!-- 2-5 ejemplos representativos. Cada ejemplo debe mostrar input y output esperado -->

### Example 1: [TÍTULO DESCRIPTIVO]

**User request:**
```
[Lo que dice el usuario]
```

**Expected output:**
```
[Output esperado de la skill]
```

**Explanation:** [Breve explicación de por qué este output es correcto]

---

### Example 2: [TÍTULO DESCRIPTIVO]

**User request:**
```
[Lo que dice el usuario]
```

**Expected output:**
```
[Output esperado de la skill]
```

**Explanation:** [Breve explicación de por qué este output es correcto]

---

### Example 3: [TÍTULO DESCRIPTIVO - CASO EDGE]

**User request:**
```
[Lo que dice el usuario - caso edge]
```

**Expected output:**
```
[Output esperado - cómo manejar el edge case]
```

**Explanation:** [Cómo se maneja este caso especial]

---

## Success Criteria

<!-- Define cuándo la ejecución es exitosa. 3-5 criterios. -->

The skill execution is successful when:
- [CRITERIO 1]
- [CRITERIO 2]
- [CRITERIO 3]

<!-- Ejemplo:
- Query returns expected data structure
- No SQL injection vulnerabilities introduced
- Results formatted for user readability
- Error handled gracefully if query fails
-->

---

## Limitations

<!-- Documenta límites conocidos. Gestiona expectativas. -->

- [LIMITACIÓN 1]
- [LIMITACIÓN 2]
- [LIMITACIÓN 3]

<!-- Ejemplo:
- Only SELECT operations supported (no INSERT/UPDATE/DELETE)
- Maximum 1000 rows returned per query
- No cross-database queries
-->

---

## Error Handling

<!-- Opcional: Guía para manejar errores comunes -->

If [ERROR CONDITION]:
- [ACCIÓN A TOMAR]

<!-- Ejemplo:
If query returns 0 results:
- Suggest alternative queries or tables
- Ask user if they want to expand search criteria
-->

---

## References

<!-- Opcional: Links a documentación externa o recursos -->

- [Título del recurso](URL) - [Breve descripción de qué contiene]
- [Título del recurso](URL) - [Breve descripción de qué contiene]

<!-- Ejemplo:
- [Protheus Table Reference](https://tdn.totvs.com/display/PROT/Tables) - Official TDN documentation
- [SQL Best Practices](https://example.com/sql-guide) - Query optimization guide
-->

<!--
============================================================================
FIN DEL EXECUTION LAYER
============================================================================
-->
```

---

## 3. Guía de Uso por Sección

### 3.1 Discovery Layer (YAML Frontmatter)

#### `name` (REQUERIDO)

**Propósito**: Identificador único para discovery y routing.

**Reglas**:
- Solo lowercase y hyphens (`-`)
- Sin espacios, underscores, o caracteres especiales
- 3-64 caracteres
- Debe ser descriptivo pero conciso

| ✅ Correcto | ❌ Incorrecto |
|-------------|---------------|
| `protheus-query` | `ProtheusQuery` |
| `code-review` | `code_review` |
| `data-export` | `export data` |
| `api-rest-client` | `api` (muy genérico) |

#### `description` (REQUERIDO)

**Propósito**: Explica cuándo y por qué usar la skill. Crítico para discovery automático.

**Estructura recomendada**:
```
[QUÉ hace] when [CONDICIÓN/CONTEXTO].
Use for [CASOS DE USO].
Triggers: "[palabra1]", "[palabra2]".
```

**Longitud**: max 1024 chars (recomendado 150-300 chars)

| ✅ Buena description | ❌ Mala description |
|---------------------|---------------------|
| `Execute SQL queries against Protheus ERP when the user needs business data. Use for SELECT on tables SA1, SB1, SC5. Triggers: "protheus", "query", "erp data".` | `Query database` |
| `Review code for bugs, security issues, and style violations when user asks for PR review. Triggers: "review", "pr", "code quality".` | `Helper skill` |
| `Export query results to Excel when user needs to download data. Triggers: "export", "excel", "download".` | `Does things with data` |

#### `keywords` (OPCIONAL)

**Propósito**: Mejora el matching semántico para discovery.

**Recomendación**: 5-10 keywords que usuarios usarían naturalmente.

```yaml
keywords:
  - protheus      # Producto
  - query         # Acción
  - erp           # Categoría
  - totvs         # Vendor
  - database      # Dominio
```

### 3.2 Execution Layer (Body)

#### `Role` (CRÍTICO)

**Propósito**: Define el expertise y perspectiva del agente.

**Template**: `Act as a [ROL] with expertise in [DOMINIO].`

**Ejemplos**:
- `Act as a database analyst with expertise in Protheus ERP data structures.`
- `Act as a security engineer with expertise in vulnerability assessment.`
- `Act as a technical writer with expertise in API documentation.`

#### `Procedure` (CRÍTICO)

**Propósito**: Pasos específicos para ejecutar la skill.

**Buenas prácticas**:
- Cada paso es una acción concreta
- Número de pasos: 3-7 (no más de 10)
- Incluir validaciones y checks
- Formato: `**[Verbo]**: [Acción específica]`

**Ejemplo bien estructurado**:
```markdown
1. **Validate request**: Ensure the query is a SELECT operation
   - Reject INSERT, UPDATE, DELETE operations
   - Check for SQL injection patterns

2. **Parse tables**: Identify Protheus tables involved
   - Map user terms to table codes (customers → SA1, products → SB1)
   
3. **Generate query**: Write SQL following conventions
   - Use proper column prefixes (A1_COD, B1_DESC)
   - Include appropriate WHERE clauses

4. **Execute**: Run the query safely
   - Apply row limit if not specified
   - Format results for readability
```

#### `Examples` (RECOMENDADO - 2-5)

**Propósito**: Demostrar el output esperado al agente.

**Estructura de cada ejemplo**:
```markdown
### Example N: [Título]

**User request:**
```
[Input del usuario]
```

**Expected output:**
```
[Output de la skill]
```

**Explanation:** [Por qué es correcto]
```

**Cantidad óptima**: 2-5 ejemplos (meseta de rendimiento)

| Tipo de Skill | # Ejemplos |
|---------------|------------|
| Transformación simple | 2 |
| Código/generación | 3-5 |
| Análisis/razonamiento | 1-2 |
| Casos edge complejos | 4-5 |

#### `Success Criteria` (RECOMENDADO)

**Propósito**: Define cuándo la ejecución es correcta.

**Estructura**: Lista de 3-5 criterios verificables.

```markdown
The skill execution is successful when:
- Query returns expected data structure
- No SQL injection vulnerabilities introduced
- Results formatted for user readability
- Error handled gracefully if query fails
```

#### `Limitations` (OPCIONAL)

**Propósito**: Gestiona expectativas del usuario.

**Cuándo incluir**: Siempre que haya restricciones conocidas.

```markdown
## Limitations

- Only SELECT operations supported
- Maximum 1000 rows returned
- No cross-database queries
- Requires valid database connection
```

---

## 4. Checklist de Calidad

Antes de finalizar una skill, verificar:

### Discovery Layer
- [ ] `name` es 3-64 chars, lowercase + hyphens
- [ ] `description` incluye QUÉ hace, CUÁNDO usarla, y TRIGGERS
- [ ] `description` < 1024 chars
- [ ] `keywords` tiene 5-10 items (si se incluye)

### Execution Layer
- [ ] `Role` define expertise específico
- [ ] `Procedure` tiene 3-7 pasos concretos
- [ ] `Examples` incluye 2-5 ejemplos representativos
- [ ] `Success Criteria` tiene 3-5 criterios verificables
- [ ] Body < 500 líneas
- [ ] Instructions < 8,000 chars

### Anti-patrones
- [ ] NO es una "God Skill" (un dominio, un workflow)
- [ ] NO tiene descripciones vagas
- [ ] NO tiene instrucciones contradictorias
- [ ] NO tiene over-engineering innecesario

---

## 5. Ejemplo Completo: Skill de Query

```markdown
---
name: protheus-query
description: |
  Execute SQL queries against Protheus ERP database when the user needs 
  business data, reports, or record analysis. Use for SELECT operations 
  on Protheus tables (SA1 customers, SB1 products, SC5 orders).
  Triggers: "protheus", "query", "erp data", "totvs", "database".
keywords:
  - protheus
  - query
  - erp
  - totvs
  - database
  - sql
version: "1.0.0"
author: "mapj_cli team"
---

# Instructions

## Role

Act as a database analyst with expertise in Protheus ERP (Totvs) data structures.

---

## Context

This skill operates on the Protheus ERP database. Tables use prefix conventions:
- SA1: Customers
- SB1: Products  
- SC5: Sales orders
- SC6: Order items
- SE1: Accounts receivable

---

## Procedure

1. **Validate request**: Ensure the query is a SELECT operation
   - Reject INSERT, UPDATE, DELETE, DROP operations
   - Check for SQL injection patterns (`;`, `--`, `DROP`, etc.)

2. **Parse tables**: Identify Protheus tables from user request
   - Map natural language to table codes
   - Examples: "customers" → SA1, "products" → SB1, "orders" → SC5

3. **Generate query**: Write SQL following Protheus conventions
   - Use proper column prefixes (A1_COD, A1_NOME, B1_DESC)
   - Include company filter (D_E_L_E_T_ = '')
   - Apply appropriate WHERE clauses

4. **Execute**: Run the validated query
   - Apply row limit (default 100, max 1000)
   - Format results as readable table

---

## Examples

### Example 1: Simple customer lookup

**User request:**
```
Show me customers from São Paulo
```

**Expected output:**
```sql
SELECT A1_COD, A1_NOME, A1_EST, A1_CGC 
FROM SA1010 
WHERE A1_EST = 'SP' AND D_E_L_E_T_ = ''
LIMIT 100
```

**Explanation:** Query filters customers by state (SP = São Paulo), includes relevant columns, and applies Protheus soft-delete filter.

---

### Example 2: Order summary

**User request:**
```
Get total sales by region for this month
```

**Expected output:**
```sql
SELECT A1_EST AS region, 
       COUNT(DISTINCT C5_NUM) AS order_count,
       SUM(C5_TOTAL) AS total_sales
FROM SC5010 C5
JOIN SA1010 A1 ON C5_CLIENTE = A1_COD
WHERE C5_EMISSAO BETWEEN '20260401' AND '20260430'
  AND C5.D_E_L_E_T_ = ''
  AND A1.D_E_L_E_T_ = ''
GROUP BY A1_EST
ORDER BY total_sales DESC
```

**Explanation:** Joins orders with customers to aggregate by region, filters current month, uses proper date format (YYYYMMDD).

---

### Example 3: Error handling - Invalid request

**User request:**
```
Delete all customers from São Paulo
```

**Expected output:**
```
I cannot execute DELETE operations. This skill only supports SELECT queries 
for retrieving data. Would you like to view the customers instead?
```

**Explanation:** Skill rejects destructive operations and offers constructive alternative.

---

## Success Criteria

The skill execution is successful when:
- Query is a safe SELECT operation
- Results match user's data request
- No SQL injection vulnerabilities introduced
- Results formatted for readability
- Appropriate row limits applied

---

## Limitations

- Only SELECT operations supported
- Maximum 1000 rows returned per query
- No cross-database queries
- Requires active Protheus connection
- Date format must follow YYYYMMDD convention

---

## Error Handling

If query returns 0 results:
- Suggest alternative search terms
- Ask if user wants to broaden criteria

If user requests non-SELECT operation:
- Politely decline
- Explain limitation
- Offer alternative (view data instead)
```

---

## 6. Estructura de Directorio Completa

Para skills que requieren recursos adicionales, usar estructura de directorio:

```
my-skill/
├── SKILL.md              # Required - Frontmatter + Instructions
├── scripts/              # Optional - Executable scripts
│   ├── helper.py
│   └── validate.sh
├── references/           # Optional - Reference documents
│   ├── api-docs.md
│   └── schema.json
└── assets/               # Optional - Static files
    ├── templates.json
    └── examples.yaml
```

**Progressive disclosure de 3 niveles** (Anthropic):
- **Nivel 1**: Frontmatter de SKILL.md (siempre)
- **Nivel 2**: Body de SKILL.md (cuando skill es invocada)
- **Nivel 3**: scripts/, references/, assets/ (cuando se necesitan)

---

## 7. Fuentes y Trazabilidad

| Hallazgo | Fuente | Referencia |
|----------|--------|------------|
| SKILL.md + YAML frontmatter | Anthropic Engineering Blog | "Equipping agents for the real world" (Oct 2025) |
| agentskills.io spec | Agent Skills Standard | https://agentskills.io |
| name: 3-64 chars | Anthropic, OpenAI | Building Effective AI Agents |
| description: max 1024 chars | Claude docs | Skill definition specs |
| instructions: <8,000 chars | Microsoft Learn | Best practices for declarative agents |
| Body: <500 líneas | Claude docs | Best practices guide |
| Few-shot: 2-5 examples | arXiv:2407.12994 | Prompt Engineering Methods Survey |
| Progressive disclosure | Anthropic | "Equipping agents for the real world" |
| Anti-patrón "God Skill" | Anthropic | "Building Effective AI Agents" |
| Directory-based skills | CrewAI Docs | https://docs.crewai.com/en/concepts/skills |

---

## 8. Conclusión

Este template proporciona todo lo necesario para crear skills efectivas siguiendo el estándar emergente:

1. **Discovery Layer completo**: name, description, keywords
2. **Execution Layer estructurado**: Role, Procedure, Examples, Success Criteria, Limitations
3. **Ejemplos concretos**: 3 ejemplos de la skill de Protheus
4. **Checklist de calidad**: Verificación antes de finalizar
5. **Estructura de directorio**: Para skills con recursos adicionales

**Para usar el template**: Copiar la sección "Template Completo" y reemplazar los placeholders con contenido específico de tu skill.

---

**Validación**: Este documento cumple con VAL-APP-001:
- ✅ Template de SKILL.md completo con YAML frontmatter
- ✅ Secciones placeholder comentadas
- ✅ Ejemplos de contenido para cada sección
- ✅ Template listo para copiar y usar
