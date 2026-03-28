package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
	Limit    int      // Max results per request (default: 25)
	Start    int      // Pagination offset
}

// SearchResult is the structured output of a CQL search.
type SearchResult struct {
	Results  []PageRef `json:"results"`
	Count    int       `json:"count"`
	Total    int       `json:"total"`
	Start    int       `json:"start"`
	HasNext  bool      `json:"hasNext"`
	NextStart int      `json:"nextStart,omitempty"`
	CQL      string    `json:"cql"`
}

// PageRef is a single search result item.
type PageRef struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	URL         string    `json:"url,omitempty"`
	Space       SpaceRef  `json:"space"`
	Labels      []string  `json:"labels,omitempty"`
	Excerpt     string    `json:"excerpt,omitempty"`
	Ancestors   []AncestorRef `json:"ancestors,omitempty"`
	Version     int       `json:"version,omitempty"`
	LastUpdated *time.Time `json:"lastUpdated,omitempty"`
	LastUpdatedBy string  `json:"lastUpdatedBy,omitempty"`
}

// SpaceRef is the space-key+name pair included in each result.
type SpaceRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// AncestorRef is defined in pages.go — breadcrumb item (parent page chain).

// Search executes a CQL search against /rest/api/content/search.
//
// The siteSearch field is a TOTVS-custom Confluence field that searches across
// title, body, and labels simultaneously. It is more powerful than the standard
// `text` field for TDN searches.
func (c *Client) Search(ctx context.Context, opts *SearchOpts) (*SearchResult, error) {
	if opts.Limit == 0 {
		opts.Limit = 25
	}
	if opts.Type == "" {
		opts.Type = "page"
	}

	cql := buildCQL(opts)

	params := map[string]string{
		"cql":    cql,
		"limit":  fmt.Sprintf("%d", opts.Limit),
		"start":  fmt.Sprintf("%d", opts.Start),
		"expand": "space,metadata.labels,version,history.lastUpdated,ancestors",
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
				Name string `json:"name"`
			} `json:"space"`
			Version struct {
				Number int `json:"number"`
			} `json:"version"`
			History struct {
				LastUpdated struct {
					When string `json:"when"`
					By   struct {
						DisplayName string `json:"displayName"`
					} `json:"by"`
				} `json:"lastUpdated"`
			} `json:"history"`
			Metadata struct {
				Labels struct {
					Results []struct {
						Name string `json:"name"`
					} `json:"results"`
				} `json:"labels"`
			} `json:"metadata"`
			Ancestors []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"ancestors"`
			Links struct {
				WebUI  string `json:"webui"`
				TinyUI string `json:"tinyui"`
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

	refs := make([]PageRef, 0, len(raw.Results))
	for _, r := range raw.Results {
		ref := PageRef{
			ID:      r.ID,
			Type:    r.Type,
			Title:   r.Title,
			Space:   SpaceRef{Key: r.Space.Key, Name: r.Space.Name},
			Version: r.Version.Number,
		}

		// Build canonical URL from webui link
		if r.Links.WebUI != "" {
			ref.URL = baseURL + r.Links.WebUI
		} else if r.Space.Key != "" && r.ID != "" {
			ref.URL = fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", baseURL, r.ID)
		}

		// Extract labels
		for _, l := range r.Metadata.Labels.Results {
			ref.Labels = append(ref.Labels, l.Name)
		}

		// Parse last updated time
		if r.History.LastUpdated.When != "" {
			t, err := time.Parse(time.RFC3339, r.History.LastUpdated.When)
			if err == nil {
				ref.LastUpdated = &t
			}
			ref.LastUpdatedBy = r.History.LastUpdated.By.DisplayName
		}

		// Build ancestors breadcrumb
		for _, a := range r.Ancestors {
			ref.Ancestors = append(ref.Ancestors, AncestorRef{
				ID:    a.ID,
				Title: a.Title,
			})
		}

		refs = append(refs, ref)
	}

	// Determine next page
	hasNext := raw.Links.Next != ""
	nextStart := opts.Start + raw.Size

	return &SearchResult{
		Results:   refs,
		Count:     raw.Size,
		Total:     raw.Size, // REST API /content/search doesn't return total easily — use count
		Start:     raw.Start,
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
