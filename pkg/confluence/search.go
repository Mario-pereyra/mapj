package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SearchOpts configures a CQL search against the Confluence REST API.
type SearchOpts struct {
	Query    string   // Full-text search term (uses siteSearch field — title+body+labels)
	Space    string   // Single space key filter
	Spaces   []string // Multiple space keys (OR)
	Label    string   // Single label filter
	Labels   []string // Multiple labels (AND)
	Type     string   // Content type: page, blogpost, attachment (default: page)
	Ancestor string   // Page ID: return all descendants of this page
	Since    string   // Date expression: "2024-01-01", "1w", "4d", "2m"
	Limit    int      // Max results to return (auto-paginated)
	Start    int      // Pagination offset
}

// SearchResult is the structured output of a CQL search.
type SearchResult struct {
	Results   []PageRef `json:"results"`
	Count     int       `json:"count"`
	Total     int       `json:"total"`
	Start     int       `json:"start"`
	HasNext   bool      `json:"hasNext"`
	NextStart int       `json:"nextStart,omitempty"`
	CQL       string    `json:"cql"`
}

// PageRef is a single search result item, optimized for token efficiency.
type PageRef struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url,omitempty"`
	ChildCount *int   `json:"childCount,omitempty"` // nil = not fetched, 0 = leaf, N = has children
	Type       string `json:"-"` // Internal use
}

// SpaceRef is the space-key+name pair included in each result.
type SpaceRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// GetChildCount returns the number of direct child pages for a given page ID.
func (c *Client) GetChildCount(ctx context.Context, pageID string) (int, error) {
	params := map[string]string{
		"limit":  "1", // We only need size, not actual children
		"expand": "",
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/content/%s/child/page", pageID), params)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Size int `json:"size"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to parse child count: %w", err)
	}
	return resp.Size, nil
}

// Search executes a CQL search against /rest/api/content/search with auto-pagination.
func (c *Client) Search(ctx context.Context, opts *SearchOpts) (*SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 25
	}
	if opts.Type == "" {
		opts.Type = "page"
	}

	cql := buildCQL(opts)
	
	var allRefs []PageRef
	currentStart := opts.Start
	remainingLimit := opts.Limit
	
	// API typically allows up to 100 per request
	batchSize := 100
	if remainingLimit < 100 {
		batchSize = remainingLimit
	}
	
	hasNext := false
	nextStart := 0
	total := 0

	for remainingLimit > 0 {
		batchSize = 100
		if remainingLimit < 100 {
			batchSize = remainingLimit
		}

		params := map[string]string{
			"cql":    cql,
			"limit":  fmt.Sprintf("%d", batchSize),
			"start":  fmt.Sprintf("%d", currentStart),
			// Minimal expand to reduce payload size
			"expand": "space", 
		}

		body, err := c.do(ctx, "GET", "/rest/api/content/search", params)
		if err != nil {
			return nil, err
		}

		var raw struct {
			Results []struct {
				ID    string `json:"id"`
				Type  string `json:"type"`
				Title string `json:"title"`
				Space struct {
					Key  string `json:"key"`
				} `json:"space"`
				Links struct {
					WebUI  string `json:"webui"`
				} `json:"_links"`
			} `json:"results"`
			Size  int `json:"size"`
			Start int `json:"start"`
			Links struct {
				Next string `json:"next"`
				Base string `json:"base"`
			} `json:"_links"`
		}

		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse search response: %w", err)
		}

		baseURL := raw.Links.Base
		if baseURL == "" {
			baseURL = c.BaseURL
		}

		for _, r := range raw.Results {
			ref := PageRef{
				ID:    r.ID,
				Title: r.Title,
				Type:  r.Type,
			}

			// Build canonical URL
			if r.Links.WebUI != "" {
				ref.URL = baseURL + r.Links.WebUI
			} else if r.Space.Key != "" && r.ID != "" {
				ref.URL = fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", baseURL, r.ID)
			}
			
			allRefs = append(allRefs, ref)
		}
		
		total += raw.Size
		remainingLimit -= raw.Size
		currentStart += raw.Size
		
		if raw.Links.Next == "" || raw.Size == 0 {
			hasNext = false
			break
		} else {
			hasNext = true
			nextStart = currentStart
		}
	}

	return &SearchResult{
		Results:   allRefs,
		Count:     total,
		Total:     total, 
		Start:     opts.Start,
		HasNext:   hasNext,
		NextStart: nextStart,
		CQL:       cql,
	}, nil
}

// buildCQL constructs a CQL query string from SearchOpts.
func buildCQL(opts *SearchOpts) string {
	var clauses []string

	// Text search — use siteSearch (TOTVS custom field: title+body+labels)
	if opts.Query != "" {
		clauses = append(clauses, fmt.Sprintf(`siteSearch ~ "%s"`, opts.Query))
	}

	// Space filter — single or multi-space
	switch {
	case opts.Space != "":
		clauses = append(clauses, fmt.Sprintf(`space = "%s"`, opts.Space))
	case len(opts.Spaces) == 1:
		clauses = append(clauses, fmt.Sprintf(`space = "%s"`, opts.Spaces[0]))
	case len(opts.Spaces) > 1:
		keys := make([]string, len(opts.Spaces))
		for i, k := range opts.Spaces {
			keys[i] = fmt.Sprintf(`"%s"`, k)
		}
		clauses = append(clauses, fmt.Sprintf(`space IN (%s)`, strings.Join(keys, ", ")))
	}

	// Label filters — each label is an AND condition
	if opts.Label != "" {
		clauses = append(clauses, fmt.Sprintf(`label = "%s"`, opts.Label))
	}
	for _, l := range opts.Labels {
		clauses = append(clauses, fmt.Sprintf(`label = "%s"`, l))
	}

	// Ancestor filter (returns all descendants of given page ID)
	if opts.Ancestor != "" {
		clauses = append(clauses, fmt.Sprintf(`ancestor = %s`, opts.Ancestor))
	}

	// Content type
	if opts.Type != "" {
		clauses = append(clauses, fmt.Sprintf(`type = %s`, opts.Type))
	}

	// Date filter — supports relative ("1w", "4d", "2m") and absolute ("2024-01-01")
	if opts.Since != "" {
		dateExpr := parseSinceExpr(opts.Since)
		clauses = append(clauses, fmt.Sprintf(`lastmodified >= %s`, dateExpr))
	}

	return strings.Join(clauses, " AND ")
}

// parseSinceExpr converts user-friendly date expressions to CQL date expressions.
// Supports: "1w" (1 week), "4d" (4 days), "2m" (2 months), "1y" (1 year), or ISO date "2024-01-01".
func parseSinceExpr(since string) string {
	if len(since) >= 2 {
		last := since[len(since)-1]
		num := since[:len(since)-1]
		switch last {
		case 'w', 'W':
			return fmt.Sprintf(`now("-%sw")`, num)
		case 'd', 'D':
			return fmt.Sprintf(`now("-%sd")`, num)
		case 'm', 'M':
			return fmt.Sprintf(`now("-%sm")`, num)
		case 'y', 'Y':
			return fmt.Sprintf(`now("-%sy")`, num)
		}
	}
	// Assume ISO date format "2024-01-01"
	return fmt.Sprintf(`"%s"`, since)
}
