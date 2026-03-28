# CQL Reference — TOTVS TDN Search

Full CQL (Confluence Query Language) reference for TDN (`tdn.totvs.com`).

---

## Fields Available

### Text fields
| Field | What it searches | Operators | Note |
|---|---|---|---|
| `siteSearch` | Title + body + labels | `~` only | ⭐ **TDN custom field — use this** |
| `text` | Body only | `~` `!~` | Standard Confluence |
| `title` | Title only | `~` `!~` | Standard Confluence |

### Equality fields
| Field | Example |
|---|---|
| `space` | `space = "PROT"` |
| `type` | `type = page` |
| `label` | `label = "versao_12"` |
| `creator` | `creator = "user.name"` |
| `contributor` | `contributor = "user.name"` |
| `ancestor` | `ancestor = 187531295` |
| `parent` | `parent = 237387586` |

### Date fields
| Field | Example |
|---|---|
| `created` | `created >= "2024-01-01"` |
| `lastmodified` | `lastmodified >= now("-1m")` |

---

## Operators

| Operator | Example |
|---|---|
| `~` CONTAINS | `siteSearch ~ "advpl"` |
| `!~` NOT CONTAINS | `siteSearch !~ "deprecated"` |
| `=` EQUALS | `space = PROT` |
| `!=` NOT EQUALS | `type != blogpost` |
| `IN` | `space IN ("PROT", "LDT")` |
| `NOT IN` | `type NOT IN (attachment)` |
| `>` `>=` `<` `<=` | `lastmodified >= "2024-01-01"` |
| `AND` / `OR` / `NOT` | `siteSearch ~ "api" AND type = page` |

---

## Date Functions

| Function | Meaning |
|---|---|
| `now("-1w")` | 1 week ago |
| `now("-4d")` | 4 days ago |
| `now("-1m")` | 1 month ago |
| `now("-1y")` | 1 year ago |
| `startOfDay()` | Today 00:00 |
| `startOfMonth()` | 1st of month |
| `startOfYear()` | Jan 1st |

---

## mapj `--since` Flag → CQL Date Mapping

| `--since` value | CQL generated |
|---|---|
| `1w` | `lastmodified >= now("-1w")` |
| `4d` | `lastmodified >= now("-4d")` |
| `2m` | `lastmodified >= now("-2m")` |
| `1y` | `lastmodified >= now("-1y")` |
| `2024-01-01` | `lastmodified >= "2024-01-01"` |

---

## Content Types (`--type`)

| `--type` value | CQL type | Content |
|---|---|---|
| `page` (default) | `page` | Documentation pages |
| `blogpost` | `blogpost` | Blog posts |
| `attachment` | `attachment` | Files, PDFs, ZIPs, PRW |
| `comment` | `comment` | Page comments |

---

## CQL Query Examples

```
# All AdvPL pontos de entrada in Protheus 12
siteSearch ~ "ponto de entrada" AND space = "PROT" AND label = "versao_12" AND type = page

# API documentation updated in the last month
siteSearch ~ "REST API" AND space IN ("PROT", "TASS") AND lastmodified >= now("-1m") AND type = page

# All pages under a specific section (Pontos de Entrada P12)
ancestor = 811253174 AND type = page

# Attachments (apostilas/PDFs) in Protheus space
siteSearch ~ "apostila" AND space = "PROT" AND type = attachment

# Multi-label AND filter
label = "ponto_de_entrada" AND label = "versao_12" AND space = "PROT"

# All reference docs
label = "documento_de_referencia" AND space = "PROT" AND type = page

# Fuzzy search for mis-spellings
siteSearch ~ "advnpl~" AND space = "PROT"

# TLPP or AdvPL
(siteSearch ~ "advpl" OR siteSearch ~ "tlpp") AND space = "PROT" AND type = page
```

---

## Text Search Modifiers

```
"exact phrase"            → exact phrase match
word1 AND word2           → both words must appear
word1 OR word2            → either word
word1 NOT word2           → word1 without word2
adv*                      → wildcard (advpl, advplide...)
advp?l                    → single char wildcard
advnpl~                   → fuzzy (typo tolerance)
"ponto entrada"~2         → words within 2 words of each other
```
