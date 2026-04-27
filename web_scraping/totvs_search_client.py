"""
TOTVS CST Search API Client
============================
Standalone client for searching the TOTVS knowledge base aggregator.
Backend: https://ti-services.totvs.com.br/cst/BUSCA

Usage:
    python totvs_search_client.py                     # Run examples
    python -c "from totvs_search_client import search_totvs; print(search_totvs('protheus', max_results=5))"
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
        self.delay = delay
        self._filtros_cache = None
        self._last_request = 0.0

    def _rate_limit(self):
        elapsed = time.time() - self._last_request
        if elapsed < self.delay:
            time.sleep(self.delay - elapsed)
        self._last_request = time.time()

    def _post(self, url: str, body: dict) -> dict:
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
        if self._filtros_cache is not None and not force_refresh:
            return self._filtros_cache
        self._filtros_cache = self._get(self.FILTROS_URL)
        return self._filtros_cache

    def get_categories(self) -> list[dict]:
        return self.get_filters().get("categorias", [])

    def get_sections(self) -> list[dict]:
        return self.get_filters().get("secoes", [])

    def get_products(self) -> list[dict]:
        return self.get_filters().get("produtos", [])

    def find_sections_by_category(self, category_name: str) -> list[dict]:
        cats = self.get_categories()
        cat_ids = [c["id"] for c in cats if c["name"] == category_name]
        if not cat_ids:
            return []
        return [s for s in self.get_sections() if s.get("category_id") == cat_ids[0]]

    # ── Article Retrieval ───────────────────────────────────

    @staticmethod
    def parse_hit(hit: dict) -> dict:
        s = hit["_source"]
        id_raw = hit["_id"]
        search_type = s.get("search_type", "unknown")
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
        self._rate_limit()
        url = f"{self.ARTICLE_API_BASE}/{locale}/articles/{article_id}.json"
        req = Request(url, headers={"Accept": "application/json", "User-Agent": "TotvsSearchClient/1.0"})
        try:
            with urlopen(req) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return data.get("article")
        except HTTPError as e:
            if e.code == 404 and locale != "pt-br":
                return self.get_article(article_id, "pt-br")
            return None

    def get_article_body_text(self, article_id: int, locale: str = "pt-br") -> Optional[str]:
        article = self.get_article(article_id, locale)
        if not article:
            return None
        body = article.get("body", "")
        body = re.sub(r"<[^>]+>", " ", body)
        body = re.sub(r"\s+", " ", body).strip()
        return body

    def extract_results(self, hits: list[dict]) -> list[dict]:
        return [self.parse_hit(h) for h in hits]


# ── Module-level convenience ────────────────────────────────

def search_totvs(query: str, max_results: int = 50, **kwargs) -> list[dict]:
    """Quick one-shot search returning normalized results."""
    client = TotvsSearchClient(delay=0.3)
    per_page = min(max_results, 100)
    pages_needed = (max_results + per_page - 1) // per_page
    hits = client.search_all_pages(query, per_page=per_page, max_pages=pages_needed, **kwargs)
    return client.extract_results(hits[:max_results])


def get_article_text(article_id: int, locale: str = "pt-br") -> Optional[str]:
    client = TotvsSearchClient()
    return client.get_article_body_text(article_id, locale)


# ── CLI ─────────────────────────────────────────────────────

if __name__ == "__main__":
    import sys

    if len(sys.argv) > 1:
        query = " ".join(sys.argv[1:])
        print(f"Searching for: {query}\n")
        results = search_totvs(query, max_results=10, langs=["es", "pt-br"])
        for i, r in enumerate(results, 1):
            print(f"{i}. [{r['type'].upper()}] {r['title'][:100]}")
            print(f"   URL: {r['url']}")
            print(f"   Updated: {r['updated_at']}")
            print()
    else:
        print("=" * 60)
        print("TOTVS CST Search API Client - Usage Examples")
        print("=" * 60)

        client = TotvsSearchClient(delay=0.5)

        # Example 1: Basic search
        print("\n1. Basic search: 'protheus advpl'")
        results = search_totvs("protheus advpl", max_results=5, langs=["es"])
        for r in results:
            print(f"   [{r['type']}] {r['title'][:80]}")

        # Example 2: Date-filtered
        print("\n2. Date-filtered search: 'protheus' (2024-2025)")
        resp = client.search(
            "protheus", types=["zendesk"], langs=["es"],
            date_start="2024-01-01", date_end="2025-12-31", per_page=3,
        )
        for hit in resp["hits"]:
            item = client.parse_hit(hit)
            print(f"   [{item['updated_at']}] {item['title'][:80]}")

        # Example 3: Full article content
        print("\n3. Full article body (id=360028554332)")
        text = client.get_article_body_text(360028554332)
        if text:
            print(f"   Preview: {text[:200]}...")
            print(f"   Length: {len(text)} chars")

        # Example 4: Available filters
        print("\n4. Available categories (first 5):")
        for c in client.get_categories()[:5]:
            print(f"   [{c['locale']}] {c['name']}")

        # Example 5: Save to JSON
        print("\n5. Saving results to JSON...")
        hits = client.search_all_pages("advpl", per_page=100, max_pages=2, langs=["es"])
        items = client.extract_results(hits)
        output_path = "web_scraping/search_results.json"
        with open(output_path, "w", encoding="utf-8") as f:
            json.dump(items, f, ensure_ascii=False, indent=2)
        print(f"   Saved {len(items)} results to {output_path}")

        # Type distribution
        type_counts = {}
        for item in items:
            type_counts[item["type"]] = type_counts.get(item["type"], 0) + 1
        print(f"   Type distribution: {type_counts}")
