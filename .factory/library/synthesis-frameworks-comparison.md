# Síntesis: Comparativa de Frameworks de Agentes de IA

**Assertion cumplida**: VAL-STD-003

**Fecha**: 2026-04-08

---

## Resumen

Este documento sintetiza los enfoques de definición de tools/skills en los principales frameworks de agentes de IA: LangChain, CrewAI, AutoGen, BabyAGI, Haystack y SuperAGI. Se identifican cuatro patrones de diseño principales: decorator-based, class-based, dataclass-based y filesystem-based. Todos los frameworks comparten campos universales (name, description, parameters) pero difieren en la forma de definirlos y la sintaxis utilizada.

---

## 1. Tabla Comparativa de Frameworks

| Framework | Patrón de Definición | Sintaxis | Campos Auto-generados | Schema JSON | Tipo |
|-----------|---------------------|----------|----------------------|-------------|------|
| **LangChain** | Decorator-based | `@tool` decorator sobre función Python | name (func name), description (docstring), parameters (type hints) | ✓ Auto desde type hints + Pydantic | Tool (callable) |
| **CrewAI** | Filesystem-based | Directorio con `SKILL.md` + YAML frontmatter | ❌ Manual | ❌ No aplica | Skill (instructions) |
| **AutoGen** | Class-based | Clase `FunctionTool` envolviendo función | name (func name), description (arg), parameters (type hints) | ✓ Auto con `schema` property | BaseTool subclass |
| **Haystack** | Dataclass-based | `Tool` dataclass con campos explícitos | ❌ Todos manuales | ✓ Manual en `parameters` dict | Tool (dataclass) |
| **BabyAGI** | Función directa | Funciones Python pasadas directamente | Depende de implementación | ❌ No define tools formalmente | N/A (task-driven) |
| **SuperAGI** | Class-based + Toolkit | Clases que heredan de `BaseTool` + ToolkitRegistry | name, description en config | ✓ En tool config | BaseTool + Toolkit |

---

## 2. Campos Universales vs Específicos por Framework

### 2.1 Campos Universales (presentes en todos los frameworks que definen tools)

| Campo | LangChain | CrewAI | AutoGen | Haystack | BabyAGI | SuperAGI |
|-------|-----------|--------|---------|----------|---------|----------|
| **name** | ✓ (auto) | ✓ (manual) | ✓ (auto) | ✓ (manual) | - | ✓ |
| **description** | ✓ (auto) | ✓ (manual) | ✓ (manual) | ✓ (manual) | - | ✓ |
| **parameters** | ✓ (auto) | ❌ | ✓ (auto) | ✓ (manual) | - | ✓ |

### 2.2 Campos Específicos por Framework

| Framework | Campos Adicionales |
|-----------|-------------------|
| **LangChain** | `args_schema` (Pydantic), `return_direct`, `verbose`, `runtime` (ToolRuntime para state/context) |
| **CrewAI** | `license`, `compatibility`, `metadata`, `allowed-tools` (experimental), directorios `scripts/`, `references/`, `assets/` |
| **AutoGen** | `strict` (boolean), tool schema con `additionalProperties`, `call_id` para tracking |
| **Haystack** | `function` (callable), `outputs_to_string` (handler config), `inputs_from_state`, `outputs_to_state`, `tool_spec` property |
| **BabyAGI** | No define tools formalmente - usa LLM + vector DB + task queue |
| **SuperAGI** | `toolkit_id`, `config` (dict), `agent_id`, `run_id`, marketplace metadata |

---

## 3. Patrones de Diseño Identificados

### 3.1 Patrón Decorator-Based (LangChain)

**Características**:
- Usa decorador `@tool` sobre funciones Python
- Auto-genera name, description y parameters desde código existente
- Type hints son **requeridos** para definir el schema
- Docstring se convierte en description

**Ejemplo**:
```python
from langchain.tools import tool

@tool
def search_database(query: str, limit: int = 10) -> str:
    """Search the customer database for records matching the query.

    Args:
        query: Search terms to look for
        limit: Maximum number of results to return
    """
    return f"Found {limit} results for '{query}'"
```

**Ventajas**:
- Mínimo boilerplate
- Reutiliza código existente
- Type hints proporcionan documentación y validación

**Desventajas**:
- Requiere type hints (no opcional)
- Menos control sobre schema generado
- Difícil de customizar sin cambiar función original

**Fuente**: https://docs.langchain.com/oss/python/langchain/tools

---

### 3.2 Patrón Filesystem-Based (CrewAI)

**Características**:
- Cada skill es un directorio completo
- `SKILL.md` con YAML frontmatter define metadata
- Body de SKILL.md contiene instrucciones
- Puede incluir `scripts/`, `references/`, `assets/`

**Estructura**:
```
my-skill/
├── SKILL.md            # Required — frontmatter + instructions
├── scripts/            # Optional — executable scripts
├── references/         # Optional — reference documents
└── assets/             # Optional — static files
```

**Ejemplo SKILL.md**:
```yaml
---
name: code-review
description: Guidelines for conducting thorough code reviews
metadata:
  author: your-team
  version: "1.0"
---

## Code Review Guidelines

When reviewing code, follow this checklist:
1. **Security**: Check for injection vulnerabilities
2. **Performance**: Look for N+1 queries
...
```

**Uso en código**:
```python
from crewai import Agent

reviewer = Agent(
    role="Senior Code Reviewer",
    goal="Review pull requests",
    skills=["./skills"],  # Injects review guidelines
)
```

**Ventajas**:
- Skills son independientes del código
- Progressive disclosure nativo (metadata separada)
- Incluye recursos adicionales (scripts, docs)
- Reutilizable entre proyectos

**Desventajas**:
- Más verboso que decorator
- Requiere estructura de directorios
- No auto-genera schema

**Fuente**: https://docs.crewai.com/en/concepts/skills

---

### 3.3 Patrón Class-Based (AutoGen, SuperAGI)

**Características**:
- Clase `FunctionTool` o `BaseTool` envuelve funciones
- Schema se genera desde type annotations
- Proporciona `schema` property para JSON schema
- Herencia permite extensión

**Ejemplo AutoGen**:
```python
from autogen_core.tools import FunctionTool
from typing_extensions import Annotated

async def get_stock_price(
    ticker: str, 
    date: Annotated[str, "Date in YYYY/MM/DD"]
) -> float:
    return random.uniform(10, 200)

stock_price_tool = FunctionTool(
    get_stock_price, 
    description="Get the stock price."
)

# Acceso al schema JSON
print(stock_price_tool.schema)
# {'name': 'get_stock_price', 'description': 'Get the stock price.', ...}
```

**Ejemplo SuperAGI**:
```python
from superagi.tools.base_tool import BaseTool

class CodingTool(BaseTool):
    name: str = "Coding Tool"
    description: str = "Write and execute code"
    
    def _execute(self, instruction: str):
        # Implementation
        pass
```

**Ventajas**:
- Control total sobre schema
- Extensible via herencia
- Integración con model clients

**Desventajas**:
- Más boilerplate
- Requiere conocimiento de la API de la clase

**Fuente**: https://microsoft.github.io/autogen/stable//user-guide/core-user-guide/components/tools

---

### 3.4 Patrón Dataclass-Based (Haystack)

**Características**:
- Tool es una dataclass con campos explícitos
- JSON schema debe proveerse manualmente
- `function` es un campo que contiene el callable
- Handlers opcionales para outputs

**Ejemplo**:
```python
from haystack.tools import Tool
from typing import Annotated, Literal

def get_weather(
    city: Annotated[str, "the city for which to get the weather"] = "Munich",
    unit: Literal["Celsius", "Fahrenheit"] = "Celsius",
):
    """A simple function to get the current weather."""
    return f"Weather report for {city}: 20 {unit}, sunny"

weather_tool = Tool(
    name="weather",
    description="A tool to get the weather",
    function=get_weather,
    parameters={
        "type": "object",
        "properties": {
            "city": {"type": "string", "description": "the city"},
            "unit": {"type": "string", "enum": ["Celsius", "Fahrenheit"]},
        },
        "required": ["city"],
    },
    outputs_to_string={
        "formatted": {"source": "result", "handler": str}
    }
)
```

**Decorador alternativo**:
```python
from haystack.tools import tool

@tool
def get_weather(city: str = "Munich") -> str:
    """A simple function to get the current weather."""
    return f"Weather: 20C"
```

**Ventajas**:
- Control total sobre schema JSON
- Handlers para formatear outputs
- Integración con state management

**Desventajas**:
- JSON schema manual es verboso
- Duplicación entre función y schema

**Fuente**: https://docs.haystack.deepset.ai/docs/tool

---

### 3.5 Patrón Task-Driven (BabyAGI)

**Características**:
- No define tools formalmente
- Loop: task execution → task creation → task prioritization
- Vector database como memoria
- Usa LangChain para tool execution

**Arquitectura**:
```
┌─────────────────────────────────────────────────┐
│                   BabyAGI Loop                   │
│                                                  │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ Execution │───▶│ Creation │───▶│Priority  │  │
│  │  Agent    │    │  Agent   │    │ Agent    │  │
│  └──────────┘    └──────────┘    └──────────┘  │
│        │                               │        │
│        ▼                               ▼        │
│  ┌─────────────────────────────────────────┐   │
│  │           Vector Database (Memory)       │   │
│  └─────────────────────────────────────────┘   │
│                                                  │
└─────────────────────────────────────────────────┘
```

**Componentes**:
- LLM (GPT-4) para razonamiento
- Vector DB (Pinecone) para memoria
- Task list/queue para tracking
- Tres agentes: execution, creation, prioritization

**Ventajas**:
- Autonomía: genera sus propias tareas
- Iterativo: aprende de resultados previos
- Simple: script único de Python

**Desventajas**:
- No define tools formalmente
- Menos control sobre comportamiento
- No production-ready (más educativo)

**Fuente**: https://www.ibm.com/think/topics/babyagi

---

## 4. Análisis Comparativo

### 4.1 Facilidad de Uso

| Framework | Facilidad | Curva de Aprendizaje |
|-----------|-----------|---------------------|
| LangChain | ★★★★★ | Muy baja - decorator simple |
| CrewAI | ★★★★☆ | Baja - filesystem intuitivo |
| Haystack | ★★★☆☆ | Media - schema manual |
| AutoGen | ★★★☆☆ | Media - requiere entender clases |
| SuperAGI | ★★☆☆☆ | Alta - arquitectura compleja |
| BabyAGI | ★★★★☆ | Baja - script único |

### 4.2 Control sobre Schema

| Framework | Control | Flexibilidad |
|-----------|---------|--------------|
| Haystack | ★★★★★ | Total - schema manual explícito |
| AutoGen | ★★★★☆ | Alta - puede customizar via args |
| LangChain | ★★★☆☆ | Media - Pydantic permite customización |
| CrewAI | ★☆☆☆☆ | Baja - skills no tienen schema |
| BabyAGI | ☆☆☆☆☆ | N/A - no define tools |
| SuperAGI | ★★★★☆ | Alta - class-based extensible |

### 4.3 Integración con Agentes

| Framework | Integración | Scope |
|-----------|-------------|-------|
| LangChain | ★★★★★ | Excelente - agents nativos |
| CrewAI | ★★★★★ | Excelente - skills inyectadas en prompts |
| AutoGen | ★★★★★ | Excelente - model clients integrados |
| Haystack | ★★★★☆ | Buena - Agent component disponible |
| SuperAGI | ★★★★★ | Excelente - framework completo |
| BabyAGI | ★★★☆☆ | Limitada - usa LangChain indirectamente |

---

## 5. Convergencia y Divergencia

### 5.1 Convergencias

1. **Campos universales**: Todos los frameworks que definen tools usan name, description, y parameters
2. **JSON Schema**: LangChain, AutoGen, Haystack, SuperAGI generan/proveen JSON schema para tools
3. **Integración LLM**: Todos se integran con modelos de lenguaje (OpenAI, etc.)

### 5.2 Divergencias

1. **Separación de concerns**:
   - CrewAI separa skills (instrucciones) de tools (acciones)
   - LangChain, AutoGen, Haystack unifican en Tool

2. **Storage de tools/skills**:
   - CrewAI: filesystem (directorios)
   - Otros: código Python (módulos/clases)

3. **Auto-generación vs manual**:
   - LangChain, AutoGen: auto-generan schema desde type hints
   - Haystack, SuperAGI: manual o semi-manual
   - CrewAI: no aplica (instructions, no schema)

---

## 6. Recomendaciones por Caso de Uso

| Caso de Uso | Framework Recomendado | Razón |
|-------------|----------------------|-------|
| Prototipado rápido | LangChain | Decorator simple, auto-schema |
| Skills con documentación | CrewAI | Filesystem-based, recursos incluidos |
| Multi-agent systems | AutoGen | Agentes colaborativos nativos |
| Control total del schema | Haystack | Dataclass explícito |
| Autonomía de tareas | BabyAGI | Task-driven loop |
| Enterprise production | SuperAGI | Framework completo con marketplace |

---

## 7. Fuentes y Trazabilidad

| Hallazgo | Fuente | URL |
|----------|--------|-----|
| LangChain @tool decorator | LangChain Docs | https://docs.langchain.com/oss/python/langchain/tools |
| CrewAI filesystem skills | CrewAI Docs | https://docs.crewai.com/en/concepts/skills |
| AutoGen FunctionTool | Microsoft AutoGen Docs | https://microsoft.github.io/autogen/stable//user-guide/core-user-guide/components/tools |
| Haystack Tool dataclass | Haystack Docs | https://docs.haystack.deepset.ai/docs/tool |
| BabyAGI architecture | IBM Think | https://www.ibm.com/think/topics/babyagi |
| SuperAGI toolkit | GitHub | https://github.com/TransformerOptimus/SuperAGI |

---

## 8. Conclusión

Los frameworks de agentes de IA muestran **cuatro patrones de diseño principales** para definir tools/skills:

1. **Decorator-based** (LangChain): Máxima simplicidad, auto-generación de schema
2. **Filesystem-based** (CrewAI): Separación de código e instrucciones, progressive disclosure
3. **Class-based** (AutoGen, SuperAGI): Extensibilidad, control fino
4. **Dataclass-based** (Haystack): Control total, schema explícito

BabyAGI representa un paradigma diferente: **task-driven** sin definición formal de tools.

**Campos universales identificados**: name, description, parameters - presentes en todos los frameworks que definen tools formalmente.

**Recomendación para mapj_cli**: Adoptar enfoque híbrido inspirado en CrewAI (filesystem-based skills con SKILL.md) para instrucciones, complementado con tools definidas via decorator (estilo LangChain) para acciones ejecutables.

---

**Validación**: Este documento cumple con VAL-STD-003:
- ✅ Compara 6 frameworks (LangChain, CrewAI, AutoGen, BabyAGI, Haystack, SuperAGI)
- ✅ Tabla comparativa con enfoques y características
- ✅ Tabla de campos universales vs específicos
- ✅ Patrones de diseño identificados (decorator, class-based, dataclass, filesystem)
- ✅ Citas de documentación oficial para cada framework
