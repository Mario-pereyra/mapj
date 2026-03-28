package confluence

import (
	"context"
	"encoding/json"
	"fmt"
)

// SpaceDetail holds full space information.
type SpaceDetail struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	HomepageID  string `json:"homepage_id,omitempty"`
}

// GetSpace retrieves a single space by key.
func (c *Client) GetSpace(ctx context.Context, spaceKey string) (*SpaceDetail, error) {
	params := map[string]string{
		"expand": "homepage",
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/space/%s", spaceKey), params)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description struct {
			Plain struct {
				Value string `json:"value"`
			} `json:"plain"`
		} `json:"description"`
		Homepage struct {
			ID string `json:"id"`
		} `json:"homepage"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse space: %w", err)
	}

	return &SpaceDetail{
		Key:         raw.Key,
		Name:        raw.Name,
		Type:        raw.Type,
		Description: raw.Description.Plain.Value,
		HomepageID:  raw.Homepage.ID,
	}, nil
}

// GetAllSpaces retrieves all global, current spaces.
func (c *Client) GetAllSpaces(ctx context.Context) ([]SpaceDetail, error) {
	var allSpaces []SpaceDetail
	start := 0
	limit := 100

	for {
		params := map[string]string{
			"type":   "global",
			"status": "current",
			"expand": "homepage",
			"limit":  fmt.Sprintf("%d", limit),
			"start":  fmt.Sprintf("%d", start),
		}

		body, err := c.do(ctx, "GET", "/rest/api/space", params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Results []struct {
				Key      string `json:"key"`
				Name     string `json:"name"`
				Type     string `json:"type"`
				Homepage struct {
					ID string `json:"id"`
				} `json:"homepage"`
			} `json:"results"`
			Size int `json:"size"`
		}

		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse spaces: %w", err)
		}

		for _, s := range resp.Results {
			allSpaces = append(allSpaces, SpaceDetail{
				Key:        s.Key,
				Name:       s.Name,
				Type:       s.Type,
				HomepageID: s.Homepage.ID,
			})
		}

		if resp.Size < limit {
			break
		}
		start += resp.Size
	}

	return allSpaces, nil
}

// GetSpacePageIDs returns all page IDs in a space by fetching the homepage and its descendants.
func (c *Client) GetSpacePageIDs(ctx context.Context, spaceKey string) ([]string, error) {
	space, err := c.GetSpace(ctx, spaceKey)
	if err != nil {
		return nil, err
	}

	if space.HomepageID == "" {
		return nil, fmt.Errorf("space %q has no homepage", spaceKey)
	}

	descendants, err := c.GetDescendants(ctx, space.HomepageID)
	if err != nil {
		return nil, err
	}

	// Prepend homepage ID
	return append([]string{space.HomepageID}, descendants...), nil
}
