# Skill Design Research

**What belongs here:** Hallazgos de investigación sobre diseño de skills para AI agents.

---

## Estándar Emergente: SKILL.md + YAML Frontmatter

### Estructura
```markdown
---
name: skill-name          # max 64 chars, lowercase + hyphens
description: When and why to use  # max 1024 chars
---

# Instructions
[Procedural guidance, <500 lines]
```

### Campos
| Campo | Requerido | Límite |
|-------|-----------|--------|
| `name` | ✅ | 3-64 chars |
| `description` | ✅ | max 1024 chars |
| `instructions/body` | ✅ | <500 lines, <8,000 chars |

## Empresas Líderes

| Empresa | Formato | Estándar Abierto | Características |
|---------|---------|------------------|-----------------|
| Anthropic | SKILL.md + YAML | ✓ agentskills.io | Progressive disclosure, 100+ skills internas |
| OpenAI | SKILL.md + YAML | ✓ compatible | Hosted/local shell modes |
| Microsoft | JSON manifest | ✗ | Instructions ≤8,000 chars, knowledge sources |
| Google | JSON/YAML | ✗ | Agent Designer visual, GitHub data stores |
| Amazon | JSON interaction model | ✗ | Voice UI, Lambda functions |

## Frameworks

| Framework | Definición | Campos Requeridos |
|-----------|------------|-------------------|
| LangChain | @tool decorator | name, description (auto from docstring) |
| CrewAI | Directory + SKILL.md | name, description |
| AutoGen | Función Python | name, description, parameters (auto) |
| Haystack | Tool dataclass | name, description, parameters, function |

## Contenido Óptimo

### Few-shot Examples
- **Óptimo**: 2-5 ejemplos
- **Evidencia**: arXiv papers muestran meseta después de 2-3 ejemplos
- Para modelos de razonamiento (o1, R1): usar 0-1 ejemplos

### Progressive Disclosure
- **Discovery Layer**: name, description, keywords (siempre visible)
- **Execution Layer**: instrucciones detalladas, ejemplos (carga condicional)

## Anti-patrones

| Anti-patrón | Problema | Solución |
|-------------|----------|----------|
| "God Skill" | Instrucciones contradictorias | Dividir en skills específicas |
| Descripciones vagas | Discovery falla | Keywords específicos, triggers claros |
| Skills largas | Context pollution | <500 líneas, progressive disclosure |
| Over-engineering | Complejidad innecesaria | Empezar simple, escalar cuando necesario |

## Fuentes

- Anthropic Engineering Blog: "Building Effective AI Agents" (Dec 2024)
- Anthropic: "Equipping agents for the real world with Agent Skills" (Oct 2025)
- Microsoft Learn: "Best practices for building declarative agents"
- arXiv:2407.12994 - Prompt Engineering Methods Survey
- arXiv:2507.21504 - LLM Agent Evaluation Survey
- Agent Skills Standard: agentskills.io
