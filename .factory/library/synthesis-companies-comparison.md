# Síntesis: Comparativa de Enfoques de Diseño de Skills por Empresas Líderes

**Assertion cumplida**: VAL-STD-002

**Fecha**: 2026-04-08

---

## Resumen

Este documento sintetiza los enfoques de diseño de skills/agentes de las principales empresas de IA: Anthropic, OpenAI, Microsoft, Google y Amazon. Se identifican dos tendencias principales: (1) el formato SKILL.md + YAML frontmatter como estándar abierto adoptado por Anthropic y OpenAI, y (2) formatos propietarios JSON/YAML utilizados por Microsoft, Google y Amazon.

---

## 1. Tabla Comparativa

| Empresa | Formato | Estándar Abierto | Características Distintivas |
|---------|---------|-------------------|----------------------------|
| **Anthropic** | SKILL.md + YAML frontmatter | ✓ [agentskills.io](https://agentskills.io/) | Progressive disclosure en 3 niveles, 100+ skills internas, directory-based skills con scripts y recursos |
| **OpenAI** | SKILL.md + YAML frontmatter | ✓ Compatible con [agentskills.io](https://agentskills.io/) | Progressive disclosure, invocación implícita/explícita, hosted y local shell modes, plugins como unidad de distribución |
| **Microsoft** | JSON manifest (app manifest) | ✗ Propietario | Declarative agents con instructions ≤8,000 chars, knowledge sources separados, integración Microsoft 365 |
| **Google** | JSON/YAML config + Agent Designer | ✗ Propietario (ADK open source) | Visual low-code designer, Agent Development Kit (ADK), Agent Garden con samples, GitHub data stores |
| **Amazon** | JSON interaction model (ASK) | ✗ Propietario | Voice-first design, Lambda functions, Alexa Skills Kit, utterances y slots |

---

## 2. Análisis Detallado por Empresa

### 2.1 Anthropic

**Formato**: SKILL.md con YAML frontmatter

**Estándar Abierto**: ✓ Anthropic publicó el estándar abierto [agentskills.io](https://agentskills.io/) en diciembre 2025.

**Características Distintivas**:

1. **Progressive Disclosure en 3 niveles**:
   - **Nivel 1 (Discovery)**: `name` y `description` en frontmatter - siempre cargados para routing
   - **Nivel 2 (Execution)**: Body completo de SKILL.md - cargado cuando la skill es relevante
   - **Nivel 3 (Deep)**: Archivos adicionales referenciados - cargados condicionalmente

2. **Directory-based Skills**: Cada skill es un directorio que puede contener:
   - `SKILL.md` - Instrucciones principales
   - `scripts/` - Código ejecutable
   - `references/` - Documentación
   - `assets/` - Templates y recursos

3. **100+ Skills Internas**: Claude Code y la plataforma Claude utilizan más de 100 skills internas para diferentes capacidades.

**Cita de Documentación Oficial**:

> "A skill is a directory containing a SKILL.md file that contains organized folders of instructions, scripts, and resources that give agents additional capabilities."
> 
> "This metadata is the first level of progressive disclosure: it provides just enough information for Claude to know when each skill should be used without loading all of it into context."
> 
> — Anthropic Engineering Blog, "Equipping agents for the real world with Agent Skills" (October 2025)

**Referencia**: https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills

---

### 2.2 OpenAI

**Formato**: SKILL.md con YAML frontmatter

**Estándar Abierto**: ✓ Compatible con el estándar abierto [agentskills.io](https://agentskills.io/).

**Características Distintivas**:

1. **Progressive Disclosure**: Mismo patrón que Anthropic - metadata siempre visible, body cargado condicionalmente.

2. **Invocación Dual**:
   - **Explícita**: `$skill-name` o `/skills` command
   - **Implícita**: Codex selecciona la skill cuando la descripción matchea con el prompt del usuario

3. **Plugins como Unidad de Distribución**: Skills pueden empaquetarse como plugins para distribución, incluyendo MCP server config y app integrations.

4. **Múltiples Scopes de Skills**:
   - `REPO`: `.agents/skills` en el repositorio
   - `USER`: `$HOME/.agents/skills`
   - `ADMIN`: `/etc/codex/skills`
   - `SYSTEM`: Bundled con Codex

**Cita de Documentación Oficial**:

> "Use agent skills to extend Codex with task-specific capabilities. A skill packages instructions, resources, and optional scripts so Codex can follow a workflow reliably. Skills build on the open agent skills standard."
> 
> "Skills use progressive disclosure to manage context efficiently: Codex starts with each skill's metadata (name, description, file path...). Codex loads the full SKILL.md instructions only when it decides to use a skill."
> 
> — OpenAI Developers, "Agent Skills – Codex" (March 2026)

**Referencia**: https://developers.openai.com/codex/skills

---

### 2.3 Microsoft

**Formato**: JSON manifest (app manifest) para Microsoft 365 Copilot

**Estándar Abierto**: ✗ Propietario - formato específico del ecosistema Microsoft.

**Características Distintivas**:

1. **Declarative Agents**: Personalización de Copilot via:
   - `instructions`: Guía de comportamiento (≤8,000 chars)
   - `actions`: Conexiones a APIs externas
   - `knowledge`: Fuentes de conocimiento (SharePoint, files, URLs)

2. **Límite Hard**: Instructions limitadas a 8,000 caracteres.

3. **Integración Microsoft 365**: Diseñados para operar dentro del ecosistema Office/Teams.

4. **No Progressive Disclosure**: Todo el manifest se carga de una vez.

**Cita de Documentación Oficial**:

> "Declarative agents customize Microsoft 365 Copilot via instructions, actions, and knowledge."
> 
> "Write effective instructions for declarative agents - learn how to give your agent the right context and guidance."
> 
> — Microsoft Learn, "Best practices for building declarative agents in Microsoft 365 Copilot" (October 2025)

**Referencia**: https://learn.microsoft.com/en-us/microsoft-365/copilot/extensibility/declarative-agents-overview

---

### 2.4 Google

**Formato**: JSON/YAML config con Agent Designer visual y Agent Development Kit (ADK)

**Estándar Abierto**: ✗ Propietario, pero ADK es open source.

**Características Distintivas**:

1. **Agent Designer**: Low-code visual designer para diseñar y testear agentes sin código.

2. **Agent Development Kit (ADK)**: Framework open source para construir multi-agent systems con control preciso del comportamiento.

3. **Agent Garden**: Biblioteca de agentes y tools sample prebuilt para acelerar desarrollo.

4. **Vertex AI Agent Engine**: Runtime gestionado para deploy, scale y govern agents en producción.

5. **Ecosystem Tools**: Soporte para tools de LangChain, CrewAI, MCP tools.

**Cita de Documentación Oficial**:

> "Vertex AI Agent Builder is a suite of products that help developers build, scale, and govern AI agents in production."
> 
> "Agent Development Kit (ADK) is an open-source framework that simplifies the process of building sophisticated multi-agent systems while maintaining precise control over agent behavior."
> 
> — Google Cloud Documentation, "Vertex AI Agent Builder overview" (April 2026)

**Referencia**: https://docs.cloud.google.com/agent-builder/overview

---

### 2.5 Amazon

**Formato**: JSON interaction model (Alexa Skills Kit)

**Estándar Abierto**: ✗ Propietario - diseñado específicamente para Alexa.

**Características Distintivas**:

1. **Voice-First Design**: Skills diseñadas para interacción por voz.

2. **Interaction Model**: Define:
   - `intents`: Acciones que la skill puede realizar
   - `utterances`: Frases que trigger intents
   - `slots`: Parámetros variables

3. **Lambda Functions**: Backend serverless para lógica de la skill.

4. **Pre-built Models**: Modelos predefinidos para intents comunes (Amazon intents).

**Cita de Documentación Oficial**:

> "The Alexa Skills Kit (ASK) is a software development framework that enables you to create content, called skills. Skills are like apps for Alexa."
> 
> "With an interactive voice interface, built-in intents and slots, and a hosted backend service..."
> 
> — Amazon Developer Documentation, "What is the Alexa Skills Kit?" (October 2025)

**Referencia**: https://developer.amazon.com/en-US/docs/alexa/ask-overviews/what-is-the-alexa-skills-kit

---

## 3. Análisis de Convergencia

### 3.1 Tendencia hacia SKILL.md

El formato **SKILL.md + YAML frontmatter** está emergiendo como estándar de facto:

| Factor | Descripción |
|--------|-------------|
| **Adopción por líderes** | Anthropic y OpenAI, los dos principales proveedores de LLMs para agentes, lo adoptan |
| **Estándar abierto** | agentskills.io provee especificación pública y cross-platform |
| **Compatibilidad** | Mismo formato funciona en Claude Code, Codex CLI, IDEs, web |

### 3.2 Divergencia en Implementación

| Aspecto | Anthropic | OpenAI | Microsoft | Google | Amazon |
|---------|-----------|--------|-----------|--------|--------|
| **Progressive Disclosure** | ✓ 3 niveles | ✓ 2 niveles | ✗ | ⬜ Parcial | ✗ |
| **Open Source** | ✓ spec pública | ✓ compatible | ✗ | ✓ ADK | ✗ |
| **Visual Designer** | ✗ | ✗ | ✗ | ✓ | ✗ |
| **Voice-First** | ✗ | ✗ | ✗ | ✗ | ✓ |
| **Distribución** | Files/dirs | Plugins | App package | Agent Engine | Alexa Store |

### 3.3 Implicaciones para Desarrolladores

1. **Portabilidad**: Skills en formato SKILL.md pueden funcionar en Claude Code y Codex sin modificación
2. **Vendor Lock-in**: Formatos propietarios (Microsoft, Google, Amazon) requieren reescritura para migrar
3. **Progressive Disclosure**: Patrón crítico para escalar a 100+ skills sin agotar contexto
4. **Ecosistema**: OpenAI y Anthropic comparten estándar, facilitando ecosistema unificado

---

## 4. Fuentes y Trazabilidad

| Hallazgo | Fuente | URL |
|----------|--------|-----|
| Anthropic: SKILL.md + YAML, progressive disclosure | Anthropic Engineering Blog | https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills |
| Anthropic: agentskills.io open standard | Anthropic News | https://agentskills.io/ |
| OpenAI: Skills con progressive disclosure | OpenAI Developers | https://developers.openai.com/codex/skills |
| OpenAI: Plugins como unidad de distribución | OpenAI Developers | https://developers.openai.com/codex/plugins/build |
| Microsoft: Declarative agents, JSON manifest | Microsoft Learn | https://learn.microsoft.com/en-us/microsoft-365/copilot/extensibility/declarative-agents-overview |
| Microsoft: Instructions ≤8,000 chars | Microsoft Learn | https://learn.microsoft.com/en-us/microsoft-365/copilot/extensibility/declarative-agent-instructions-best-practices |
| Google: Agent Builder, ADK, Agent Designer | Google Cloud Docs | https://docs.cloud.google.com/agent-builder/overview |
| Google: ADK open source | Google GitHub | https://google.github.io/adk-docs/ |
| Amazon: Alexa Skills Kit, interaction model | Amazon Developer Docs | https://developer.amazon.com/en-US/docs/alexa/ask-overviews/what-is-the-alexa-skills-kit |

---

## 5. Conclusión

La comparativa revela una **bifurcación del mercado**:

1. **Estandarización (Anthropic + OpenAI)**: Adoptan SKILL.md + YAML como formato común, con progressive disclosure para escalabilidad. Publican especificación abierta (agentskills.io).

2. **Propietarios (Microsoft + Google + Amazon)**: Mantienen formatos propios con features diferenciadores (visual designer, integración enterprise, voice UI).

**Recomendación para mapj_cli**: Adoptar el formato **SKILL.md + YAML frontmatter** porque:
- Compatible con ecosistema Anthropic/OpenAI
- Progressive disclosure permite escalar
- Estándar abierto facilita portabilidad
- Simplicidad del formato favorece adopción

---

**Validación**: Este documento cumple con VAL-STD-002:
- ✅ Compara 5 empresas líderes (Anthropic, OpenAI, Microsoft, Google, Amazon)
- ✅ Tabla comparativa con enfoques, formatos y características distintivas
- ✅ Citas de documentación oficial para cada empresa
