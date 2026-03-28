package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

var retryErrorCode string

var confluenceRetryCmd = &cobra.Command{
	Use:   "retry-failed",
	Short: "Re-export pages that failed in a previous export",
	Long: `Read export-errors.jsonl from a previous export and re-export only the failed pages.

Examples:
  mapj confluence retry-failed --output-path ./docs
  mapj confluence retry-failed --output-path ./docs --error-code PATH_TOO_LONG
  mapj confluence retry-failed --output-path ./docs --verbose`,
	RunE: confluenceRetryRun,
}

func init() {
	confluenceRetryCmd.Flags().StringVar(&confluenceOutputPath, "output-path", ".", "Directory containing export-errors.jsonl")
	confluenceRetryCmd.Flags().StringVar(&retryErrorCode, "error-code", "", "Only retry pages with this error code")
	confluenceRetryCmd.Flags().BoolVar(&confluenceAttachments, "with-attachments", false, "Download page attachments (images, files, etc.)")
	confluenceRetryCmd.Flags().BoolVar(&confluenceVerbose, "verbose", false, "Show detailed progress")
	confluenceRetryCmd.Flags().BoolVar(&confluenceDebug, "debug", false, "Enable debug output")
	confluenceRetryCmd.MarkFlagRequired("output-path")

	confluenceCmd.AddCommand(confluenceRetryCmd)
}

// errorLogEntry matches the JSONL format in export-errors.jsonl
type errorLogEntry struct {
	Timestamp string `json:"ts"`
	PageID    string `json:"page_id"`
	Title     string `json:"title"`
	Phase     string `json:"phase"`
	Code      string `json:"error_code"`
	Message   string `json:"message"`
	SourceURL string `json:"source_url,omitempty"`
	RetryCmd  string `json:"retry_cmd,omitempty"`
}

func confluenceRetryRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	formatter := GetFormatter()

	errLogPath := filepath.Join(confluenceOutputPath, "export-errors.jsonl")
	file, err := os.Open(errLogPath)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_READ_ERROR",
			fmt.Sprintf("Cannot read error log: %v", err), false)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer file.Close()

	// Parse error log entries
	var entries []errorLogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry errorLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // Skip unparseable lines
		}

		// Filter by error code if specified
		if retryErrorCode != "" && entry.Code != retryErrorCode {
			continue
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		msg := "No failed pages to retry"
		if retryErrorCode != "" {
			msg = fmt.Sprintf("No failed pages with error code %q", retryErrorCode)
		}
		env := output.NewEnvelope(cmd.CommandPath(), map[string]string{"message": msg})
		fmt.Println(formatter.Format(env))
		return nil
	}

	fmt.Fprintf(os.Stderr, "🔁 Retrying %d failed pages\n\n", len(entries))

	// Get client
	client, _, err := getConfluenceClient(cmd)
	if err != nil {
		return err
	}

	// Create new logger
	logger, err := confluence.NewExportLogger(confluenceOutputPath, getLogLevel())
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_WRITE_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer logger.Close()

	// Collect page IDs for retry
	pageIDs := make([]string, 0, len(entries))
	for _, e := range entries {
		pageIDs = append(pageIDs, e.PageID)
	}

	opts := &confluence.ExportOpts{
		Format:          "markdown",
		OutputPath:      confluenceOutputPath,
		WithAttachments: confluenceAttachments,
		Verbose:         confluenceVerbose,
		Debug:           confluenceDebug,
	}

	results, _ := client.ExportPages(ctx, pageIDs, opts, logger)

	logger.PrintSummary(os.Stderr)

	summary := map[string]interface{}{
		"retried":   len(entries),
		"succeeded": len(results),
		"failed":    len(entries) - len(results),
	}
	env := output.NewEnvelope(cmd.CommandPath(), summary)
	fmt.Println(formatter.Format(env))
	return nil
}
