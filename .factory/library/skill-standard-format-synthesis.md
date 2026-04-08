# Síntesis: Formato Estándar Emergente de Skills

**Assertion cumplida**: VAL-STD-001

**Fecha**: 2026-04-08

---

## Resumen

El formato **SKILL.md con YAML frontmatter** ha emergido como el estándar de facto para la definición de skills en agentes de IA. Este formato es adoptado por Anthropic, OpenAI, Cursor y VS Code Copilot, representando un consenso de facto entre los principales proveedores de agentes de IA.

---

## 1. Estructura del Formato Estándar

### 1.1 Esquema Base

```markdown
---
name: skill-name          # Identificador único
description: When and why to use  # Propósito y triggers
---

# Instructions

[Procedural guidance, ejemplos, y referencias]
```

### 1.2 Anatomía del YAML Frontmatter

El frontmatter es un bloque YAML delimitado por `---` al inicio del archivo. Contiene metadatos esenciales para el **discovery** y **routing** de skills.

---

## 2. Campos: Requeridos vs Opcionales

### 2.1 Tabla de Campos

| Campo | Requerido | Límite | Formato | Descripción |
|-------|-----------|--------|---------|-------------|
| `name` | ✅ REQUERIDO | 3-64 chars | lowercase + hyphens | Identificador único de la skill. Debe ser URL-safe y human-readable. |
| `description` | ✅ REQUERIDO | max 1024 chars | texto plano | Cuándo y por qué usar la skill. Crítico para discovery automático. |
| `instructions` / body | ✅ REQUERIDO | <500 líneas, <8,000 chars | Markdown | Guía procedural de ejecución. Incluye ejemplos y referencias. |
| `keywords` | ⬜ OPCIONAL | 5-10 items | array de strings | Términos para matching semántico. Mejora discovery. |
| `version` | ⬜ OPCIONAL | semver | string (ej. "1.0.0") | Versión de la skill para tracking de cambios. |
| `author` | ⬜ OPCIONAL | texto | string | Autor o equipo responsable. |
| `triggers` | ⬜ OPCIONAL | array | lista de patrones | Condiciones explícitas de activación. |
| `tools` | ⬜ OPCIONAL | array | lista de herramientas | Herramientas que la skill puede usar. |

### 2.2 Justificación de Límites

| Límite | Fuente | Razón |
|--------|--------|-------|
| `name`: 3-64 chars | Anthropic, OpenAI | Minimum legibility + maximum URL compatibility |
| `description`: max 1024 chars | Claude docs, Copilot specs | Visible en discovery UI sin truncamiento agresivo |
| `instructions`: <500 líneas | Claude docs best practices | Evita context pollution, mantiene foco |
| `instructions`: <8,000 chars | Microsoft Copilot docs | Límite hard del sistema de prompts |

### 2.3 Ejemplo de Frontmatter Completo

```yaml
---
name: mapj-protheus-query
description: |
  Execute SQL queries against Protheus ERP database when the user needs to retrieve
  business data, generate reports, or analyze records. Use for SELECT operations
  on Protheus tables (SA1 customers, SB1 products, SC5 orders, etc.).
  Triggers: "protheus", "query", "erp data", "totvs", "advpl database".
keywords:
  - protheus
  - query
  - erp
  - totvs
  - database
version: "3.2.0"
author: "mapj_cli team"
tools:
  - execute_query
  - validate_sql
---
```

---

## 3. Progressive Disclosure Pattern

### 3.1 Concepto

**Progressive disclosure** es un patrón arquitectónico que separa la información de la skill en dos capas con propósitos distintos:

| Capa | Contenido | Propósito | Visibilidad |
|------|-----------|-----------|-------------|
| **Discovery Layer** | name, description, keywords | Matching y routing de skills | Siempre cargada en el contexto del agente |
| **Execution Layer** | instructions, examples, references | Ejecución detallada de la skill | Carga condicional (cuando la skill es seleccionada) |

### 3.2 Diagrama de Capas

```
┌─────────────────────────────────────────────────────────┐
│                    AGENT CONTEXT                         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │           DISCOVERY LAYER (siempre)              │   │
│  │  • Skill 1: name, description, keywords         │   │
│  │  • Skill 2: name, description, keywords         │   │
│  │  • Skill 3: name, description, keywords         │   │
│  │  • ...                                           │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │         EXECUTION LAYER (condicional)            │   │
│  │  • Skill N: instructions, examples, refs        │   │  ← Cargada solo cuando
│  │                                                  │   │    la skill es invocada
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 3.3 Beneficios

| Beneficio | Descripción |
|-----------|-------------|
| **Eficiencia de contexto** | El agente no carga instrucciones detalladas de skills que no usará |
| **Discovery rápido** | Descripción + keywords permiten matching semántico eficiente |
| **Escalabilidad** | 100+ skills pueden existir sin agotar el contexto |
| **Reducción de noise** | Instrucciones irrelevantes no contaminan el contexto |

### 3.4 Implementación

El patrón se implementa típicamente:

1. **Fase de Discovery**: El sistema carga solo frontmatter de todas las skills
2. **Matching**: El LLM evalúa cuál skill(es) son relevantes para la query del usuario
3. **Fase de Execution**: El sistema carga el body completo de la skill seleccionada
4. **Ejecución**: El LLM sigue las instrucciones detalladas

### 3.5 Ejemplo de Implementación

```python
# Pseudocódigo de implementación
def load_discovery_layer(skills_dir):
    """Carga solo frontmatter de todas las skills"""
    skills_metadata = []
    for skill_file in skills_dir.glob("*/SKILL.md"):
        frontmatter = parse_yaml_frontmatter(skill_file)
        skills_metadata.append({
            'name': frontmatter['name'],
            'description': frontmatter['description'],
            'keywords': frontmatter.get('keywords', [])
        })
    return skills_metadata

def load_execution_layer(skill_name, skills_dir):
    """Carga el body completo de una skill específica"""
    skill_file = skills_dir / skill_name / "SKILL.md"
    return parse_full_markdown(skill_file)
```

---

## 4. Adopción por Proveedores

### 4.1 Tabla Comparativa

| Proveedor | Formato | Estándar Abierto | Progressive Disclosure | Notas |
|-----------|---------|------------------|------------------------|-------|
| **Anthropic** | SKILL.md + YAML | ✓ agentskills.io | ✅ Implementado | 100+ skills internas. Referencia principal del estándar. |
| **OpenAI** | SKILL.md + YAML | ✓ compatible | ✅ Implementado | Soporta hosted y local shell modes. |
| **Cursor** | SKILL.md + YAML | ✓ compatible | ✅ Implementado | Integración con IDE. |
| **VS Code Copilot** | SKILL.md + YAML | ✓ compatible | ✅ Implementado | Extension de VS Code. |
| **Microsoft** | JSON manifest | ✗ proprietario | ✅ Implementado | Instructions ≤8,000 chars. Knowledge sources separados. |
| **Google** | JSON/YAML config | ✗ proprietario | ⬜ Parcial | Agent Designer visual. GitHub data stores. |

### 4.2 Tendencia Observada

El formato **SKILL.md + YAML frontmatter** está convergiendo como el estándar de facto:

- **4 de 6 proveedores principales** lo adoptan
- **agentskills.io** provee especificación abierta
- **Compatibilidad cross-platform** favorece adopción

---

## 5. Fuentes y Trazabilidad

| Hallazgo | Fuente | Referencia |
|----------|--------|------------|
| name: 3-64 chars, lowercase + hyphens | Anthropic Engineering Blog | "Building Effective AI Agents" (Dec 2024) |
| description: max 1024 chars | Claude docs | Skill definition specs |
| instructions: <500 líneas | Claude docs | Best practices guide |
| instructions: <8,000 chars | Microsoft Learn | "Best practices for declarative agents" |
| Progressive disclosure | Anthropic | "Equipping agents for the real world with Agent Skills" (Oct 2025) |
| Few-shot óptimo: 2-5 | arXiv | arXiv:2407.12994 - Prompt Engineering Methods Survey |
| agentskills.io spec | Agent Skills Standard | https://agentskills.io |

---

## 6. Conclusión

El formato **SKILL.md con YAML frontmatter** representa el estándar emergente para definición de skills en agentes de IA, con:

1. **Estructura clara**: Frontmatter para discovery, body para execution
2. **Límites bien definidos**: name (64 chars), description (1024 chars), instructions (8,000 chars)
3. **Progressive disclosure**: Separa discovery layer de execution layer para eficiencia
4. **Adopción amplia**: Anthropic, OpenAI, Cursor, VS Code Copilot

La adopción de este formato en mapj_cli es alineada con las mejores prácticas de la industria y maximiza la compatibilidad con el ecosistema de agentes de IA.

---

**Validación**: Este documento cumple con VAL-STD-001:
- ✅ Documenta SKILL.md + YAML frontmatter
- ✅ Tabla de campos requeridos/opcionales con límites
- ✅ Explicación de progressive disclosure
