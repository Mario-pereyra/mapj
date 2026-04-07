package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

var confluenceCmd = &cobra.Command{
	Use:   "confluence",
	Short: "Export Confluence pages to Markdown files",
	Long: `Export Confluence pages (or entire spaces) to Markdown files with YAML front matter.

AUTH REQUIRED:
  Run first: mapj auth login confluence --url BASEURL --token TOKEN
  Check status: mapj auth status

  Server/DC (e.g. tdninterno.totvs.com) → Bearer PAT, no --username
  Cloud (company.atlassian.net) → Basic Auth, requires --username you@company.com

SUBCOMMANDS:
  mapj confluence export <url-or-id>     Export one page (or tree with --with-descendants)
  mapj confluence export-space <key>     Export every page in a space

Run 'mapj confluence <command> --help' for full schema.`,
}

// ==================== FLAGS ====================

var (
	confluenceFormat      string
	confluenceOutputPath  string
	confluenceVerbose     bool
	confluenceDescendants bool
	confluenceAttachments bool
)

// ==================== COMMANDS ====================

var confluenceExportCmd = &cobra.Command{
	Use:   "export <url-or-page-id>",
	Short: "Export Confluence page(s) to Markdown with YAML front matter",
	Long: `Export a Confluence page to Markdown. Auth required (run mapj auth status first).

INPUT FORMATS (all equivalent):
  mapj confluence export 235312129                                         # Page ID
  mapj confluence export https://tdn.totvs.com/display/PROT/AdvPL         # Server URL
  mapj confluence export https://tdn.totvs.com/pages/viewpage.action?pageId=235312129
  mapj confluence export https://company.atlassian.net/wiki/spaces/X/pages/235312129

OUTPUT STRUCTURE (files created under --output-path):
  output-path/
  ├── spaces/SPACE_KEY/
  │   ├── README.md                 ← index of all exported pages
  │   └── pages/
  │       └── PAGE_ID-slug.md       ← one file per page, YAML front matter
  ├── manifest.jsonl                ← one JSON line per exported page (inventory)

STDOUT SCHEMA (final summary):
  {"ok":true,"command":"mapj confluence export","result":{
    "exported":   1,
    "outputPath": "./docs"
  }}

GOTCHAS:
  - --with-descendants exports the full subtree recursively.
    Use 'mapj tdn search --check-children' BEFORE to know how large the tree is.
    childCount:1 can mean 171+ total pages. Preview first.
  - Without --output-path, single-page export prints Markdown to stdout.

EXAMPLES:
  mapj confluence export 235312129 --output-path ./docs
  mapj confluence export 235312129 --output-path ./docs --with-descendants
  mapj confluence export https://tdn.totvs.com/display/PROT/AdvPL --output-path ./docs
  mapj confluence export 235312129 --output-path ./docs --with-attachments`,
	Args: cobra.ExactArgs(1),
	RunE: confluenceExportRun,
}

var confluenceExportSpaceCmd = &cobra.Command{
	Use:   "export-space <space-key>",
	Short: "Export all pages in a Confluence space to Markdown",
	Long: `Export every page in a Confluence space. Auth required.

Get space keys from: mapj tdn spaces list

OUTPUT STRUCTURE (same as 'confluence export'):
  output-path/spaces/SPACE_KEY/
    README.md          ← full index
    pages/*.md         ← one file per page
  manifest.jsonl       ← inventory of all exported pages

STDOUT SCHEMA:
  {"ok":true,"command":"mapj confluence export-space","result":{
    "exported":   342,
    "outputPath": "./docs"
  }}

EXAMPLES:
  mapj confluence export-space PROT --output-path ./docs
  mapj confluence export-space PROT --output-path ./docs --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: confluenceExportSpaceRun,
}

// ==================== INIT ====================

func init() {
	// Export command flags
	confluenceExportCmd.Flags().StringVar(&confluenceOutputPath, "output-path", "", "Directory to save exported files")
	confluenceExportCmd.Flags().StringVar(&confluenceFormat, "format", "markdown", "Output format (markdown, html, json)")
	confluenceExportCmd.Flags().BoolVar(&confluenceDescendants, "with-descendants", false, "Also export all child pages recursively")
	confluenceExportCmd.Flags().BoolVar(&confluenceAttachments, "with-attachments", false, "Download page attachments (images, files, etc.)")
	confluenceExportCmd.Flags().BoolVar(&confluenceVerbose, "verbose", false, "Show detailed progress and warnings")

	// Export-space command flags
	confluenceExportSpaceCmd.Flags().StringVar(&confluenceOutputPath, "output-path", "", "Directory to save exported files")
	confluenceExportSpaceCmd.Flags().BoolVar(&confluenceAttachments, "with-attachments", false, "Download page attachments (images, files, etc.)")
	confluenceExportSpaceCmd.Flags().BoolVar(&confluenceVerbose, "verbose", false, "Show detailed progress")
	confluenceExportSpaceCmd.MarkFlagRequired("output-path")

	// Register subcommands
	confluenceCmd.AddCommand(confluenceExportCmd, confluenceExportSpaceCmd)
}

// ==================== HELPERS ====================

// getConfluenceClient creates an authenticated Confluence client from stored credentials.
func getConfluenceClient(cmd *cobra.Command) (*confluence.Client, *auth.ConfluenceCreds, error) {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil, nil, err
	}

	creds, err := store.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil, nil, err
	}

	if creds.Confluence == nil || creds.Confluence.Token == "" {
		msg := "Run 'mapj auth login confluence --url URL --token TOKEN' first"
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", msg, false)
		fmt.Println(formatter.Format(env))
		return nil, nil, errors.New("NOT_AUTHENTICATED: " + msg)
	}

	client := confluence.NewClient(creds.Confluence.BaseURL, creds.Confluence.Token)

	// Use stored AuthType to apply the correct auth scheme.
	// Legacy creds without AuthType default to bearer (backward-compatible).
	switch creds.Confluence.AuthType {
	case "basic":
		// Confluence Cloud: email + API token → Basic Auth
		if creds.Confluence.Username == "" {
			msg := "Basic auth requires a username. Re-run: mapj auth login confluence --url URL --username EMAIL --token TOKEN"
			env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_CONFIG_ERROR", msg, false)
			fmt.Println(formatter.Format(env))
			return nil, nil, errors.New(msg)
		}
		client.SetBasicAuth(creds.Confluence.Username, creds.Confluence.Token)
	default:
		// "bearer" or empty (legacy) → Confluence Server/DC PAT
		// Token is already set in NewClient. Nothing else needed.
	}

	return client, creds.Confluence, nil
}

// getLogLevel returns the appropriate log level based on CLI flags.
func getLogLevel() confluence.LogLevel {
	if confluenceVerbose {
		return confluence.LogVerbose
	}
	return confluence.LogNormal
}

// ==================== EXPORT COMMAND ====================

func confluenceExportRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	input := args[0]
	formatter := GetFormatter()

	client, confCreds, err := getConfluenceClient(cmd)
	if err != nil {
		return err
	}

	// Parse input to get page ID
	parseResult, err := confluence.ParseConfluenceInput(input)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "INVALID_URL", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	// If URL provided a base URL, use it
	if parseResult.BaseURL != "" {
		client = confluence.NewClient(parseResult.BaseURL, confCreds.Token)
		if confCreds.Username != "" {
			client.SetBasicAuth(confCreds.Username, confCreds.Token)
		}
	}

	// Resolve page ID (may need API call for title-based URLs)
	pageID, err := client.ResolvePageID(ctx, parseResult)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "PAGE_NOT_FOUND", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	opts := &confluence.ExportOpts{
		Format:          confluenceFormat,
		OutputPath:      confluenceOutputPath,
		WithDescendants: confluenceDescendants,
		WithAttachments: confluenceAttachments,
		Verbose:         confluenceVerbose,
	}

	// Single page without descendants and no output path -> inline result
	if !confluenceDescendants && confluenceOutputPath == "" {
		result, err := client.Export(ctx, pageID, opts)
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "EXPORT_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}
		env := output.NewEnvelope(cmd.CommandPath(), result)
		fmt.Println(formatter.Format(env))
		return nil
	}

	// Multi-page or file output -> use logger
	if confluenceOutputPath == "" {
		confluenceOutputPath = "."
	}

	logger, err := confluence.NewExportLogger(confluenceOutputPath, getLogLevel())
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_WRITE_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer logger.Close()

	results, err := client.ExportWithDescendants(ctx, pageID, opts, logger)
	if err != nil {
		return err
	}

	// Print summary
	logger.PrintSummary(os.Stderr)

	// Also output structured result
	summary := map[string]interface{}{
		"exported":   len(results),
		"outputPath": confluenceOutputPath,
	}
	env := output.NewEnvelope(cmd.CommandPath(), summary)
	fmt.Println(formatter.Format(env))
	return nil
}

// ==================== EXPORT-SPACE COMMAND ====================

func confluenceExportSpaceRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	spaceKey := args[0]

	client, _, err := getConfluenceClient(cmd)
	if err != nil {
		return err
	}

	logger, err := confluence.NewExportLogger(confluenceOutputPath, getLogLevel())
	if err != nil {
		formatter := GetFormatter()
		env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_WRITE_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer logger.Close()

	fmt.Fprintf(os.Stderr, "📦 Fetching pages for space: %s\n", spaceKey)

	pageIDs, err := client.GetSpacePageIDs(ctx, spaceKey)
	if err != nil {
		formatter := GetFormatter()
		env := output.NewErrorEnvelope(cmd.CommandPath(), "SPACE_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	fmt.Fprintf(os.Stderr, "📄 Found %d pages in space %s\n\n", len(pageIDs), spaceKey)

	opts := &confluence.ExportOpts{
		Format:          "markdown",
		OutputPath:      confluenceOutputPath,
		WithAttachments: confluenceAttachments,
		Verbose:         confluenceVerbose,
	}

	results, _ := client.ExportPages(ctx, pageIDs, opts, logger)

	logger.PrintSummary(os.Stderr)

	summary := map[string]interface{}{
		"space":      spaceKey,
		"exported":   len(results),
		"outputPath": confluenceOutputPath,
	}
	formatter := GetFormatter()
	env := output.NewEnvelope(cmd.CommandPath(), summary)
	fmt.Println(formatter.Format(env))
	return nil
}