package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// ParseResult holds the result of parsing a Confluence URL.
type ParseResult struct {
	BaseURL     string // Base URL of the Confluence instance (e.g., https://tdninterno.totvs.com)
	PageID      string // Numeric page ID if found
	SpaceKey    string // Space key if found in URL
	Title       string // Page title if found in URL (URL-decoded)
	OriginalURL string // The original input URL for fallback scraping
}

// ParseConfluenceInput parses a Confluence URL, page ID, or display URL into structured parts.
// Supports:
//   - Plain numeric ID: "12345"
//   - Cloud: https://x.atlassian.net/wiki/spaces/TEAM/pages/12345/Page+Title
//   - Server Display: https://tdn.totvs.com/display/tec/Page+Title
//   - Server ViewPage: https://tdn.totvs.com/pages/viewpage.action?pageId=12345
//   - Server ReleaseView: https://tdn.totvs.com/pages/releaseview.action?pageId=12345
func ParseConfluenceInput(input string) (*ParseResult, error) {
	input = strings.TrimSpace(input)

	// 1. Plain numeric ID
	if matched, _ := regexp.MatchString(`^\d+$`, input); matched {
		return &ParseResult{PageID: input}, nil
	}

	// Parse as URL
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	result := &ParseResult{OriginalURL: input}

	// Extract base URL (scheme + host)
	if u.Scheme != "" && u.Host != "" {
		result.BaseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}

	// 2. Check query params for pageId (viewpage.action, releaseview.action, etc.)
	queryParams := u.Query()
	if pageID := queryParams.Get("pageId"); pageID != "" {
		if matched, _ := regexp.MatchString(`^\d+$`, pageID); matched {
			result.PageID = pageID
			return result, nil
		}
	}

	pathParts := splitPath(u.Path)

	// 3. Cloud format: /wiki/spaces/KEY/pages/ID/optional-slug
	if idx := indexOf(pathParts, "pages"); idx >= 0 {
		if idx+1 < len(pathParts) {
			candidate := pathParts[idx+1]
			if matched, _ := regexp.MatchString(`^\d+$`, candidate); matched {
				result.PageID = candidate
			}
		}
		// Extract space key from /spaces/KEY/ if present
		if spIdx := indexOf(pathParts, "spaces"); spIdx >= 0 && spIdx+1 < len(pathParts) {
			result.SpaceKey = pathParts[spIdx+1]
		}
		if result.PageID != "" {
			return result, nil
		}
	}

	// 4. Server display format: /display/SPACEKEY/Page+Title
	//    or /display/public/SPACEKEY/Page+Title (common in TOTVS TDN)
	if idx := indexOf(pathParts, "display"); idx >= 0 {
		remaining := pathParts[idx+1:]
		// Skip 'public' prefix if present (Confluence Server convention)
		if len(remaining) > 0 && strings.EqualFold(remaining[0], "public") {
			remaining = remaining[1:]
		}
		if len(remaining) >= 2 {
			result.SpaceKey = remaining[0]
			result.Title = decodeURLComponent(remaining[1])
			return result, nil
		}
		if len(remaining) == 1 {
			result.SpaceKey = remaining[0]
			return result, nil
		}
	}

	// 5. Fallback: last two path segments as space/title (Confluence Server pattern)
	if len(pathParts) >= 2 {
		result.SpaceKey = decodeURLComponent(pathParts[len(pathParts)-2])
		result.Title = decodeURLComponent(pathParts[len(pathParts)-1])
		return result, nil
	}

	return nil, fmt.Errorf("could not parse Confluence URL: %s", input)
}

// ResolvePageID resolves a ParseResult to a page ID using the API if needed.
func (c *Client) ResolvePageID(ctx context.Context, pr *ParseResult) (string, error) {
	// If we already have a page ID, just return it
	if pr.PageID != "" {
		return pr.PageID, nil
	}

	// If we have space + title, look up the page by title first
	if pr.SpaceKey != "" && pr.Title != "" {
		page, err := c.GetPageByTitle(ctx, pr.SpaceKey, pr.Title)
		if err == nil && page.ID != "" {
			return page.ID, nil
		}

		// Fallback: try CQL search
		pageID, err := c.resolveBySearch(ctx, pr)
		if err == nil && pageID != "" {
			return pageID, nil
		}

		// Emergency fallback: scrape the page HTML for ajs-page-id (WAF bypass)
		scrapeURL := pr.OriginalURL
		if scrapeURL == "" && pr.BaseURL != "" {
			scrapeURL = fmt.Sprintf("%s/display/%s/%s", pr.BaseURL, pr.SpaceKey, url.PathEscape(pr.Title))
		}
		if scrapeURL != "" {
			if scraped, err := c.scrapePageID(ctx, scrapeURL); err == nil {
				return scraped, nil
			}
		}

		return "", fmt.Errorf("cannot resolve page: title=%q space=%q", pr.Title, pr.SpaceKey)
	}

	return "", fmt.Errorf("cannot resolve page: need either pageId or space+title")
}

// resolveBySearch uses CQL to find a page by title.
func (c *Client) resolveBySearch(ctx context.Context, pr *ParseResult) (string, error) {
	cql := fmt.Sprintf(`title="%s"`, pr.Title)
	if pr.SpaceKey != "" {
		cql += fmt.Sprintf(` AND space="%s"`, pr.SpaceKey)
	}

	params := map[string]string{
		"cql":   cql,
		"limit": "1",
	}

	body, err := c.do(ctx, "GET", "/rest/api/content/search", params)
	if err != nil {
		return "", fmt.Errorf("CQL search failed: %w", err)
	}

	var resp struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	if len(resp.Results) == 0 {
		return "", fmt.Errorf("page not found: title=%q space=%q", pr.Title, pr.SpaceKey)
	}

	return resp.Results[0].ID, nil
}

// scrapePageID extracts the page ID from the rendered HTML page.
// This is the emergency fallback when API calls fail (e.g., WAF blocking).
// Confluence Server embeds  <meta name="ajs-page-id" content="12345"> in the page HTML.
func (c *Client) scrapePageID(ctx context.Context, pageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create scrape request: %w", err)
	}

	// Use browser User-Agent to bypass WAF
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	// Apply auth
	if c.basicUser != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	} else if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("scrape request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("scrape returned status %d", resp.StatusCode)
	}

	// Read first 100KB (we only need the HEAD)
	limited := io.LimitReader(resp.Body, 100*1024)
	htmlBytes, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("failed to read scraped HTML: %w", err)
	}

	// Extract ajs-page-id from meta tag
	re := regexp.MustCompile(`name=["']ajs-page-id["']\s+content=["'](\d+)["']`)
	if m := re.FindSubmatch(htmlBytes); len(m) > 1 {
		return string(m[1]), nil
	}

	return "", fmt.Errorf("ajs-page-id not found in page HTML")
}

// splitPath splits a URL path into non-empty segments.
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// indexOf returns the index of target in parts, or -1 if not found.
func indexOf(parts []string, target string) int {
	for i, p := range parts {
		if strings.EqualFold(p, target) {
			return i
		}
	}
	return -1
}

// decodeURLComponent decodes a URL-encoded path component.
func decodeURLComponent(s string) string {
	// Try QueryUnescape first (handles + as space, common in /display/ URLs)
	decoded, err := url.QueryUnescape(s)
	if err == nil {
		return decoded
	}
	// Fallback to PathUnescape
	decoded, err = url.PathUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}
