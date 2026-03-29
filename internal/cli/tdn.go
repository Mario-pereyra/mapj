package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

// ─── Root TDN command ────────────────────────────────────────────────────────

var tdnCmd = &cobra.Command{
	Use:   "tdn",
	Short: "TOTVS Developer Network (TDN) commands",
	Long:  `Commands for searching and exploring the TOTVS Developer Network documentation (TDN).`,
}

// ─── tdn search ──────────────────────────────────────────────────────────────

var tdnSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search TDN documentation using CQL",
	Long: `Search the TOTVS Developer Network (TDN) for documentation.

Uses the siteSearch CQL field which searches across title, body, and labels simultaneously.
No authentication required for public TDN content (tdn.totvs.com).

Examples:
  mapj tdn search "AdvPL"
  mapj tdn search "ponto de entrada" --space PROT
  mapj tdn search "api rest" --space PROT --type page --since 1m
  mapj tdn search "apostila" --space PROT --type attachment
  mapj tdn search --ancestor 187531295 --space PROT
  mapj tdn search "advpl" --space PROT --label versao_12
  mapj tdn search "advpl" --space PROT --export-to ./docs`,
	Args: cobra.MaximumNArgs(1),
	RunE: tdnSearchRun,
}

var (
	tdnSpace    string
	tdnSpaces   []string
	tdnLimit    int
	tdnStart         int
	tdnType          string
	tdnSince         string
	tdnAncestor      string
	tdnLabel         string
	tdnLabels        []string
	tdnExportTo      string
	tdnCheckChildren bool
)

func init() {
	// Register subcommands
	tdnCmd.AddCommand(tdnSearchCmd)
	tdnCmd.AddCommand(tdnSpacesCmd)

	// tdn search flags
	f := tdnSearchCmd.Flags()
	f.StringVar(&tdnSpace, "space", "", "Filter by single space key (e.g. PROT)")
	f.StringSliceVar(&tdnSpaces, "spaces", nil, "Filter by multiple space keys (e.g. PROT,LDT)")
	f.IntVar(&tdnLimit, "limit", 25, "Max results per page (1–100)")
	f.IntVar(&tdnStart, "start", 0, "Pagination offset")
	f.StringVar(&tdnType, "type", "page", "Content type: page, blogpost, attachment")
	f.StringVar(&tdnSince, "since", "", `Filter by last modified date. Supports: "1w", "4d", "2m", "1y", "2024-01-01"`)
	f.StringVar(&tdnAncestor, "ancestor", "", "Return all pages under a given page ID (page tree export)")
	f.StringVar(&tdnLabel, "label", "", "Filter by a single label/tag")
	f.StringSliceVar(&tdnLabels, "labels", nil, "Filter by multiple labels (AND logic)")
	f.StringVar(&tdnExportTo, "export-to", "", "Export each found page to Markdown in this directory (search→export pipeline)")
	f.BoolVar(&tdnCheckChildren, "check-children", false, "Add childCount to each result (extra API call per page, helps decide --with-descendants)")
}

func tdnSearchRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	formatter := GetFormatter()

	if tdnLimit <= 0 || tdnLimit > 100 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "--limit must be between 1 and 100", false)
		fmt.Println(formatter.Format(env))
		return errors.New("USAGE_ERROR")
	}

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// Validate: query OR ancestor required
	if query == "" && tdnAncestor == "" && tdnLabel == "" && len(tdnLabels) == 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR",
			"provide a search query, --ancestor ID, or --label LABEL", false)
		fmt.Println(formatter.Format(env))
		return errors.New("USAGE_ERROR")
	}

	client, err := getTDNClient(ctx)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	opts := &confluence.SearchOpts{
		Query:    query,
		Space:    tdnSpace,
		Spaces:   tdnSpaces,
		Label:    tdnLabel,
		Labels:   tdnLabels,
		Type:     tdnType,
		Ancestor: tdnAncestor,
		Since:    tdnSince,
		Limit:    tdnLimit,
		Start:    tdnStart,
	}

	result, err := client.Search(ctx, opts)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "SEARCH_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	// ── Check children (optional enrichment) ──────────────────────────────────
	if tdnCheckChildren {
		enrichWithChildCount(ctx, client, result)
	}

	// ── Search → Export pipeline ─────────────────────────────────────────────
	if tdnExportTo != "" {
		return runSearchExportPipeline(ctx, cmd, formatter, client, result, opts)
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}

// enrichWithChildCount fetches child counts for all results concurrently.
// Uses a semaphore to limit concurrent API calls to 5 at a time.
func enrichWithChildCount(ctx context.Context, client *confluence.Client, result *confluence.SearchResult) {
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

			count, err := client.GetChildCount(ctx, result.Results[idx].ID)
			if err != nil {
				count = -1 // -1 = fetch error
			}
			result.Results[idx].ChildCount = &count
		}(i)
	}
	wg.Wait()
}

// runSearchExportPipeline exports every page found in the search results.
func runSearchExportPipeline(
	ctx context.Context,
	cmd *cobra.Command,
	formatter output.Formatter,
	client *confluence.Client,
	result *confluence.SearchResult,
	opts *confluence.SearchOpts,
) error {
	type pipelineResult struct {
		Searched  int      `json:"searched"`
		Exported  int      `json:"exported"`
		Failed    int      `json:"failed"`
		OutputDir string   `json:"outputDir"`
		Pages     []string `json:"pages"`
		Errors    []string `json:"errors,omitempty"`
	}

	absDir, _ := filepath.Abs(tdnExportTo)
	summary := pipelineResult{
		Searched:  result.Count,
		OutputDir: absDir,
	}

	exportOpts := &confluence.ExportOpts{
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
		_, err := client.Export(ctx, page.ID, exportOpts)
		if err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (%s): %v", page.Title, page.ID, err))
		} else {
			summary.Exported++
			summary.Pages = append(summary.Pages, fmt.Sprintf("%s (%s)", page.Title, page.ID))
		}
	}

	env := output.NewEnvelope(cmd.CommandPath(), summary)
	fmt.Println(formatter.Format(env))
	return nil
}

// ─── tdn spaces ──────────────────────────────────────────────────────────────

var tdnSpacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List available TDN spaces",
}

var tdnSpacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available public spaces in TDN",
	Long: `List all global, current spaces available in TDN.

Examples:
  mapj tdn spaces list
  mapj tdn spaces list --output table`,
	RunE: tdnSpacesListRun,
}

func init() {
	tdnSpacesCmd.AddCommand(tdnSpacesListCmd)
}

func tdnSpacesListRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	formatter := GetFormatter()

	client, err := getTDNClient(ctx)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	spaces, err := client.GetAllSpaces(ctx)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "SEARCH_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	type spaceOut struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	out := make([]spaceOut, 0, len(spaces))
	for _, s := range spaces {
		out = append(out, spaceOut{Key: s.Key, Name: s.Name, Type: s.Type})
	}

	result := map[string]interface{}{
		"spaces": out,
		"count":  len(out),
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}

// ─── Shared helper ───────────────────────────────────────────────────────────

// getTDNClient builds a Confluence client for TDN.
// Auth is OPTIONAL — tdn.totvs.com serves public content without a token.
// If credentials are stored, they are used to access private content.
func getTDNClient(ctx context.Context) (*confluence.Client, error) {
	// Try to load stored credentials (optional)
	store, err := auth.NewStore()
	if err == nil {
		creds, err := store.Load()
		if err == nil && creds.TDN != nil && creds.TDN.Token != "" {
			baseURL := creds.TDN.BaseURL
			if baseURL == "" {
				baseURL = "https://tdn.totvs.com"
			}
			baseURL = strings.TrimSuffix(baseURL, "/")

			client := confluence.NewClient(baseURL, creds.TDN.Token)
			if creds.TDN.Username != "" {
				client.SetBasicAuth(creds.TDN.Username, creds.TDN.Token)
			}
			return client, nil
		}
	}

	// No credentials — use public endpoint (no auth)
	return confluence.NewClient("https://tdn.totvs.com", ""), nil
}
