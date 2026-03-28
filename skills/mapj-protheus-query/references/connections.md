# Protheus Known Connections Reference

All pre-registered connection profiles for this environment.

---

## TOTALPEC Environment — Server: `192.168.99.102:1433`

> VPN required: TOTALPEC VPN

| Profile Name | Database | User | Purpose |
|---|---|---|---|
| **TOTALPEC_BIB** ← default | `P1212410_BIB` | `P1212410_BIB` | BI / Consulting queries |
| TOTALPEC_PRD | `P1212410_PRD` | `P1212410_PRD` | Production |
| TOTALPEC_DES | `P1212410_DES` | `P1212410_DES` | Development |
| TOTALPEC_DESII | `P1212410_DESII` | `P1212410_DESII` | Development II |

## UNION Environment

> VPN required: UNION VPN

| Profile Name | Server | Database | User |
|---|---|---|---|
| UNION_BIB | `192.168.7.97:1433` | `P1212410_BIB` | `P1212410_BIB` |
| UNION_PRD | `192.168.7.215:1433` | `P1212410_PRD` | `P1212410_PRD` |
| UNION_UPG | `192.168.7.135:1433` | `P1212410_UPG` | `P1212410_UPG` |

---

## VPN Detection by IP Range

The CLI auto-detects which VPN hint to show based on server IP:

| IP Range | Environment | VPN Needed |
|---|---|---|
| `192.168.99.x` | TOTALPEC | TOTALPEC VPN |
| `192.168.7.x` | UNION | UNION VPN |

---

## Connection String Format (internal)

```
server=HOST;port=PORT;database=DATABASE;user id=USER;password=PASS;encrypt=disable
```

`encrypt=disable` is required for internal servers without TLS certificates.

---

## Re-registering All Profiles

```bash
# TOTALPEC
mapj protheus connection add TOTALPEC_BIB   --server 192.168.99.102 --port 1433 --database P1212410_BIB   --user P1212410_BIB   --password P1212410_BIB   --use
mapj protheus connection add TOTALPEC_PRD   --server 192.168.99.102 --port 1433 --database P1212410_PRD   --user P1212410_PRD   --password P1212410_PRD
mapj protheus connection add TOTALPEC_DES   --server 192.168.99.102 --port 1433 --database P1212410_DES   --user P1212410_DES   --password P1212410_DES
mapj protheus connection add TOTALPEC_DESII --server 192.168.99.102 --port 1433 --database P1212410_DESII --user P1212410_DESII --password P1212410_DESII

# UNION
mapj protheus connection add UNION_BIB --server 192.168.7.97  --port 1433 --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB
mapj protheus connection add UNION_PRD --server 192.168.7.215 --port 1433 --database P1212410_PRD --user P1212410_PRD --password P1212410_PRD
mapj protheus connection add UNION_UPG --server 192.168.7.135 --port 1433 --database P1212410_UPG --user P1212410_UPG --password P1212410_UPG
```
