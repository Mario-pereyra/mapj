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
	Debug           bool
	DumpDebug       bool   // Full diagnostic dump for a single page
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
		entry := &ManifestEntry{
			PageID:     page.ID,
			Title:      page.Title,
			Slug:       slugify(page.Title, 80),
			SourceURL:  page.SourceURL(c.BaseURL),
			SpaceKey:   page.SpaceKey(),
			SpaceName:  page.Space.Name,
			Labels:     page.GetLabels(),
			ExportPath: exportPath,
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
		}
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

// ExportPages exports a list of pages by ID, with structured logging.
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
	for i, pageID := range pageIDs {
		// Log progress
		logger.LogProgress(i+1, total, pageID, "")

		// Export single page with error recovery
		result, err := c.exportSinglePage(ctx, pageID, opts, logger)
		if err != nil {
			continue // Error already logged
		}

		mu.Lock()
		results = append(results, result)

		// Track for space index generation
		if page, err := c.GetPage(ctx, pageID, "space,metadata.labels,ancestors"); err == nil {
			entry := &ManifestEntry{
				PageID:     page.ID,
				Title:      page.Title,
				Slug:       slugify(page.Title, 80),
				SourceURL:  page.SourceURL(c.BaseURL),
				SpaceKey:   page.SpaceKey(),
				SpaceName:  page.Space.Name,
				Labels:     page.GetLabels(),
				ExportPath: result.ExportPath,
				ExportedAt: time.Now().UTC().Format(time.RFC3339),
			}
			entries[page.SpaceKey()] = append(entries[page.SpaceKey()], entry)
		}
		mu.Unlock()

		logger.LogSuccess(pageID, result.Title)
	}

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

// exportSinglePage exports one page with comprehensive error handling.
func (c *Client) exportSinglePage(ctx context.Context, pageID string, opts *ExportOpts, logger *ExportLogger) (*ExportResult, error) {
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
		return nil, err
	}

	// Debug dump if requested
	if opts.DumpDebug {
		dumpPageDebug(logger, page, c.BaseURL)
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
			return nil, err
		}
		result.ExportPath = exportPath

		// Write manifest entry
		entry := &ManifestEntry{
			PageID:     page.ID,
			Title:      page.Title,
			Slug:       slugify(page.Title, 80),
			SourceURL:  page.SourceURL(c.BaseURL),
			SpaceKey:   page.SpaceKey(),
			SpaceName:  page.Space.Name,
			Labels:     page.GetLabels(),
			ExportPath: exportPath,
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
		}
		_ = WriteManifest(opts.OutputPath, entry)

		// Download attachments (only if explicitly requested)
		if opts.WithAttachments {
			c.downloadPageAttachments(ctx, page, opts, logger)
		}
	}

	return result, nil
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

// dumpPageDebug writes diagnostic files for a single page.
func dumpPageDebug(logger *ExportLogger, page *Page, baseURL string) {
	logger.DumpDebugFile(page.ID, "raw_export_view.html", []byte(page.GetExportViewHTML()))
	logger.DumpDebugFile(page.ID, "raw_storage.html", []byte(page.GetStorageHTML()))
	logger.DumpDebugFile(page.ID, "converted.md", []byte(ConvertToMarkdown(page.GetExportViewHTML())))

	metadata, _ := json.MarshalIndent(map[string]interface{}{
		"id":        page.ID,
		"title":     page.Title,
		"space":     page.Space,
		"ancestors": page.Ancestors,
		"labels":    page.GetLabels(),
		"version":   page.Version,
		"links":     page.Links,
		"sourceURL": page.SourceURL(baseURL),
	}, "", "  ")
	logger.DumpDebugFile(page.ID, "metadata.json", metadata)
}
