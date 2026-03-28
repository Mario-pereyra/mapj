package confluence

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetDescendants returns all descendant page IDs using CQL with pagination.
func (c *Client) GetDescendants(ctx context.Context, pageID string) ([]string, error) {
	var allIDs []string
	start := 0
	limit := 100

	for {
		params := map[string]string{
			"cql":   fmt.Sprintf("type=page AND ancestor=%s", pageID),
			"limit": fmt.Sprintf("%d", limit),
			"start": fmt.Sprintf("%d", start),
		}

		body, err := c.do(ctx, "GET", "/rest/api/content/search", params)
		if err != nil {
			return allIDs, fmt.Errorf("failed to fetch descendants for page %s: %w", pageID, err)
		}

		var resp struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
			Size  int `json:"size"`
			Links struct {
				Next string `json:"next,omitempty"`
			} `json:"_links"`
		}

		if err := json.Unmarshal(body, &resp); err != nil {
			return allIDs, fmt.Errorf("failed to parse descendants: %w", err)
		}

		for _, r := range resp.Results {
			allIDs = append(allIDs, r.ID)
		}

		if resp.Links.Next == "" || resp.Size < limit {
			break
		}
		start += resp.Size
	}

	return allIDs, nil
}
