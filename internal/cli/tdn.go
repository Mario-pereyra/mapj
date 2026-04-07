package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

// ─── Root TDN command ────────────────────────────────────────────────────────

var tdnCmd = &cobra.Command{
	Use:   "tdn",
	Short: "Search and explore TOTVS Developer Network (TDN) documentation",
	Long: `TDN (tdn.totvs.com) is the public TOTVS developer documentation portal.

No authentication required for public content.
Authentication only needed for private/internal TDN instances.

Subcommands:
  mapj tdn search <query>   Search documentation using CQL
  mapj tdn spaces list      List all available spaces (use keys in --space filter)

Run 'mapj tdn <command> --help' for the full output schema of each command.`,
}

// ─── tdn search ──────────────────────────────────────────────────────────────

var tdnSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search TDN documentation using CQL (no auth required)",
	Long: `Search TDN (tdn.totvs.com) documentation. No authentication required for public content.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj tdn search","result":{
    "results": [{
      "id":          "235312129",      // Confluence page ID — use for export
      "type":        "page",
      "title":       "AdvPL",
      "url":         "https://tdn.totvs.com/...",
      "childCount":  3               // only present with --check-children
    }],
    "count":   25,                   // results in this page
    "total":   1842,                 // total matching results
    "hasNext": true,                 // use --start N to paginate
    "cql":     "siteSearch ~ \"AdvPL\" AND ..."
  }}

GOTCHAS:
  - childCount counts DIRECT children only. childCount:1 can have 171+ total descendants.
    Always check before using --with-descendants on confluence export.
  - --since uses TDN's lastmodified field. Relative: 1w, 4d, 2m, 1y. Absolute: 2024-01-01.
  - Use result[*].id to feed into: mapj confluence export <id>
  - --export-to pipelines search results directly into confluence export (no extra step)

EXAMPLES:
  mapj tdn search "AdvPL" --space PROT --max-results 100
  mapj tdn search "ponto de entrada" --space PROT --since 1m
  mapj tdn search "advpl" --space PROT --label versao_12
  mapj tdn search "advpl" --space PROT --check-children
  mapj tdn search --ancestor 187531295 --space PROT
  mapj tdn search "advpl" --space PROT --export-to ./docs
  mapj tdn search "apostila" --spaces PROT,LDT --type attachment`,
	Args: cobra.MaximumNArgs(1),
	RunE: tdnSearchRun,
}

var (
	tdnSpace         string
	tdnSpaces        []string
	tdnMaxResults    int
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
	f.IntVar(&tdnMaxResults, "max-results", 25, "Max results to return (auto-paginates up to this number)")
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

	if tdnMaxResults <= 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "--max-results must be > 0", false)
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
		Limit:    tdnMaxResults,
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
		client.EnrichWithChildCount(ctx, result)
	}

	// ── Search → Export pipeline ─────────────────────────────────────────────
	if tdnExportTo != "" {
		summary, err := client.RunSearchExportPipeline(ctx, result, tdnExportTo)
		if err != nil {
			return err
		}
		env := output.NewEnvelope(cmd.CommandPath(), summary)
		fmt.Println(formatter.Format(env))
		return nil
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}

// ─── tdn spaces ──────────────────────────────────────────────────────────────

var tdnSpacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List available TDN spaces",
	Long: `List subcommands for TDN space discovery.

Subcommands:
  mapj tdn spaces list   Get all available space keys and names`,
}

var tdnSpacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available public TDN spaces (use keys in --space filter)",
	Long: `List all global spaces available in TDN (tdn.totvs.com).
Use the returned space keys in 'mapj tdn search --space KEY'.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj tdn spaces list","result":{
    "spaces": [{"key":"PROT","name":"Linha Microsiga Protheus","type":"global"}],
    "count":  192
  }}

EXAMPLE:
  mapj tdn spaces list
  # then: mapj tdn search "AdvPL" --space PROT`,
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
