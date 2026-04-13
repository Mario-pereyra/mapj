# Docs Worker Skill

skillName: docs-worker

## Overview
Actualiza documentación de repositorios manteniendo calidad consistente. Especializado en actualizar README, CHANGELOG, CONTRIBUTING, y archivos de skills.

## Procedures

### 1. Setup
1. Lee `AGENTS.md` del missionDir para entender scope
2. Lee `features.json` para ver qué documento actualizar
3. Lee el documento actual a actualizar
4. Verifica información de referencia (`.factory/library/architecture.md`)

### 2. Update Documentation
Para cada feature:
1. Leer documento actual
2. Identificar dónde agregar nueva información
3. Mantener estilo y formato existente
4. No inventar información - solo documentar lo que existe

### 3. Verification
- grep para verificar que información fue agregada
- Revisar que formato sea consistente
- Verificar que no se perdió información existente

## Conventions

### Estilo README.md
```markdown
| Task | Command | Output |
|------|---------|--------|
| Describe task | `mapj cmd` | Output description |
```

### Sección Detallada README
```markdown
### mapj cmd

Description of what the command does.

EXAMPLES:
  mapj cmd --flag value

OUTPUT SCHEMA:
  {"ok":true,"command":"...","result":{...}}
```

### CHANGELOG Entry
```markdown
## [v0.3.0] - YYYY-MM-DD

### Added
- Feature descriptions

### Changed
- Updates
```

## Output
Al completar:
- Documentos actualizados
- git commit con cambios
- Handoff confirmando qué se actualizó
