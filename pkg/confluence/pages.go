package confluence

import (
	"context"
	"encoding/json"
	"fmt"
)

// Page represents a Confluence page with all body representations.
type Page struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Body    *Body     `json:"body,omitempty"`
	Space   SpaceRef  `json:"space"`
	Version *Version  `json:"version,omitempty"`
	Links   PageLinks `json:"_links"`

	// Hierarchy
	Ancestors  []AncestorRef `json:"ancestors,omitempty"`
	Expandable *Expandable   `json:"_expandable,omitempty"`

	// Metadata
	Metadata *PageMetadata `json:"metadata,omitempty"`
}

// Body contains the different body representations from Confluence.
type Body struct {
	Storage    *StorageValue `json:"storage,omitempty"`
	View       *StorageValue `json:"view,omitempty"`
	ExportView *StorageValue `json:"export_view,omitempty"`
	Editor2    *StorageValue `json:"editor2,omitempty"`
}

// StorageValue holds a body representation value.
type StorageValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// AncestorRef is a minimal page reference used in ancestry chains.
type AncestorRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"`
}

// Expandable holds expandable link paths.
type Expandable struct {
	Space string `json:"space,omitempty"`
}

// PageMetadata holds page metadata like labels.
type PageMetadata struct {
	Labels *LabelResults `json:"labels,omitempty"`
}

// LabelResults wraps a list of labels.
type LabelResults struct {
	Results []Label `json:"results"`
}

// Label represents a Confluence label.
type Label struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

// VersionBy identifies who made a version change.
type VersionBy struct {
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// Version holds page version info.
type Version struct {
	Number       int        `json:"number"`
	When         string     `json:"when,omitempty"`
	FriendlyWhen string     `json:"friendlyWhen,omitempty"`
	By           *VersionBy `json:"by,omitempty"`
}

// PageLinks holds navigation links for a page.
type PageLinks struct {
	WebUI string `json:"webui,omitempty"`
	Edit  string `json:"edit,omitempty"`
	Tiny  string `json:"tinyui,omitempty"`
	Base  string `json:"base,omitempty"`
}

// GetPage fetches a single page with all body representations needed for export.
func (c *Client) GetPage(ctx context.Context, pageID string, expand string) (*Page, error) {
	params := map[string]string{}
	if expand != "" {
		params["expand"] = expand
	} else {
		// Default: fetch all representations needed for high-quality export
		params["expand"] = "body.export_view,body.view,body.storage,body.editor2,version,space,ancestors,metadata.labels"
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

// GetPageByTitle fetches a page by space key and title.
func (c *Client) GetPageByTitle(ctx context.Context, spaceKey, title string) (*Page, error) {
	params := map[string]string{
		"title":  title,
		"expand": "version",
		"type":   "page",
	}

	if spaceKey != "" {
		params["spaceKey"] = spaceKey
	}

	body, err := c.do(ctx, "GET", "/rest/api/content", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []Page `json:"results"`
		Size    int    `json:"size"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse page results: %w", err)
	}

	if resp.Size == 0 {
		return nil, fmt.Errorf("page not found: title=%q space=%q", title, spaceKey)
	}

	return &resp.Results[0], nil
}

// GetPageChildren retrieves the immediate children of a page.
func (c *Client) GetPageChildren(ctx context.Context, pageID string) ([]AncestorRef, error) {
	params := map[string]string{
		"limit":  "100",
		"expand": "space",
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/rest/api/content/%s/child/page", pageID), params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []AncestorRef `json:"results"`
		Size    int           `json:"size"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse children: %w", err)
	}

	return resp.Results, nil
}

// GetExportViewHTML returns the best available HTML for markdown conversion.
// Priority: export_view > view > storage.
func (p *Page) GetExportViewHTML() string {
	if p.Body == nil {
		return ""
	}
	if p.Body.ExportView != nil && p.Body.ExportView.Value != "" {
		return p.Body.ExportView.Value
	}
	if p.Body.View != nil && p.Body.View.Value != "" {
		return p.Body.View.Value
	}
	if p.Body.Storage != nil && p.Body.Storage.Value != "" {
		return p.Body.Storage.Value
	}
	return ""
}

// GetStorageHTML returns the raw storage format HTML.
func (p *Page) GetStorageHTML() string {
	if p.Body != nil && p.Body.Storage != nil {
		return p.Body.Storage.Value
	}
	return ""
}

// GetLabels returns page labels as string slice.
func (p *Page) GetLabels() []string {
	if p.Metadata == nil || p.Metadata.Labels == nil {
		return nil
	}
	labels := make([]string, 0, len(p.Metadata.Labels.Results))
	for _, l := range p.Metadata.Labels.Results {
		labels = append(labels, l.Name)
	}
	return labels
}

// AncestorIDs returns the IDs of all ancestors (excluding the root).
func (p *Page) AncestorIDs() []string {
	if len(p.Ancestors) <= 1 {
		return nil
	}
	// Skip the first ancestor (root) to match Python behavior
	ids := make([]string, 0, len(p.Ancestors)-1)
	for _, a := range p.Ancestors[1:] {
		ids = append(ids, a.ID)
	}
	return ids
}

// AncestorTitles returns the titles of all ancestors (excluding the root).
func (p *Page) AncestorTitles() []string {
	if len(p.Ancestors) <= 1 {
		return nil
	}
	titles := make([]string, 0, len(p.Ancestors)-1)
	for _, a := range p.Ancestors[1:] {
		titles = append(titles, a.Title)
	}
	return titles
}

// SpaceKey extracts the space key from the page's expandable or space ref.
func (p *Page) SpaceKey() string {
	if p.Space.Key != "" {
		return p.Space.Key
	}
	return ""
}

// SourceURL constructs the source URL for this page.
func (p *Page) SourceURL(baseURL string) string {
	if p.Links.Base != "" && p.Links.WebUI != "" {
		return p.Links.Base + p.Links.WebUI
	}
	if baseURL != "" && p.Links.WebUI != "" {
		return baseURL + p.Links.WebUI
	}
	return fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", baseURL, p.ID)
}
