---
name: analysis-worker
description: Worker para síntesis de análisis y producción de informes comparativos
---

# Analysis Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Usar para features que requieren:
- Sintetizar hallazgos de análisis previos
- Producir secciones del informe comparativo
- Consolidar evaluaciones y puntuaciones
- Generar recomendaciones basadas en evidencia

## Required Skills

None - Este worker sintetiza análisis existentes, no ejecuta herramientas externas.

## Work Procedure

### Para síntesis de código/funcionalidad/tests/docs:

1. **Leer análisis previos** - Los hallazgos están disponibles en los resultados de los subagentes:
   - Calidad código gem: 8.1/10
   - Calidad código main: 8.5/10
   - Tests: gem +339 líneas, +66.7% cobertura errors
   - Docs: gem significativamente superior
   - Refactorings: mejoras reales + 3 bugs críticos

2. **Consolidar en tabla comparativa** - Para cada criterio:
   - Puntuación main (1-10)
   - Puntuación gem (1-10)
   - Ganador
   - Justificación breve

3. **Citar evidencia específica** - Cada conclusión debe incluir:
   - Archivo afectado
   - Código/diff específico cuando sea relevante
   - Números concretos (líneas, cobertura, etc.)

4. **Declarar ganador** - Basado en puntuaciones, declarar ganador para el área.

### Para informe final:

1. **Consolidar todas las síntesis** - Integrar las 4 secciones previas.

2. **Calcular puntaje total** - Promedio ponderado de todos los criterios.

3. **Declarar ganador global** - Con justificación cuantitativa y cualitativa.

4. **Proponer plan de acción** - Una de tres opciones:
   - **Merge directo**: Si gem es claramente superior y sin bugs críticos
   - **Fix-and-merge**: Si gem es superior pero tiene bugs que deben fixearse primero
   - **Hybrid**: Si ambas tienen ventajas distintas que deben combinarse

5. **Escribir informe markdown** - Usar el formato especificado en AGENTS.md.

## Example Handoff

```json
{
  "salientSummary": "Sintetizó análisis de calidad de código para ambas ramas. main obtuvo 8.5/10 vs gem 8.1/10. Ganador: main por arquitectura más limpia y menor complejidad ciclomática.",
  "whatWasImplemented": "Sección 'Calidad de Código' del informe con tabla comparativa de 4 áreas (arquitectura, patrones, errores, dominios), puntuaciones por área, y observaciones específicas citando archivos.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [],
    "interactiveChecks": [
      {
        "action": "Verificar tabla incluye todas las áreas",
        "observed": "Tabla tiene 4 filas: arquitectura, patrones, errores, dominios"
      },
      {
        "action": "Verificar ganador declarado",
        "observed": "Sección finaliza con 'Ganador: main (8.5 vs 8.1)'"
      }
    ]
  },
  "tests": {
    "added": []
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Si los análisis previos no están disponibles o son insuficientes
- Si hay contradicciones entre hallazgos que no pueden resolverse
- Si se descubre información nueva que requiere investigación adicional
