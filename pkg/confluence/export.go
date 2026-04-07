package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ExportOpts controls the export behavior.
type ExportOpts struct {
	Format          string // markdown, html, json
	IncludeComments bool
	OutputPath      string // Directory to save output
	WithDescendants bool   // Export child pages recursively
	WithAttachments bool   // Download page attachments (default: false)
	Verbose         bool
}

// ExportResult holds the result of a single page export.
type ExportResult struct {
	PageID     string `json:"pageId"`
	Title      string `json:"title"`
	Format     string `json:"format"`
	Content    string `json:"content,omitempty"`
	URL        string `json:"url"`
	ExportPath string `json:"exportPath,omitempty"`
}

// Export exports a single page and returns the result.
func (c *Client) Export(ctx context.Context, pageID string, opts *ExportOpts) (*ExportResult, error) {
	if opts.Format == "" {
		opts.Format = "markdown"
	}

	page, err := c.GetPage(ctx, pageID, "")
	if err != nil {
		return nil, err
	}

	result := &ExportResult{
		PageID: page.ID,
		Title:  page.Title,
		Format: opts.Format,
		URL:    page.SourceURL(c.BaseURL),
	}

	switch opts.Format {
	case "markdown":
		html := page.GetExportViewHTML()
		md := ConvertToMarkdown(html)

		// Add front matter + title
		frontMatter := GenerateFrontMatter(page, c.BaseURL, "")
		titleLine := fmt.Sprintf("# %s\n\n", page.Title)
		result.Content = frontMatter + titleLine + md
	case "html":
		result.Content = page.GetExportViewHTML()
	case "json":
		data, err := json.MarshalIndent(page, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal page to JSON: %w", err)
		}
		result.Content = string(data)
	}

	// Write to disk if output path is specified
	if opts.OutputPath != "" {
		exportPath, err := WriteExportedPage(opts.OutputPath, page, result.Content)
		if err != nil {
			return result, err
		}
		result.ExportPath = exportPath

		// Write manifest entry
		entry := buildManifestEntry(page, c.BaseURL, exportPath)
		_ = WriteManifest(opts.OutputPath, entry)
	}

	return result, nil
}

// ExportWithDescendants exports a page and optionally all its descendant pages.
func (c *Client) ExportWithDescendants(ctx context.Context, pageID string, opts *ExportOpts, logger *ExportLogger) ([]*ExportResult, error) {
	pageIDs := []string{pageID}

	if opts.WithDescendants {
		descendants, err := c.GetDescendants(ctx, pageID)
		if err != nil {
			logger.LogError(NewExportError(pageID, "", PhaseAPIFetch, ErrHTTPTimeout, err.Error(), opts.OutputPath))
			return nil, fmt.Errorf("failed to get descendants: %w", err)
		}
		pageIDs = append(pageIDs, descendants...)
	}

	return c.ExportPages(ctx, pageIDs, opts, logger)
}

// ExportPages exports a list of pages by ID, concurrently.
func (c *Client) ExportPages(ctx context.Context, pageIDs []string, opts *ExportOpts, logger *ExportLogger) ([]*ExportResult, error) {
	if opts.OutputPath != "" {
		// Clean up old manifest
		manifestPath := filepath.Join(opts.OutputPath, "manifest.jsonl")
		_ = os.Remove(manifestPath)
	}

	var results []*ExportResult
	var mu sync.Mutex
	entries := make(map[string][]*ManifestEntry) // grouped by space key

	total := len(pageIDs)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Pool of 10 workers

	for i, pageID := range pageIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, id string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Log progress
			logger.LogProgress(idx+1, total, id, "")

			// Export single page with error recovery
			result, entry, err := c.exportSinglePage(ctx, id, opts, logger)
			if err != nil {
				return // Error already logged
			}

			mu.Lock()
			results = append(results, result)
			if entry != nil {
				entries[entry.SpaceKey] = append(entries[entry.SpaceKey], entry)
			}
			mu.Unlock()

			logger.LogSuccess(id, result.Title)
		}(i, pageID)
	}

	wg.Wait()

	// Generate space indexes
	if opts.OutputPath != "" {
		for spaceKey, spaceEntries := range entries {
			spaceName := spaceKey
			if len(spaceEntries) > 0 {
				spaceName = spaceEntries[0].SpaceName
			}
			if err := WriteSpaceIndex(opts.OutputPath, spaceKey, spaceName, spaceEntries); err != nil {
				logger.LogWarning("", "Space Index", fmt.Sprintf("failed to write space index for %s: %v", spaceKey, err))
			}
		}
	}

	return results, nil
}

// exportSinglePage exports one page with comprehensive error handling. Returns the ExportResult and ManifestEntry.
func (c *Client) exportSinglePage(ctx context.Context, pageID string, opts *ExportOpts, logger *ExportLogger) (*ExportResult, *ManifestEntry, error) {
	// Fetch page
	page, err := c.GetPage(ctx, pageID, "")
	if err != nil {
		errMsg := err.Error()
		code := ErrHTTPTimeout
		if strings.Contains(errMsg, "status 403") {
			code = ErrHTTP403
		} else if strings.Contains(errMsg, "status 404") {
			code = ErrHTTP404
		} else if strings.Contains(errMsg, "status 429") {
			code = ErrHTTP429
		}
		logger.LogError(NewExportError(pageID, "", PhaseAPIFetch, code, errMsg, opts.OutputPath))
		return nil, nil, err
	}

	// Convert to markdown
	html := page.GetExportViewHTML()
	md := ConvertToMarkdown(html)

	// Build full markdown with front matter
	frontMatter := GenerateFrontMatter(page, c.BaseURL, "")
	titleLine := fmt.Sprintf("# %s\n\n", page.Title)
	fullContent := frontMatter + titleLine + md

	result := &ExportResult{
		PageID: page.ID,
		Title:  page.Title,
		Format: "markdown",
		URL:    page.SourceURL(c.BaseURL),
	}

	var entry *ManifestEntry

	// Write to disk
	if opts.OutputPath != "" {
		exportPath, err := WriteExportedPage(opts.OutputPath, page, fullContent)
		if err != nil {
			// Classify the write error
			code := ErrWritePermission
			if strings.Contains(err.Error(), "path too long") || len(filepath.Join(opts.OutputPath, exportPath)) > 260 {
				code = ErrPathTooLong
			}
			exportErr := NewExportError(pageID, page.Title, PhaseWrite, code, err.Error(), opts.OutputPath)
			exportErr.GeneratedPath = exportPath
			logger.LogError(exportErr)
			return nil, nil, err
		}
		result.ExportPath = exportPath

		// Write manifest entry
		entry = buildManifestEntry(page, c.BaseURL, exportPath)
		mu := sync.Mutex{}
		mu.Lock()
		_ = WriteManifest(opts.OutputPath, entry)
		mu.Unlock()

		// Download attachments (only if explicitly requested)
		if opts.WithAttachments {
			c.downloadPageAttachments(ctx, page, opts, logger)
		}
	}

	return result, entry, nil
}

// downloadPageAttachments downloads all attachments for a page.
func (c *Client) downloadPageAttachments(ctx context.Context, page *Page, opts *ExportOpts, logger *ExportLogger) {
	attachments, err := c.GetAttachments(ctx, page.ID)
	if err != nil {
		logger.LogWarning(page.ID, page.Title, fmt.Sprintf("Failed to list attachments: %v", err))
		return
	}

	for _, att := range attachments {
		if att.Links.Download == "" {
			continue
		}

		data, err := c.DownloadAttachment(ctx, att.Links.Download)
		if err != nil {
			exportErr := NewExportError(page.ID, page.Title, PhaseAttachment, ErrAttachmentFail, err.Error(), opts.OutputPath)
			logger.LogError(exportErr)
			continue
		}

		ext := filepath.Ext(att.Title)
		_, err = WriteAttachment(opts.OutputPath, page, data, att.ID, att.Title, ext)
		if err != nil {
			exportErr := NewExportError(page.ID, page.Title, PhaseWrite, ErrWritePermission, err.Error(), opts.OutputPath)
			logger.LogError(exportErr)
		}

		logger.LogVerbose("📎 %s -> %s", att.Title, page.Title)
	}
}

func buildManifestEntry(page *Page, baseURL, exportPath string) *ManifestEntry {
	entry := &ManifestEntry{
		PageID:     page.ID,
		Title:      page.Title,
		Slug:       slugify(page.Title, 80),
		SourceURL:  page.SourceURL(baseURL),
		SpaceKey:   page.SpaceKey(),
		SpaceName:  page.Space.Name,
		Labels:     page.GetLabels(),
		ExportPath: exportPath,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Depth:      len(page.Ancestors),
		Ancestors:  page.Ancestors,
		Breadcrumb: strings.Join(append(page.AncestorTitles(), page.Title), " > "),
	}
	if len(page.Ancestors) > 0 {
		entry.ParentID = page.Ancestors[len(page.Ancestors)-1].ID
		entry.ParentTitle = page.Ancestors[len(page.Ancestors)-1].Title
	}
	return entry
}