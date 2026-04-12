package cli

import (
	"github.com/Mario-pereyra/mapj/internal/errors"
	"github.com/Mario-pereyra/mapj/internal/logging"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	outputFormat string
	jsonOutput   bool
	verbose      bool
	configPath   string
	profileName  string
	noColor      bool
	logLevel     string
)

var rootCmd = &cobra.Command{
	Use:     "mapj",
	Version: "0.2.0-agentic",
	Short:   "CLI for LLM agents: search TDN docs, export Confluence, query Protheus ERP",
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
  Auto-detected by default (TOON for tables, LLM for objects).
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
  mapj protheus query      Execute SELECT on Protheus SQL Server
  mapj protheus connection Manage named DB connection profiles
  mapj auth login          Authenticate a service
  mapj auth status         Show auth state for all services
  mapj auth logout         Remove stored credentials`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logging with trace ID
		traceID := logging.GenerateTraceID()
		logging.Init(logging.Config{
			Level:   logLevel,
			TraceID: traceID,
		})
		logging.Info("command started", zap.String("command", cmd.Name()))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tdnCmd, confluenceCmd, protheusCmd)
}

func Execute() int {
	// Logging flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")

	// Output format flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "Output format: auto (default), llm, toon, json")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in pure JSON format (alias for --output llm)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Include debug/trace fields in output (schemaVersion, timestamp)")

	// Configuration flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: ~/.config/mapj/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profileName, "profile", "", "Connection profile to use for this command")

	// Display flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Silence errors for all commands that output structured JSON errors
	// Each command that returns an ExitCoder error has already output its JSON envelope
	rootCmd.SilenceErrors = true

	if err := rootCmd.Execute(); err != nil {
		// Don't print raw error - the command has already output a structured JSON envelope
		// Just return the appropriate exit code
		if exitCoder, ok := err.(errors.ExitCoder); ok {
			return exitCoder.ExitCode()
		}
		return errors.ExitError
	}
	return errors.ExitSuccess
}

// GetFormatter returns the appropriate formatter based on flags.
// --json takes precedence over --output.
// --verbose enables human-mode fields (schemaVersion, timestamp).
func GetFormatter() output.Formatter {
	// --json takes precedence
	if jsonOutput {
		return output.NewFormatterWithVerbose("llm", verbose)
	}
	return output.NewFormatterWithVerbose(outputFormat, verbose)
}

// GetConfigPath returns the config file path from --config flag.
func GetConfigPath() string {
	return configPath
}

// GetProfile returns the profile name from --profile flag.
func GetProfile() string {
	return profileName
}

// IsNoColor returns true if --no-color was specified.
func IsNoColor() bool {
	return noColor
}

// IsVerbose returns true if --verbose was specified.
func IsVerbose() bool {
	return verbose
}
