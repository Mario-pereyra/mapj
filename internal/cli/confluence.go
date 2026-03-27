package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

var confluenceCmd = &cobra.Command{
	Use:   "confluence",
	Short: "Confluence commands",
}

var confluenceExportCmd = &cobra.Command{
	Use:   "export <url-or-page-id>",
	Short: "Export Confluence page to markdown, HTML, or JSON",
	Long: `Export a Confluence page to various formats.

Examples:
  mapj confluence export https://company.atlassian.net/wiki/spaces/TEAM/pages/12345 --format markdown
  mapj confluence export 12345 --format html
  mapj confluence export --format json`,
	Args: cobra.ExactArgs(1),
	RunE: confluenceExportRun,
}

var confluenceFormat string
var confluenceIncludeComments bool
var confluenceOutputPath string

func init() {
	confluenceCmd.AddCommand(confluenceExportCmd)
	confluenceExportCmd.Flags().StringVar(&confluenceFormat, "format", "markdown", "Output format (markdown, html, json)")
	confluenceExportCmd.Flags().BoolVar(&confluenceIncludeComments, "include-comments", false, "Include page comments")
	confluenceExportCmd.Flags().StringVar(&confluenceOutputPath, "output-path", "", "Save output to file")
}

func confluenceExportRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	input := args[0]
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil
	}
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	if creds.Confluence == nil || creds.Confluence.Token == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", "Run 'mapj auth login confluence --url URL --token TOKEN' first", false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	baseURL, pageID, err := confluence.ParseConfluenceURL(input)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "INVALID_URL", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	if baseURL == "" {
		baseURL = creds.Confluence.BaseURL
	}

	client := confluence.NewClient(baseURL, creds.Confluence.Token)
	if creds.Confluence.Username != "" {
		client.SetBasicAuth(creds.Confluence.Username, creds.Confluence.Token)
	}

	opts := &confluence.ExportOpts{
		Format:          confluenceFormat,
		IncludeComments: confluenceIncludeComments,
	}

	result, err := client.Export(ctx, pageID, opts)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "EXPORT_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return nil
	}

	if confluenceOutputPath != "" {
		if err := os.WriteFile(confluenceOutputPath, []byte(result.Content), 0644); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_WRITE_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return nil
		}
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}
