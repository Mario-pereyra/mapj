# 📘 Guía de Usuario — `mapj protheus`

> **Para:** Mario (usuario del CLI `mapj`)  
> **Versión:** 3.1 — Multi-perfil + output-file  
> **Fecha:** Marzo 2026

---

## ¿Qué es esto?

`mapj protheus` ejecuta queries SELECT sobre las bases de datos SQL Server del ERP Protheus.
Permite gestionar **múltiples conexiones nombradas** (perfiles) que se guardan cifradas,
sin necesidad de reescribir todos los datos cada vez que cambiás de DB.

---

## Conexiones disponibles

### Servidor TOTALPEC (`192.168.99.102`) — VPN TOTALPEC

| Perfil | Database | Usuario |
|--------|----------|---------|
| **TOTALPEC_BIB** ← default | `P1212410_BIB` | `P1212410_BIB` |
| TOTALPEC_PRD | `P1212410_PRD` | `P1212410_PRD` |
| TOTALPEC_DES | `P1212410_DES` | `P1212410_DES` |
| TOTALPEC_DESII | `P1212410_DESII` | `P1212410_DESII` |

### Servidor UNION — VPN UNION

| Perfil | Server | Database |
|--------|--------|----------|
| UNION_BIB | `192.168.7.97` | `P1212410_BIB` |
| UNION_PRD | `192.168.7.215` | `P1212410_PRD` |
| UNION_UPG | `192.168.7.135` | `P1212410_UPG` |

---

## 1. Gestión de perfiles (conexiones)

### Ver todos los perfiles registrados

```bash
mapj protheus connection list
```

Salida:
```
Registered profiles:
  * TOTALPEC_BIB          192.168.99.102:1433 / P1212410_BIB / user: P1212410_BIB  ← ACTIVE
    TOTALPEC_DES          192.168.99.102:1433 / P1212410_DES / user: P1212410_DES
    TOTALPEC_DESII        192.168.99.102:1433 / P1212410_DESII / user: P1212410_DESII
    TOTALPEC_PRD          192.168.99.102:1433 / P1212410_PRD / user: P1212410_PRD
    UNION_BIB             192.168.7.97:1433 / P1212410_BIB / user: P1212410_BIB
    UNION_PRD             192.168.7.215:1433 / P1212410_PRD / user: P1212410_PRD
    UNION_UPG             192.168.7.135:1433 / P1212410_UPG / user: P1212410_UPG

Total: 7 profile(s). Active: TOTALPEC_BIB
```

El `*` marca la conexión activa.

### Ver el estado general

```bash
mapj auth status
```

```json
{"ok":true,"result":{"tdn":{"authenticated":false},"confluence":{"authenticated":true,"url":"https://tdninterno.totvs.com"},"protheus":{"authenticated":true,"activeProfile":"TOTALPEC_BIB","server":"192.168.99.102","database":"P1212410_BIB","totalProfiles":7}}}
```

> Tip: usar `-o json` para verlo indentado: `mapj auth status -o json`

### Agregar un nuevo perfil

```bash
mapj protheus connection add <nombre> \
  --server <ip> \
  --port 1433 \
  --database <db> \
  --user <user> \
  --password <pass> \
  [--use]   # opcional: activarlo inmediatamente
```

Ejemplo — agregar y activar un nuevo entorno:
```bash
mapj protheus connection add MI_NUEVO_ENV \
  --server 192.168.99.102 \
  --database P1212410_NUEVO \
  --user P1212410_NUEVO \
  --password P1212410_NUEVO \
  --use
```

> El primer perfil que agregás queda activo automáticamente.  
> Sin `--use`, el nuevo perfil queda registrado pero no cambia la conexión activa.

### Cambiar la conexión activa

```bash
# Sin tocar credenciales, solo cambiá el puntero
mapj protheus connection use TOTALPEC_PRD
```

Salida:
```
✓ Switched active profile: TOTALPEC_BIB → TOTALPEC_PRD
  Server: 192.168.99.102:1433 / Database: P1212410_PRD
```

### Ver detalles de un perfil (password enmascarado)

```bash
mapj protheus connection show                # muestra el activo
mapj protheus connection show TOTALPEC_PRD   # muestra uno específico
```

```
Profile: TOTALPEC_PRD  ← ACTIVE
  Server:   192.168.99.102
  Port:     1433
  Database: P1212410_PRD
  User:     P1212410_PRD
  Password: P1**********RD
```

### Probar si la conexión está online

```bash
mapj protheus connection ping               # prueba el activo
mapj protheus connection ping TOTALPEC_BIB  # prueba uno específico
```

✅ Si está disponible:
```
Pinging TOTALPEC_BIB → 192.168.99.102:1433/P1212410_BIB ...
✓ OK — 155ms  [192.168.99.102:1433 / P1212410_BIB]
```

❌ Si falla (con hint de VPN):
```
Pinging UNION_BIB → 192.168.7.97:1433/P1212410_BIB ...
✗ FAILED (ping failed: unable to open tcp connection...)

💡 Suggestions:
   1. Verify you are connected to the VPN for this server:
      UNION servers (192.168.7.97) — connect to the UNION VPN
   2. Verify credentials: user='P1212410_BIB', database='P1212410_BIB'
   3. Verify the server is reachable: ping 192.168.7.97
```

### Eliminar un perfil

```bash
mapj protheus connection remove TOTALPEC_DESII
```

> Si era el activo, auto-selecciona otro perfil disponible.  
> Si era el último, queda sin conexión activa.

---

## 2. Ejecutar queries

### Sintaxis

```bash
mapj protheus query "<SQL>" [--format json|csv] [--max-rows N] [--connection NOMBRE]
```

### Flags

| Flag | Default | Descripción |
|------|---------|-------------|
| `--format` | `json` | `json` (estructurado) o `csv` (RFC 4180, con escape correcto) |
| `--max-rows` | `10000` | Límite de filas (client-side). `0` = sin límite |
| `--connection` | (activo) | Ejecutar contra un perfil específico SIN cambiar el activo |
| `--output-file` | stdout | Escribir resultado a un archivo en lugar de stdout (stdout solo recibe resumen) |

### Queries básicas

```bash
# Verificar cuál DB está activa
mapj protheus query "SELECT DB_NAME() AS bd_activa, @@SERVERNAME AS servidor"

# Contar registros
mapj protheus query "SELECT COUNT(*) AS total FROM SA1010"

# Primeras N filas con columnas específicas
mapj protheus query "SELECT TOP 10 A1_COD, A1_LOJA, A1_NOME FROM SA1010"

# Con filtro
mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL != '1'"
```

### Consultar otro perfil SIN cambiar el activo

```bash
# Estoy en TOTALPEC_BIB pero quiero ver algo en PRD
mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD
```

Ideal para comparar datos entre entornos sin hacer switch.

### Paginación

```bash
# TOP en SQL (DB-side, más eficiente)
mapj protheus query "SELECT TOP 100 * FROM SA1010 ORDER BY A1_COD"

# OFFSET para páginas siguientes
mapj protheus query "SELECT * FROM SA1010 ORDER BY A1_COD OFFSET 100 ROWS FETCH NEXT 100 ROWS ONLY"
```

### CTEs (WITH)

```bash
mapj protheus query "
  WITH activos AS (
    SELECT A1_COD, A1_NOME FROM SA1010 WHERE A1_MSBLQL != '1'
  )
  SELECT COUNT(*) AS total FROM activos
"
```

### Explorar el schema

```bash
# Todas las tablas disponibles
mapj protheus query "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"

# Estructura de una tabla
mapj protheus query "SELECT COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'SA1010' ORDER BY ORDINAL_POSITION"
```

### Formato CSV

```bash
mapj protheus query "SELECT TOP 100 A1_COD, A1_NOME FROM SA1010" --format csv
```

El CSV es RFC 4180: los campos con comas, comillas o saltos de línea se escapan correctamente.

### Guardar resultado en archivo (para queries grandes)

Cuando una query retorna muchas filas, el LLM puede quedar sin contexto. Usá `--output-file`:

```bash
# Solo el resumen va a stdout, el resultado completo al archivo
mapj protheus query "SELECT * FROM SA1010" --output-file ./sa1010.json
# stdout: {"rows": 1500, "columns": 45, "format": "json", "output_file": "./sa1010.json"}

# CSV masivo
mapj protheus query "SELECT * FROM SA1010" --format csv --output-file ./sa1010.csv
```

---

## 3. Restricciones de seguridad

Solo queries de **lectura**. Bloqueado antes de llegar al servidor:

| Categoría | Keywords bloqueados |
|-----------|---------------------|
| DML | `INSERT`, `UPDATE`, `DELETE`, `MERGE` |
| DDL | `CREATE`, `ALTER`, `DROP`, `TRUNCATE` |
| DCL | `GRANT`, `REVOKE`, `DENY` |
| Ejecución | `EXEC`, `EXECUTE` |
| Movimiento | `INTO` ← bloquea SELECT INTO también |
| Backup | `BACKUP`, `RESTORE` |

La query debe empezar con `SELECT` o `WITH`.

---

## 4. Errores comunes

| Error / Mensaje | Causa | Solución |
|----------------|-------|----------|
| `NOT_AUTHENTICATED` | Sin perfil activo | `mapj protheus connection add ...` |
| `PROFILE_NOT_FOUND` | Nombre de `--connection` incorrecto | `mapj protheus connection list` |
| `validation error: forbidden keyword` | Query con INSERT/UPDATE/etc. | Solo SELECT |
| `ping failed: i/o timeout` | Servidor inaccesible | Verificar VPN (ver hint del mensaje) |
| `login error` | Usuario/contraseña incorrectos | Verificar credenciales del perfil |
| `Invalid object name 'SA1010'` | Tabla no existe en esa DB | Verificar que estás en la DB correcta |

---

## 5. Resumen de comandos

### Gestión de perfiles

| Comando | Descripción |
|---------|-------------|
| `mapj protheus connection list` | Listar todos los perfiles (activo marcado con `*`) |
| `mapj protheus connection add <name> --server ... --database ... --user ... --password ...` | Registrar nuevo perfil |
| `mapj protheus connection use <name>` | Cambiar perfil activo |
| `mapj protheus connection show [name]` | Ver detalles de un perfil |
| `mapj protheus connection ping [name]` | Probar conectividad (con VPN hint si falla) |
| `mapj protheus connection remove <name>` | Eliminar perfil |

### Queries

| Comando | Descripción |
|---------|-------------|
| `mapj protheus query "SELECT ..."` | Ejecutar query en el perfil activo |
| `mapj protheus query "..." --connection NOMBRE` | Ejecutar en un perfil específico sin cambiar activo |
| `mapj protheus query "..." --format csv` | Resultado como CSV (RFC 4180) |
| `mapj protheus query "..." --max-rows 500` | Limitar filas (client-side) |
| `mapj protheus query "..." --output-file ./result.json` | Escribir a archivo (stdout solo resumen) |
| `mapj auth status` | Ver perfil activo y total de perfiles |
