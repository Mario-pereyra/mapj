# TOTVS CST Search API — Reverse Engineering Reference

> **Target**: `https://totvscst.zendesk.com/hc/en-us#/search`  
> **Backend**: `https://ti-services.totvs.com.br` (custom Elasticsearch aggregator)  
> **Date**: 2026-04-27  
> **Status**: Fully verified — all endpoints tested with cURL, HTTP 200

---

## 1. Architecture Overview

This is **NOT** a standard Zendesk Guide Help Center. TOTVS built a custom **multi-source search aggregator** on top of Elasticsearch that unifies content from 4 independent sources:

```
┌─────────────────────────────────────────────────────┐
│  totvscst.zendesk.com/hc/en-us (SPA frontend)       │
│  Hash-based routing: #/search?query=...             │
└────────────────┬────────────────────────────────────┘
                 │ POST /cst/BUSCA
                 ▼
┌─────────────────────────────────────────────────────┐
│  ti-services.totvs.com.br (Elasticsearch backend)   │
│  Index: cst_totvs                                    │
│  ┌──────────┬──────────┬──────────┬──────────┐     │
│  │ Zendesk  │ Central  │   TDN    │ YouTube  │     │
│  │ Articles │ Downloads│(Confl.)  │  Videos  │     │
│  └──────────┴──────────┴──────────┴──────────┘     │
└─────────────────────────────────────────────────────┘
```

### Content Source Mapping

| Source | `_id` prefix | `search_type` | Canonical URL pattern |
|--------|-------------|---------------|----------------------|
| Zendesk Guide | `z_` | `zendesk` | `https://centraldeatendimento.totvs.com/hc/{locale}/articles/{id}-{slug}` |
| Central de Download | `cd_` | `central` | `https://suporte.totvs.com/portal/p/10098/download?e={id}` |
| TDN (Confluence) | `c_` | `confluence` | `https://tdn.totvs.com/pages/viewpage.action?pageId={id}` |
| YouTube | `y_` | `youtube` | `https://youtube.com/watch?v={video_id}` |

**Extract real ID**: strip the prefix (`z_`, `cd_`, `c_`, `y_`) from the `_id` field.

---

## 2. Endpoints

### 2.1 `POST /cst/BUSCA` — Search

**Full URL**: `https://ti-services.totvs.com.br/cst/BUSCA`  
**Method**: `POST`  
**Content-Type**: `application/json`  
**CORS**: `access-control-allow-origin: *`  
**Auth**: None required

#### Request Headers

| Header | Value | Required |
|--------|-------|----------|
| `Content-Type` | `application/json` | Yes |
| `Origin` | `https://totvscst.zendesk.com` | Yes |
| `Referer` | `https://totvscst.zendesk.com/` | Recommended |

#### Request Body Schema

```json
{
  "query": "protheus",                  // * Search query text
  "assunto": "protheus",                //   Secondary keyword filter (refines within results)
  "sort_order": "score",                //   "score" (relevance) | "updated_at" (date)
  "perPage": "10",                      //   10 | 20 | 50 | 100
  "page": "1",                          //   Page number (1-based, optional, default=1)
  "types": [                            // * Content sources to search
    "zendesk",                          //     Zendesk articles
    "central",                          //     Central de Download
    "confluence",                       //     TDN documentation
    "youtube"                           //     YouTube videos
  ],
  "langs": ["es", "pt-br", "en-us"],    // * Language filters
  "sections": [],                       //   Section IDs to filter (integers)
  "produtos": [],                       //   Product keys to filter (strings)
  "labels": [],                         //   Label IDs to filter (strings)
  "playlists": [],                      //   YouTube playlist IDs (strings)
  "lines": [],                          //   Download line IDs (strings like "000052")
  "environments": [],                   //   Environment IDs (strings like "04.2507")
  "versions": [],                       //   Version IDs (strings like "12.1.2210")
  "highlightResults": true,             //   Include HTML highlight snippets in response
  "hostURL": "...",                     //   Full page URL (for analytics, optional)
  "datainicial": "2024-01-01",          //   Date range start (YYYY-MM-DD, optional)
  "datafinal": "2025-12-31"             //   Date range end (YYYY-MM-DD, optional)
}
```

- `*` = required fields (minimum: `query`, `perPage`, `types`, `langs`)
- All array fields default to `[]`
- `page` defaults to `1`
- `sort_order` defaults to `"score"`

#### Response Schema

```json
{
  "pages": 3727,           // Total pages
  "page": 1,               // Current page
  "count": 11180,          // Total hits
  "took": 4,               // Response time in ms (Elasticsearch)
  "hits": [
    {
      "_index": "cst_totvs",
      "_type": "articles",
      "_id": "z_20884859553943",
      "_score": 15.497806,
      "_source": {
        "hideDescription": false,
        "updated_at": "2024-01-26T15:25:55Z",
        "html_url": "https://centraldeatendimento.totvs.com/hc/es/articles/20884859553943-...",
        "title_name": "FRAMEWORK - Framework ...",
        "title": "FRAMEWORK - Framework (Línea Protheus) - MI - ...",
        "body": " ... full body text with HTML ... ",
        "thumbnails": null,
        "search_type": "zendesk"
      },
      "highlight": {
        "body": ["... <b>Protheus</b> ..."]
      },
      "sort": [15.497806, 1706282755000]
    }
  ]
}
```

**Key fields per hit**:

| Field | Description |
|-------|-------------|
| `_source.html_url` | Canonical URL to the content |
| `_source.title` | Article/video title |
| `_source.body` | Full body text (HTML) |
| `_source.updated_at` | Last update timestamp (ISO 8601) |
| `_source.search_type` | Content source (`zendesk`, `central`, `confluence`, `youtube`) |
| `_source.thumbnails` | Thumbnail URLs (only for YouTube) |
| `_score` | Relevance score from Elasticsearch |
| `highlight.body[]` | HTML snippets with `<b>` highlighting |

---

### 2.2 `GET /cst/BUSCA_FILTROS` — Filter Metadata

**Full URL**: `https://ti-services.totvs.com.br/cst/BUSCA_FILTROS`  
**Method**: `GET`  
**Response Size**: ~7.5 MB (cached, ETag supported with `If-None-Match`)

#### Response Schema

```json
{
  "categorias": [
    {
      "id": 115001937708,
      "name": "Administrativo e Financeiro",
      "description": "Artigos diversos...",
      "locale": "pt-br",
      "url": "https://totvscst.zendesk.com/api/v2/help_center/pt-br/categories/115001937708.json",
      "html_url": "https://totvscst.zendesk.com/hc/pt-br/categories/115001937708-..."
    }
  ],
  "secoes": [
    {
      "id": 360003678812,
      "name": "0 - Framework",
      "category_id": 360001488852,
      "parent_section_id": 1500000596481,
      "locale": "pt-br",
      "url": "https://totvsexterno.zendesk.com/api/v2/help_center/pt-br/sections/360003678812.json"
    }
  ],
  "playlists": [
    {
      "id": "PLD-htCoWcvYqLah55pDSonaq8gXFJdBh2",
      "title": "Apps de parceiros",
      "channelId": "UCfVwm8KPsx1Ryy256SM80fg",
      "itemCount": 12,
      "locale": "pt-br"
    }
  ],
  "produtos": [
    {
      "id": 268468234,
      "name": "Consultoria de Segmentos",
      "key": "ConSeg",
      "locale": "pt-br"
    }
  ],
  "lines": [
    { "id": "000052", "name": "Automação Fiscal" }
  ],
  "environments": [
    { "id": "04.2507", "name": "04.2507" }
  ],
  "versions": [
    { "id": "10.0.0", "name": "10.0.0" }
  ],
  "labels": [ ... ]  // Array of strings (332K+ items)
}
```

#### Filter Counts

| Filter | Count | Used in param |
|--------|-------|---------------|
| `categorias` | 65 | via section resolution |
| `secoes` | 3,297 | `sections[]` (integer ID) |
| `playlists` | 25 | `playlists[]` (string ID) |
| `produtos` | 64 | `produtos[]` (string key) |
| `labels` | 332,664 | `labels[]` (string) |
| `lines` | 34 | `lines[]` (string ID) |
| `environments` | 739 | `environments[]` (string ID) |
| `versions` | 341 | `versions[]` (string ID) |

---

### 2.3 `GET /api/v2/help_center/{locale}/articles/{id}.json` — Individual Article

**Standard Zendesk API**. The search results aggregate from multiple Zendesk subdomains. The article URL structure tells you which subdomain to use:

| Subdomain | Custom Domain |
|-----------|--------------|
| `totvscst.zendesk.com` | `totvscst.zendesk.com` |
| `totvsexterno.zendesk.com` | `centraldeatendimento.totvs.com` |
| `totvssuporte.zendesk.com` | `atendimento.totvs.com` |
| `cstadmgente.zendesk.com` | `centraldeatendimento2.totvs.com` |

**Method**: `GET`  
**Auth**: None (public articles)  
**Cache**: ETag supported

#### Response Schema

```json
{
  "article": {
    "id": 360028554332,
    "title": "Framework - Linha Protheus - Atalhos no Protheus 12",
    "body": "<p><strong>Dúvida</strong>...</p>",
    "html_url": "https://centraldeatendimento.totvs.com/hc/pt-br/articles/360028554332-...",
    "locale": "pt-br",
    "source_locale": "pt-br",
    "section_id": 360000075448,
    "author_id": 8467859487,
    "created_at": "2019-05-28T17:58:34Z",
    "updated_at": "2024-05-09T11:08:37Z",
    "edited_at": "2024-05-09T11:08:32Z",
    "label_names": ["protheus", "framework", "smartclient", "atalhos"],
    "draft": false,
    "promoted": false,
    "vote_sum": 0,
    "vote_count": 0,
    "outdated": false,
    "position": 0,
    "user_segment_id": null,
    "permission_group_id": 41427,
    "content_tag_ids": []
  }
}
```

**Important**: Articles may only be available in certain locales. The search engine indexes all locales, but a specific locale may return 404. Always fall back to the `source_locale` (usually `pt-br`).

---

### 2.4 `POST /cst/TEMPLATES` — HTML Templates

**Full URL**: `https://ti-services.totvs.com.br/cst/TEMPLATES`  
**Method**: `POST`  
**Body**: `{"template":"busca","lang":"en-us","token":null}`

Returns the search page HTML skeleton (with filter panel markup). Not needed for scraping.

---

### 2.5 `GET /cst/SESSION` — Session Management

**Full URL**: `https://ti-services.totvs.com.br/cst/SESSION?user=null&session=`  
**Method**: `GET`

Session initialization. Not needed for search — the search API works without a session.

---

### 2.6 `GET /cst/api_files/locales.json` — Available Locales

**Full URL**: `https://ti-services.totvs.com.br/cst/api_files/locales.json`  
**Method**: `GET`

```json
[
  { "id": 1,  "locale": "en-US", "name": "English" },
  { "id": 2,  "locale": "es",    "name": "Español" },
  { "id": 19, "locale": "pt-br", "name": "Português (Brasil)" },
  { "id": 1011, "locale": "pt",  "name": "Português (Portugal)" }
]
```

Use these values in the `langs[]` parameter.

---

### 2.7 `GET /cst/api_files/dynamic_content.json` — Dynamic Content

**Full URL**: `https://ti-services.totvs.com.br/cst/api_files/dynamic_content.json`  
**Method**: `GET`

Returns dynamic page content (not needed for scraping).

---

## 3. Pagination & Limits

| Parameter | Values | Default | Maximum per page |
|-----------|--------|---------|-----------------|
| `perPage` | `"10"`, `"20"`, `"50"`, `"100"` | `"10"` | 100 |
| `page` | integer (1-based) | `1` | ~3730 for "protheus", varies by query |

- **Total results**: given by `count` field in response
- **Total pages**: `pages` field = `ceil(count / perPage)`
- **Deep pagination**: Works up to page 1000+ (tested), but Elasticsearch may limit very deep pagination
- **Strategy for bulk extraction**: Use `perPage=100` and iterate pages 1..N with 1-second delays

---

## 4. Sort Modes

| Value | Behavior |
|-------|----------|
| `"score"` | Relevance (default). Secondary sort by date descending. |
| `"updated_at"` | Date, ascending (oldest first in observed results) |

The `sort` field in each hit is `[_score, timestamp_ms]`.

---

## 5. Rate Limiting & Security

- **No rate limiting detected**: 5 requests in rapid succession all returned HTTP 200 with consistent response times (3-8ms)
- **No authentication**: The API accepts anonymous requests
- **CORS open**: `access-control-allow-origin: *` (can call from browser)
- **Cloudflare**: Requests go through Cloudflare (`cf-ray` header) but no bot protection triggered on API requests
- **Cookies**: `_cfuvid` cookie is set by Cloudflare but not required. The browser gets `_help_center_session` from the Zendesk domain but the TOTVS API doesn't need it.

**Recommendation**: Add 500ms-1s delay between requests to be a good citizen, even though no rate limit was detected.

---

## 6. Autocomplete / Suggestions

**Not available**. The search page only triggers on explicit submit (click "Buscar" button). There is no typeahead/autocomplete endpoint. The `<div class="search-results-container">` in the template is for displaying results after search, not for live suggestions.

To implement your own autocomplete, you could call `/cst/BUSCA` with `perPage=3` and use the titles from results.

---

## 7. Full Article Content Retrieval

For Zendesk articles, use the standard Zendesk Help Center API:

```
GET https://centraldeatendimento.totvs.com/api/v2/help_center/{locale}/articles/{id}.json
```

Where:
- `{locale}` = `pt-br`, `en-us`, or `es`
- `{id}` = the numeric article ID (strip `z_` prefix from search `_id`)

The article `body` field contains HTML. For LLM consumption, strip HTML tags.

**Multi-subdomain strategy**: The search aggregates from multiple Zendesk instances. Parse the `html_url` to determine which subdomain to query for the full article:

```
totvscst.zendesk.com        → totvscst.zendesk.com
centraldeatendimento.totvs.com → totvsexterno.zendesk.com (API) / centraldeatendimento.totvs.com (web)
atendimento.totvs.com       → totvssuporte.zendesk.com (API)
centraldeatendimento2.totvs.com → cstadmgente.zendesk.com (API)
```

But in practice, trying `centraldeatendimento.totvs.com` as the API host works for most articles since they share the same Zendesk instance (`totvsexterno`).

---

## 8. Python Client

```python
"""
TOTVS CST Search API Client
Full-featured client for searching the TOTVS knowledge base.
Tested and verified 2026-04-27.
"""

import json
import time
import re
from typing import Optional
from urllib.request import Request, urlopen
from urllib.error import HTTPError


class TotvsSearchClient:
    """Client for the TOTVS CST search API (ti-services.totvs.com.br)."""

    BUSCA_URL = "https://ti-services.totvs.com.br/cst/BUSCA"
    FILTROS_URL = "https://ti-services.totvs.com.br/cst/BUSCA_FILTROS"
    ARTICLE_API_BASE = "https://centraldeatendimento.totvs.com/api/v2/help_center"

    def __init__(self, delay: float = 0.5):
        """
        Args:
            delay: Seconds to wait between API calls (default 0.5s).
        """
        self.delay = delay
        self._filtros_cache = None
        self._last_request = 0.0

    def _rate_limit(self):
        """Enforce delay between requests."""
        elapsed = time.time() - self._last_request
        if elapsed < self.delay:
            time.sleep(self.delay - elapsed)
        self._last_request = time.time()

    def _post(self, url: str, body: dict) -> dict:
        """POST JSON and return parsed response."""
        self._rate_limit()
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
        req = Request(
            url,
            data=data,
            headers={
                "Content-Type": "application/json",
                "Origin": "https://totvscst.zendesk.com",
                "Referer": "https://totvscst.zendesk.com/",
                "Accept": "application/json",
                "User-Agent": "TotvsSearchClient/1.0",
            },
        )
        with urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))

    def _get(self, url: str) -> dict:
        """GET JSON and return parsed response."""
        self._rate_limit()
        req = Request(
            url,
            headers={
                "Accept": "application/json",
                "Origin": "https://totvscst.zendesk.com",
                "User-Agent": "TotvsSearchClient/1.0",
            },
        )
        with urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))

    # ── Search ──────────────────────────────────────────────

    def search(
        self,
        query: str,
        *,
        page: int = 1,
        per_page: int = 10,
        types: Optional[list[str]] = None,
        langs: Optional[list[str]] = None,
        sort_order: str = "score",
        sections: Optional[list[int]] = None,
        produtos: Optional[list[str]] = None,
        labels: Optional[list[str]] = None,
        playlists: Optional[list[str]] = None,
        lines: Optional[list[str]] = None,
        environments: Optional[list[str]] = None,
        versions: Optional[list[str]] = None,
        date_start: Optional[str] = None,
        date_end: Optional[str] = None,
    ) -> dict:
        """
        Execute a search query.

        Args:
            query: Search text.
            page: Page number (1-based).
            per_page: Results per page (10, 20, 50, or 100).
            types: Content sources. Default ["zendesk", "central", "confluence", "youtube"].
            langs: Language codes. Default ["es", "pt-br", "en-us"].
            sort_order: "score" or "updated_at".
            sections: Section IDs to filter by.
            produtos: Product keys to filter by.
            labels: Label IDs to filter by.
            playlists: YouTube playlist IDs.
            lines: Download line IDs.
            environments: Environment IDs.
            versions: Version IDs.
            date_start: Filter by start date (YYYY-MM-DD).
            date_end: Filter by end date (YYYY-MM-DD).

        Returns:
            Raw API response dict with keys: pages, page, count, took, hits.
        """
        body = {
            "query": query,
            "assunto": query,
            "sort_order": sort_order,
            "perPage": str(per_page),
            "page": str(page),
            "types": types or ["zendesk", "central", "confluence", "youtube"],
            "langs": langs or ["es", "pt-br", "en-us"],
            "sections": sections or [],
            "produtos": produtos or [],
            "labels": labels or [],
            "playlists": playlists or [],
            "lines": lines or [],
            "environments": environments or [],
            "versions": versions or [],
            "highlightResults": True,
        }
        if date_start:
            body["datainicial"] = date_start
        if date_end:
            body["datafinal"] = date_end

        return self._post(self.BUSCA_URL, body)

    def search_all_pages(
        self,
        query: str,
        *,
        per_page: int = 100,
        max_pages: Optional[int] = None,
        **kwargs,
    ) -> list[dict]:
        """
        Fetch all pages of search results automatically.

        Args:
            query: Search text.
            per_page: Results per page (max 100).
            max_pages: Limit total pages (None = all).
            **kwargs: Passed through to search().

        Returns:
            List of raw hit dicts from all pages.
        """
        all_hits = []
        first = self.search(query, page=1, per_page=per_page, **kwargs)
        all_hits.extend(first["hits"])
        total_pages = first["pages"]

        end_page = min(total_pages, max_pages) if max_pages else total_pages

        for p in range(2, end_page + 1):
            page_data = self.search(query, page=p, per_page=per_page, **kwargs)
            all_hits.extend(page_data["hits"])

        return all_hits

    # ── Filter Metadata ─────────────────────────────────────

    def get_filters(self, force_refresh: bool = False) -> dict:
        """
        Get filter metadata (categories, sections, products, labels, etc.).

        Args:
            force_refresh: If True, bypass cache.

        Returns:
            Dict with keys: categorias, secoes, playlists, produtos, labels,
            lines, environments, versions.
        """
        if self._filtros_cache is not None and not force_refresh:
            return self._filtros_cache
        self._filtros_cache = self._get(self.FILTROS_URL)
        return self._filtros_cache

    def get_categories(self) -> list[dict]:
        """Return list of all categories."""
        return self.get_filters().get("categorias", [])

    def get_sections(self) -> list[dict]:
        """Return list of all sections."""
        return self.get_filters().get("secoes", [])

    def get_products(self) -> list[dict]:
        """Return list of all TDN products."""
        return self.get_filters().get("produtos", [])

    def find_sections_by_category(self, category_name: str) -> list[dict]:
        """Find sections belonging to a category by name."""
        cats = self.get_categories()
        cat_ids = [c["id"] for c in cats if c["name"] == category_name]
        if not cat_ids:
            return []
        target_id = cat_ids[0]
        return [s for s in self.get_sections() if s.get("category_id") == target_id]

    # ── Article Retrieval ───────────────────────────────────

    @staticmethod
    def parse_hit(hit: dict) -> dict:
        """
        Extract normalized fields from a search hit.

        Returns dict with: id, type, title, url, body, updated_at, score.
        """
        s = hit["_source"]
        id_raw = hit["_id"]
        search_type = s.get("search_type", "unknown")

        # Strip prefix to get real ID
        prefixes = {"z_": "zendesk", "cd_": "central", "c_": "confluence", "y_": "youtube"}
        real_id = id_raw
        for prefix in prefixes:
            if id_raw.startswith(prefix):
                real_id = id_raw[len(prefix):]
                break

        return {
            "id": real_id,
            "raw_id": id_raw,
            "type": search_type,
            "title": s.get("title", ""),
            "title_name": s.get("title_name", ""),
            "url": s.get("html_url", ""),
            "body": s.get("body", ""),
            "updated_at": s.get("updated_at", ""),
            "score": hit.get("_score", 0),
            "highlight": hit.get("highlight", {}),
            "thumbnails": s.get("thumbnails"),
            "hide_description": s.get("hideDescription", False),
        }

    def get_article(self, article_id: int, locale: str = "pt-br") -> Optional[dict]:
        """
        Fetch a full Zendesk article by ID using the standard Help Center API.

        Args:
            article_id: Zendesk article ID (numeric).
            locale: Article locale ("pt-br", "en-us", "es").

        Returns:
            Article dict or None if not found.
        """
        self._rate_limit()
        url = f"{self.ARTICLE_API_BASE}/{locale}/articles/{article_id}.json"
        req = Request(
            url,
            headers={
                "Accept": "application/json",
                "User-Agent": "TotvsSearchClient/1.0",
            },
        )
        try:
            with urlopen(req) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return data.get("article")
        except HTTPError as e:
            if e.code == 404:
                # Try source_locale fallback — pt-br is the source for most content
                if locale != "pt-br":
                    return self.get_article(article_id, "pt-br")
            return None

    def get_article_body_text(self, article_id: int, locale: str = "pt-br") -> Optional[str]:
        """
        Fetch article body as plain text (HTML tags stripped).

        Args:
            article_id: Zendesk article ID.
            locale: Article locale.

        Returns:
            Plain text body or None.
        """
        article = self.get_article(article_id, locale)
        if not article:
            return None
        # Simple HTML tag removal
        body = article.get("body", "")
        body = re.sub(r"<[^>]+>", " ", body)
        body = re.sub(r"\s+", " ", body).strip()
        return body

    # ── Utility ─────────────────────────────────────────────

    def extract_results(self, hits: list[dict]) -> list[dict]:
        """
        Extract and normalize a list of raw hits into clean result dicts.

        Args:
            hits: Raw hits from API response.

        Returns:
            List of normalized dicts (see parse_hit).
        """
        return [self.parse_hit(h) for h in hits]


# ── Convenience Functions ───────────────────────────────────

def search_totvs(query: str, max_results: int = 50, **kwargs) -> list[dict]:
    """
    Quick one-shot search. Returns normalized results.

    Args:
        query: Search text.
        max_results: Maximum results to return.
        **kwargs: Passed to client.search().

    Returns:
        List of normalized result dicts.
    """
    client = TotvsSearchClient(delay=0.3)
    per_page = min(max_results, 100)
    pages_needed = (max_results + per_page - 1) // per_page
    hits = client.search_all_pages(query, per_page=per_page, max_pages=pages_needed, **kwargs)
    return client.extract_results(hits[:max_results])


def get_article_text(article_id: int, locale: str = "pt-br") -> Optional[str]:
    """Quick one-shot article content fetch."""
    client = TotvsSearchClient()
    return client.get_article_body_text(article_id, locale)


# ── Example Usage ───────────────────────────────────────────

if __name__ == "__main__":
    client = TotvsSearchClient(delay=0.5)

    # 1. Basic search
    print("=== Basic search ===")
    results = search_totvs("protheus advpl", max_results=10)
    for r in results:
        print(f"  [{r['type']}] {r['title'][:80]}")
        print(f"    url: {r['url']}")

    # 2. Search with filters
    print("\n=== Filtered search (zendesk only, Spanish, sorted by date) ===")
    resp = client.search(
        "protheus",
        types=["zendesk"],
        langs=["es"],
        sort_order="updated_at",
        per_page=5,
    )
    for hit in resp["hits"]:
        item = client.parse_hit(hit)
        print(f"  [{item['updated_at']}] {item['title'][:80]}")

    # 3. Fetch full article body
    print("\n=== Full article ===")
    text = client.get_article_body_text(360028554332)
    if text:
        print(f"  {text[:300]}...")
        print(f"  Total length: {len(text)} chars")

    # 4. Explore filters
    print("\n=== Available categories ===")
    cats = client.get_categories()
    for c in cats[:5]:
        print(f"  [{c['locale']}] {c['name']} (id={c['id']})")

    # 5. Save results to JSON
    print("\n=== Saving results ===")
    all_hits = client.search_all_pages("advpl", per_page=100, max_pages=3)
    items = client.extract_results(all_hits)
    with open("search_results.json", "w", encoding="utf-8") as f:
        json.dump(items, f, ensure_ascii=False, indent=2)
    print(f"  Saved {len(items)} results to search_results.json")

    # 6. Aggregate stats
    types = {}
    for item in items:
        types[item["type"]] = types.get(item["type"], 0) + 1
    print(f"  Type distribution: {types}")
```

---

## 9. Edge Cases & Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| HTTP 400 "Bad Request" | JSON body malformed (encoding issues in shell) | Use file-based body: `curl -d @body.json` |
| HTTP 404 on article | Article not available in requested locale | Fall back to `pt-br` locale |
| HTML in body field | Normal — Zendesk stores articles as HTML | Strip tags with regex: `re.sub(r'<[^>]+>', ' ', body)` |
| Empty results | Query too specific or date filter excludes everything | Remove `date_start`/`date_end` or broaden query |
| Encoding issues (Ã© instead of é) | latin1 vs utf-8 mismatch | Ensure `response.read().decode('utf-8')` |
| Large filter response (7.5MB) | BUSCA_FILTROS is heavy | Cache it locally, use ETag/If-None-Match |
| Unknown `_id` prefix | Future content types may be added | Handle gracefully, extract everything after `_` as ID |

---

## 10. LLM Integration Patterns

### Building a Knowledge Base Index

```python
# 1. Get all available filters for context
filters = client.get_filters()

# 2. Search broadly
all_hits = client.search_all_pages("protheus", per_page=100, max_pages=5, langs=["pt-br"])
items = client.extract_results(all_hits)

# 3. For each article, fetch full body
for item in items:
    if item["type"] == "zendesk":
        art_id = int(item["id"])
        full_text = client.get_article_body_text(art_id)
        item["full_body"] = full_text

# 4. Save as JSONL for LLM ingestion
import json
with open("knowledge_base.jsonl", "w", encoding="utf-8") as f:
    for item in items:
        f.write(json.dumps(item, ensure_ascii=False) + "\n")
```

### Search-as-URL-resolver

```python
def resolve_urls_for_llm(queries: list[str]) -> dict[str, list[str]]:
    """Given a list of search queries, return relevant URLs for each."""
    client = TotvsSearchClient()
    result = {}
    for q in queries:
        hits = search_totvs(q, max_results=10, types=["zendesk"], langs=["pt-br","es"])
        result[q] = [h["url"] for h in hits]
    return result
```

---

## 11. Verification Log

All claims verified with actual HTTP requests:

| Test | Date | Result |
|------|------|--------|
| Basic search (POST BUSCA) | 2026-04-27 | HTTP 200, valid JSON |
| All 4 content types | 2026-04-27 | zendesk/central/confluence/youtube all return hits |
| Sort by updated_at | 2026-04-27 | Works, returns date-ordered results |
| Deep pagination (page=1000) | 2026-04-27 | HTTP 200, returns 3 hits |
| Max perPage (100) | 2026-04-27 | HTTP 200, returns 100 hits |
| Minimal body (no hostURL) | 2026-04-27 | HTTP 200, works with just query+perPage+types+langs |
| Date filtering | 2026-04-27 | `datainicial`/`datafinal` parameters work |
| Rate limit test (5 rapid requests) | 2026-04-27 | All HTTP 200, 3-8ms each |
| Article API (pt-br) | 2026-04-27 | HTTP 200, full article body |
| Article API (es - missing) | 2026-04-27 | HTTP 404, falls back to pt-br |
| BUSCA_FILTROS | 2026-04-27 | HTTP 200, 7.5MB, all filter types present |
| Templates endpoint | 2026-04-27 | HTTP 200, returns HTML skeleton |
| Locales endpoint | 2026-04-27 | HTTP 200, 4 locales |
