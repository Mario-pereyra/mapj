# 📘 Guía de Usuario — `mapj confluence export`

> **Para:** Mario (usuario del CLI `mapj`)  
> **Versión:** 2.1 (post-refactorización agentic)  
> **Fecha:** Abril 2026

---

## ¿Qué es esto?

`mapj confluence export` exporta páginas de Confluence a Markdown optimizado para agentes IA.

- **Auto-Healing:** Reintenta automáticamente fallos de red (429/50x) con exponential backoff.
- **Worker Pool:** Exporta múltiples páginas en paralelo (10 workers concurrentes).
- **Markdown Singleton:** Conversión ultra-rápida sin reconstruir reglas por cada página.
- **Sin Basura:** No genera archivos `.debug` ni logs de errores manuales innecesarios.

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

---

## 2. Formatos de URL soportados

El CLI acepta **casi cualquier formato** de URL de Confluence:

| Formato | Ejemplo |
|---------|---------|
| **Page ID solo** | `22479548` |
| **URL Cloud** | `https://tu-empresa.atlassian.net/wiki/spaces/TEAM/pages/12345/Titulo` |
| **URL Server Display** | `https://tdn.totvs.com/display/framework/SDK+Microsiga+Protheus` |
| **ViewPage action** | `https://tdninterno.totvs.com/pages/viewpage.action?pageId=22479548` |

---

## 3. Comandos principales

### 3.1 Exportar una sola página (resultado en pantalla)

```bash
mapj confluence export 22479548
```

### 3.2 Exportar página a un directorio

```bash
mapj confluence export 22479548 --output-path ./docs
```

Crea la estructura:
```
docs/
  spaces/
    SPACE_KEY/
      README.md          ← índice del espacio
      pages/
        ID-slug.md       ← archivo markdown
  manifest.jsonl         ← metadata de todo lo exportado
```

### 3.3 Exportar con todos los hijos recursivamente (Concurrent)

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-descendants
```

### 3.4 Exportar también con attachments (imágenes, archivos)

```bash
mapj confluence export 22479548 \
  --output-path ./docs \
  --with-attachments
```

### 3.5 Exportar un espacio entero (Alta Velocidad)

```bash
mapj confluence export-space framework \
  --output-path ./docs
```

---

## 4. Observabilidad y Resiliencia

### Ver progreso

```bash
mapj confluence export 22479548 --output-path ./docs --verbose
```

### Auto-Healing

Ya no existe el comando `retry-failed`. Si la red falla temporalmente o el servidor te limita (429), la CLI **esperará y reintentará automáticamente hasta 3 veces** antes de rendirse.

---

## 5. Estructura de los archivos generados

Cada página exportada tiene front matter YAML con toda la metadata:

```markdown
---
page_id: "22479548"
title: "SX1 - Perguntas do usuário"
source_url: "https://tdn.totvs.com/pages/viewpage.action?pageId=22479548"
space_key: "framework"
labels: ["sx1", "dicionario"]
exported_at: "2026-03-28T20:48:15Z"
---
# SX1 - Perguntas do usuário

Contenido...
```

---

## 6. Resumen de flags por comando

### `mapj confluence export`

| Flag | Default | Descripción |
|------|---------|-------------|
| `--output-path PATH` | (stdout) | Directorio de salida. |
| `--format` | `markdown` | `markdown`, `html`, `json` |
| `--with-descendants` | `false` | Exporta hijos recursivamente. |
| `--with-attachments` | `false` | Descarga adjuntos. |
| `--verbose` | `false` | Muestra progreso detallado. |

### `mapj confluence export-space`

| Flag | Default | Descripción |
|------|---------|-------------|
| `--output-path PATH` | (requerido) | Directorio de salida. |
| `--with-attachments` | `false` | Descarga adjuntos. |
| `--verbose` | `false` | Muestra progreso. |
