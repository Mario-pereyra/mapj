# 📘 Guía de Usuario — `mapj confluence export`

> **Para:** Mario (usuario del CLI `mapj`)  
> **Versión:** 2.0 (post-migración a Go)  
> **Fecha:** Marzo 2026

---

## ¿Qué es esto?

`mapj confluence export` es el reemplazo completo de la GUI Python. Exporta páginas de Confluence a Markdown con:

- Front matter YAML con metadata completa
- Conversión fiel del HTML de Confluence (macros, tablas, alertas, código)
- Soporte para exportar páginas individuales, con sus hijos, o espacios enteros
- Estructura de archivos organizada para consumo por LLMs o agentes
- Logs estructurados para debuguear fallos

---

## 1. Primer uso — Login

**Solo hay que hacerlo una vez.** Las credenciales se guardan cifradas localmente con el `authType` correcto.

El CLI **auto-detecta** qué tipo de autenticación usar según la URL:

| URL contiene | Auth detectada | Header enviado |
|---|---|---|
| `atlassian.net` | `basic` (email + API token) | `Authorization: Basic base64(email:token)` |
| cualquier otra | `bearer` (PAT) | `Authorization: Bearer TOKEN` |

> ✅ **No necesitás pensar en esto.** Si tu URL es `tdninterno.totvs.com` o `tdn.totvs.com`, el CLI elige `bearer` solo.

### Para `tdninterno.totvs.com` (instancia interna — Bearer PAT)

```bash
# Solo el token. La URL no es atlassian.net → auto-detecta Bearer
mapj auth login confluence \
  --url "https://tdninterno.totvs.com" \
  --token "TU_TOKEN_PAT"

# Output: Confluence login successful (auth: bearer)
```

### Para `tdn.totvs.com` (portal público)

El portal público de TDN no requiere auth. El CLI usa un fallback de HTML scraping automático para resolver las URLs.

### Para Confluence Cloud (`company.atlassian.net`)

```bash
# La URL contiene atlassian.net → auto-detecta Basic, requiere --username
mapj auth login confluence \
  --url "https://tu-empresa.atlassian.net" \
  --username "tu-email@empresa.com" \
  --token "TU_API_TOKEN_CLOUD"

# Output: Confluence login successful (auth: basic)
```

### Override manual (casos especiales)

Si tu servidor no es `atlassian.net` pero requiere Basic Auth, usá `--auth-type`:

```bash
mapj auth login confluence \
  --url "https://mi-confluence.empresa.com" \
  --username "usuario" \
  --token "password_o_token" \
  --auth-type basic
```

### Verificar que el login funcionó

```bash
mapj auth status
```

Salida esperada:
```
Authentication Status:
  TDN:        ✓ configured
  Confluence: ✓ configured
  Protheus:   ✗ not configured
```

> ⚠️ **Si tenés credenciales anteriores guardadas** (antes de esta versión), re-hacé el login para que queden con el `authType` correcto.

## 2. Formatos de URL soportados

El CLI acepta **casi cualquier formato** de URL de Confluence:

| Formato | Ejemplo |
|---------|---------|
| **Page ID solo** | `22479548` |
| **URL Cloud** | `https://empresa.atlassian.net/wiki/spaces/TEAM/pages/12345/Titulo` |
| **URL Server Display** | `https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus` |
| **URL pública con prefijo** | `https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus` |
| **ViewPage action** | `https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548` |
| **ReleaseView action** | `https://tdninterno.totvs.com/pages/releaseview.action?pageId=22479548` |

---

## 3. Comandos principales

### 3.1 Exportar una sola página (resultado en pantalla)

```bash
mapj confluence export 22479548
```

Imprime el Markdown directamente al terminal con el envelope JSON.

### 3.2 Exportar página a un directorio

```bash
mapj confluence export 22479548 --output-path ./docs
```

Crea la estructura:
```
docs/
  spaces/
    framework/
      README.md          ← índice del espacio
      pages/
        22479548-sx1-perguntas-do-usuario.md
  manifest.jsonl         ← metadata de todo lo exportado
  export-errors.jsonl    ← registro de errores (si los hay)
```

### 3.3 Exportar desde URL completa

```bash
mapj confluence export \
  "https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus" \
  --output-path ./docs
```

### 3.4 Exportar con todos los hijos recursivamente

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-descendants
```

Exporta la página + TODAS sus páginas hijas (recurse completo). Probado con 785 páginas en ~6 minutos.

### 3.5 Exportar también con attachments (imágenes, archivos)

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-attachments
```

```bash
# Combinado: hijos + attachments
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-descendants \
  --with-attachments
```

Los attachments se guardan en `docs/spaces/SPACE/attachments/PAGE_ID/archivo.ext`

### 3.6 Exportar un espacio entero

```bash
mapj confluence export-space framework \
  --output-path ./docs
```

Con attachments:
```bash
mapj confluence export-space framework \
  --output-path ./docs \
  --with-attachments
```

---

## 4. Debugging y observabilidad

### Ver progreso detallado

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --verbose
```

### Guardar el HTML crudo para inspección

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --debug
```

Guarda el HTML en `docs/.debug/22479548-body.html`.

### Dump completo de diagnóstico

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --dump-debug
```

Guarda raw HTML + storage format + markdown convertido + metadata en `docs/.debug/`.

### Re-exportar solo las páginas que fallaron

Si una exportación grande tuvo algunos errores, no hace falta rehacer todo:

```bash
mapj confluence retry-failed \
  --output-path ./docs
```

Solo los errores de un tipo específico:
```bash
mapj confluence retry-failed \
  --output-path ./docs \
  --error-code HTTP_TIMEOUT
```

Con attachments en el retry:
```bash
mapj confluence retry-failed \
  --output-path ./docs \
  --with-attachments
```

---

## 5. Estructura de los archivos generados

### Markdown de cada página

Cada página exportada tiene front matter YAML con toda la metadata:

```markdown
---
page_id: "22479548"
title: "SX1 - Perguntas do usuário"
source_url: "https://tdn.totvs.com/pages/viewpage.action?pageId=22479548"
space_key: "framework"
space_name: "Frameworksp"
labels:
  - "sx1"
  - "dicionario"
updated_at: "2025-08-05T09:30:23.513-03:00"
author: "Sandro Constancio Ferreira"
version: 3
exported_at: "2026-03-28T20:48:15Z"
---
# SX1 - Perguntas do usuário

Contenido en Markdown...
```

### manifest.jsonl

Cada página exportada genera una línea JSON en `manifest.jsonl`. Ideal para que los agentes hagan lookup por ID, título, o espacio sin leer todos los archivos:

```jsonl
{"page_id":"22479548","title":"SX1 - Perguntas do usuário","slug":"sx1-perguntas-do-usuario","source_url":"...","space_key":"framework","export_path":"spaces/framework/pages/22479548-sx1-perguntas-do-usuario.md","exported_at":"2026-03-28T20:48:15Z"}
```

### export-errors.jsonl

Cada error queda registrado con el comando exacto para re-intentar:

```jsonl
{"ts":"2026-03-28T20:00:00Z","page_id":"123456","title":"Página X","phase":"FETCH","error_code":"HTTP_TIMEOUT","message":"request timeout after 30s","retry_cmd":"mapj confluence export 123456 --output-path ./docs"}
```

---

## 6. Tips y casos comunes

### "Me dan 401 en tdninterno"

Tu token PAT puede haber expirado o lo configuraste con `--username` (activa Basic Auth que el servidor rechaza).

```bash
# Re-login correcto
mapj auth login confluence \
  --url "https://tdninterno.totvs.com" \
  --token "TOKEN_ACTUALIZADO"
```

### "No sé el page ID, solo tengo el título"

Usá la URL display completa. El CLI la resuelve automáticamente:

```bash
mapj confluence export \
  "https://tdn.totvs.com/display/framework/REST+API+Guide" \
  --output-path ./docs
```

Si la API falla, hay un fallback que raspa el HTML de la página para extraer el ID (como hacía la GUI Python).

### "La página tiene muchos hijos pero no quiero todos"

Exportá solo la página raíz sin `--with-descendants`:

```bash
mapj confluence export 152798711 --output-path ./docs
```

### "Quiero leer el output con jq"

Los resultados inline (sin `--output-path`) son JSON puro:

```bash
mapj confluence export 22479548 | jq '.result.content'
```

### Exportación masiva de un módulo completo

```bash
# 1. Exportar toda la documentación del SDK (785 páginas)
mapj confluence export \
  "https://tdn.totvs.com/display/public/framework/SDK+Microsiga+Protheus" \
  --output-path ~/docs/protheus-sdk \
  --with-descendants \
  --verbose

# 2. Ver el índice generado
cat ~/docs/protheus-sdk/spaces/framework/README.md

# 3. Consultar el manifest para buscar una página
grep "REST API" ~/docs/protheus-sdk/manifest.jsonl | jq .

# 4. Si hubo errores, reintentar solo esos
mapj confluence retry-failed --output-path ~/docs/protheus-sdk
```

---

## 7. Resumen de flags por comando

### `mapj confluence export`

| Flag | Default | Descripción |
|------|---------|-------------|
| `--output-path PATH` | (inline) | Directorio de salida. Sin esto, imprime en pantalla |
| `--format` | `markdown` | `markdown`, `html`, `json` |
| `--with-descendants` | `false` | También exporta todas las páginas hijas |
| `--with-attachments` | `false` | Descarga imágenes y archivos adjuntos |
| `--verbose` | `false` | Muestra progreso detallado |
| `--debug` | `false` | Guarda HTML crudo en `.debug/` |
| `--dump-debug` | `false` | Dump completo para diagnóstico |

### `mapj confluence export-space`

| Flag | Default | Descripción |
|------|---------|-------------|
| `--output-path PATH` | (requerido) | Directorio de salida |
| `--with-attachments` | `false` | Descarga imágenes y archivos |
| `--verbose` | `false` | Muestra progreso |
| `--debug` | `false` | HTML crudo en `.debug/` |

### `mapj confluence retry-failed`

| Flag | Default | Descripción |
|------|---------|-------------|
| `--output-path PATH` | `.` | Directorio con el `export-errors.jsonl` |
| `--error-code CODE` | (todos) | Filtrar por código de error específico |
| `--with-attachments` | `false` | Incluir attachments en el retry |
| `--verbose` | `false` | Muestra progreso |
