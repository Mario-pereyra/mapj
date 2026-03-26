package confluence

import (
	"context"
	"encoding/json"
	"fmt"
)

type Page struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Body    *Body     `json:"body,omitempty"`
	Space   SpaceRef  `json:"space"`
	Version *Version  `json:"version,omitempty"`
	Links   PageLinks `json:"_links"`
}

type Body struct {
	Storage    *StorageValue `json:"storage,omitempty"`
	ExportView *StorageValue `json:"export_view,omitempty"`
}

type StorageValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type VersionBy struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type Version struct {
	Number int        `json:"number"`
	When   string     `json:"when,omitempty"`
	By     *VersionBy `json:"by,omitempty"`
}

type PageLinks struct {
	WebUI string `json:"webui,omitempty"`
	Edit  string `json:"edit,omitempty"`
	Tiny  string `json:"tinyui,omitempty"`
}

func (c *Client) GetPage(ctx context.Context, pageID string, expand string) (*Page, error) {
	params := map[string]string{}
	if expand != "" {
		params["expand"] = expand
	} else {
		params["expand"] = "body.storage.value,version,space"
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/content/%s", pageID), params)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page: %w", err)
	}

	return &page, nil
}

func (c *Client) GetPageChildren(ctx context.Context, pageID string) ([]PageRef, error) {
	params := map[string]string{
		"limit":  "100",
		"expand": "space,metadata.labels",
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/content/%s/child", pageID), params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []PageRef `json:"results"`
		Size    int       `json:"size"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse children: %w", err)
	}

	return resp.Results, nil
}
