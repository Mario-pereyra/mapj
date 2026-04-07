package cli

import (
	"fmt"
	"os"

	"github.com/Mario-pereyra/mapj/internal/errors"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "mapj",
	Short: "CLI for LLM agents: search TDN docs, export Confluence, query Protheus ERP",
	Long: `mapj — Agentic CLI for the TOTVS ecosystem.

What it does:
  Search documentation on TDN (tdn.totvs.com), export Confluence pages to Markdown,
  and execute SELECT queries on Protheus ERP SQL Server databases.

ONBOARDING — run in this order:
  1. mapj auth status                          # check what is already authenticated
  2. mapj auth login confluence --url URL --token TOKEN   # authenticate Confluence/TDN
  3. mapj protheus connection add NAME \       # register Protheus DB connection
       --server HOST --database DB --user U --password P --use
  4. mapj tdn search "your query" --space PROT # start searching

OUTPUT FORMAT (all commands):
  Default (-o llm): compact JSON, no extra metadata — optimized for token efficiency.
  Human  (-o json): indented JSON + schemaVersion + timestamp.
  All output goes to stdout. Progress/logs go to stderr.

  Success envelope:  {"ok":true,  "command":"...","result":{...}}
  Error envelope:    {"ok":false, "command":"...","error":{"code":"...","message":"...","hint":"...","retryable":false}}

EXIT CODES:
  0  Success — parse result
  1  General / auth error — read error.message
  2  Usage error (bad args, forbidden SQL) — fix command and retry
  3  Auth error — run mapj auth login <service>
  4  Retryable (timeout, rate limit) — wait 2s, retry up to 3x

AVAILABLE COMMANDS (run --help on each for full schema):
  mapj tdn search          Search TDN docs (no auth required for public content)
  mapj tdn spaces list     List all available TDN spaces
  mapj confluence export   Export page(s) to Markdown files
  mapj confluence export-space  Export an entire Confluence space
  mapj confluence retry-failed  Retry previously failed exports
  mapj protheus query      Execute SELECT on Protheus SQL Server
  mapj protheus connection Manage named DB connection profiles
  mapj auth login          Authenticate a service
  mapj auth status         Show auth state for all services
  mapj auth logout         Remove stored credentials`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tdnCmd, confluenceCmd, protheusCmd)
}

func Execute() int {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "llm", "Output format: llm (compact JSON, default), json (pretty JSON), csv, toon (compact tabular)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return errors.MapErrorToCode(err)
	}
	return errors.ExitSuccess
}

func GetFormatter() output.Formatter {
	return output.NewFormatter(outputFormat)
}
