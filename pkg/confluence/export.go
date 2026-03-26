package confluence

import (
	"context"
	"encoding/json"
)

type ExportOpts struct {
	Format          string
	IncludeComments bool
}

type ExportResult struct {
	PageID  string `json:"pageId"`
	Title   string `json:"title"`
	Format  string `json:"format"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

func (c *Client) Export(ctx context.Context, pageID string, opts *ExportOpts) (*ExportResult, error) {
	if opts.Format == "" {
		opts.Format = "markdown"
	}

	var expand string
	switch opts.Format {
	case "markdown":
		expand = "body.export_view,space,version"
	case "html":
		expand = "body.storage.value,space,version"
	case "json":
		expand = "body.storage.value,space,version,metadata.labels"
	default:
		expand = "body.export_view,space,version"
	}

	page, err := c.GetPage(ctx, pageID, expand)
	if err != nil {
		return nil, err
	}

	result := &ExportResult{
		PageID: page.ID,
		Title:  page.Title,
		Format: opts.Format,
		URL:    c.BaseURL + page.Links.WebUI,
	}

	switch opts.Format {
	case "markdown":
		if page.Body != nil && page.Body.ExportView != nil && page.Body.ExportView.Value != "" {
			result.Content = page.Body.ExportView.Value
		} else if page.Body != nil && page.Body.Storage != nil {
			result.Content = page.Body.Storage.Value
		}
	case "html":
		if page.Body != nil && page.Body.Storage != nil {
			result.Content = page.Body.Storage.Value
		}
	case "json":
		data, _ := json.MarshalIndent(page, "", "  ")
		result.Content = string(data)
	}

	return result, nil
}
