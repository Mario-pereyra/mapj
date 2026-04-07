package confluence

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// PipelineResult summarizes the outcome of a search-to-export pipeline.
type PipelineResult struct {
	Searched  int      `json:"searched"`
	Exported  int      `json:"exported"`
	Failed    int      `json:"failed"`
	OutputDir string   `json:"outputDir"`
	Pages     []string `json:"pages"`
	Errors    []string `json:"errors,omitempty"`
}

// EnrichWithChildCount fetches child counts for all results concurrently.
// Uses a semaphore to limit concurrent API calls and adds a strict timeout to each.
func (c *Client) EnrichWithChildCount(ctx context.Context, result *SearchResult) {
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i := range result.Results {
		if result.Results[i].Type != "page" || result.Results[i].ID == "" {
			count := 0
			result.Results[i].ChildCount = &count
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// 2-second strict timeout per child count request
			reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			count, err := c.GetChildCount(reqCtx, result.Results[idx].ID)
			if err != nil {
				count = -1 // -1 = fetch error or timeout
			}
			result.Results[idx].ChildCount = &count
		}(i)
	}
	wg.Wait()
}

// RunSearchExportPipeline exports every page found in the search results to the given path.
func (c *Client) RunSearchExportPipeline(ctx context.Context, result *SearchResult, exportTo string) (*PipelineResult, error) {
	absDir, _ := filepath.Abs(exportTo)
	summary := &PipelineResult{
		Searched:  result.Count,
		OutputDir: absDir,
	}

	exportOpts := &ExportOpts{
		OutputPath:      absDir,
		WithDescendants: false,
		WithAttachments: false,
	}

	for _, page := range result.Results {
		if page.ID == "" {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("page '%s' has no ID", page.Title))
			continue
		}
		_, err := c.Export(ctx, page.ID, exportOpts)
		if err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (%s): %v", page.Title, page.ID, err))
		} else {
			summary.Exported++
			summary.Pages = append(summary.Pages, fmt.Sprintf("%s (%s)", page.Title, page.ID))
		}
	}

	return summary, nil
}
