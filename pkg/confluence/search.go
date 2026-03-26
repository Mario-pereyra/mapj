package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type SearchOpts struct {
	Query string
	Space string
	Label string
	Type  string
	Limit int
	Start int
}

type SearchResult struct {
	Results []PageRef `json:"results"`
	Count   int       `json:"count"`
	Total   int       `json:"total"`
}

type PageRef struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	URL        string   `json:"url,omitempty"`
	DisplayURL string   `json:"displayUrl,omitempty"`
	Space      SpaceRef `json:"space"`
	Labels     []string `json:"labels,omitempty"`
	Excerpt    string   `json:"excerpt,omitempty"`
}

type SpaceRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (c *Client) Search(ctx context.Context, opts *SearchOpts) (*SearchResult, error) {
	if opts.Limit == 0 {
		opts.Limit = 25
	}
	if opts.Type == "" {
		opts.Type = "page"
	}

	var clauses []string
	if opts.Query != "" {
		clauses = append(clauses, fmt.Sprintf(`text ~ "%s"`, opts.Query))
	}
	if opts.Space != "" {
		clauses = append(clauses, fmt.Sprintf(`space = "%s"`, opts.Space))
	}
	if opts.Label != "" {
		clauses = append(clauses, fmt.Sprintf(`label = "%s"`, opts.Label))
	}
	if opts.Type != "" {
		clauses = append(clauses, fmt.Sprintf(`type = "%s"`, opts.Type))
	}

	cql := strings.Join(clauses, " AND ")

	params := map[string]string{
		"cql":    cql,
		"limit":  fmt.Sprintf("%d", opts.Limit),
		"start":  fmt.Sprintf("%d", opts.Start),
		"expand": "space,metadata.labels,excerpt",
	}

	body, err := c.do(ctx, "GET", "/rest/api/content/search", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []PageRef `json:"results"`
		Size    int       `json:"size"`
		Total   int       `json:"total"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for i := range resp.Results {
		if resp.Results[i].Space.Key != "" {
			resp.Results[i].URL = fmt.Sprintf("%s/wiki/spaces/%s/pages/%s", c.BaseURL, resp.Results[i].Space.Key, resp.Results[i].ID)
		}
	}

	return &SearchResult{
		Results: resp.Results,
		Count:   resp.Size,
		Total:   resp.Total,
	}, nil
}
