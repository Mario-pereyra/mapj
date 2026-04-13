package cli

import (
	"context"
	"time"

	"github.com/Mario-pereyra/mapj/internal/errors"
	"github.com/Mario-pereyra/mapj/internal/logging"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
	jsonOutput   bool
	verbose      bool
	configPath   string
	profileName  string
	noColor      bool
	logLevel     string
	observe      bool // --observe flag for opt-in observability
	// commandStartTime tracks when the current command started executing
	commandStartTime time.Time
	// currentCommand tracks the current executing command for post-run logging
	currentCommand *cobra.Command
	// currentRunErr captures any error from command execution for observability
	currentRunErr error
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
		// Capture start time and command for post-run logging
		commandStartTime = time.Now()
		currentCommand = cmd

		// Initialize logging with trace ID
		traceID := logging.GenerateTraceID()
		logging.Init(logging.Config{
			Level:   logLevel,
			TraceID: traceID,
		})

		// Log command start with full command path
		logging.LogCommandStart(cmd)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Note: We can't access the run error here directly
		// The error is logged separately in Execute()
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

	// Observability flag
	rootCmd.PersistentFlags().BoolVar(&observe, "observe", false, "Enable observability middleware (or set MAPJ_OBSERVE=1)")

	// Silence errors for all commands that output structured JSON errors
	// Each command that returns an ExitCoder error has already output its JSON envelope
	rootCmd.SilenceErrors = true

	// Set global observeEnabled based on flag
	observeEnabled = observe

	var runErr error
	if runErr = rootCmd.Execute(); runErr != nil {
		// Capture runErr for observability
		currentRunErr = runErr

		// Call Observe() for registered observables if enabled
		if currentCommand != nil && isObservabilityEnabled() {
			duration := time.Since(commandStartTime)
			observeCommand(context.Background(), currentCommand, runErr, duration)
		}

		// Log error completion with latency and status
		if currentCommand != nil {
			duration := time.Since(commandStartTime)
			logging.LogCommandComplete(currentCommand, duration, false)
		}

		// Don't print raw error - the command has already output a structured JSON envelope
		// Just return the appropriate exit code
		var exitCode int
		if exitCoder, ok := runErr.(errors.ExitCoder); ok {
			exitCode = exitCoder.ExitCode()
		} else {
			exitCode = errors.ExitError
		}

		// Record metrics for this command execution
		if currentCommand != nil {
			duration := time.Since(commandStartTime)
			RecordCommandMetrics(currentCommand.CommandPath(), exitCode, duration)
		}

		return exitCode
	}

	// Capture success for observability
	currentRunErr = nil

	// Call Observe() for registered observables if enabled
	if currentCommand != nil && isObservabilityEnabled() {
		duration := time.Since(commandStartTime)
		observeCommand(context.Background(), currentCommand, nil, duration)
	}

	// Log success completion with latency and status
	if currentCommand != nil {
		duration := time.Since(commandStartTime)
		logging.LogCommandComplete(currentCommand, duration, true)

		// Record metrics for successful command execution
		RecordCommandMetrics(currentCommand.CommandPath(), errors.ExitSuccess, duration)
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
