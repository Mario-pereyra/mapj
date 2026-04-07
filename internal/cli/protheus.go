package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
	"github.com/spf13/cobra"
)

var protheusCmd = &cobra.Command{
	Use:   "protheus",
	Short: "Execute SELECT queries and manage connections to Protheus ERP SQL Server",
	Long: `Query Protheus ERP SQL Server databases and manage named connection profiles.

TWO-STEP MODEL:
  1. Register a connection profile (one-time setup):
     mapj protheus connection add MYDB --server HOST --database DB --user U --password P --use

  2. Query the active connection:
     mapj protheus query "SELECT TOP 10 * FROM SA1010"

SUBCOMMANDS:
  mapj protheus query <sql>                 Execute a SELECT query
  mapj protheus connection list             List all registered profiles
  mapj protheus connection add <name>       Register a new profile
  mapj protheus connection use <name>       Switch the active profile
  mapj protheus connection ping [name]      Test connectivity
  mapj protheus connection show [name]      Show profile details
  mapj protheus connection remove <name>    Delete a profile

Run 'mapj protheus <command> --help' for full output schema.`,
}

var protheusQueryCmd = &cobra.Command{
	Use:   "query <sql>",
	Short: "Execute SELECT query on Protheus SQL Server (read-only enforced)",
	Long: `Execute a SELECT query on the active Protheus SQL Server connection.

SOURCE OF TRUTH: run 'mapj protheus connection list' first to know the active profile.
Run 'mapj auth status' to confirm protheus is authenticated.

OUTPUT SCHEMA (-o json, default):
  {"ok":true,"command":"mapj protheus query","result":{
    "columns": ["A1_COD","A1_NOME"],
    "rows":    [["000001","CLIENTE TESTE"],...],
    "count":   10
  }}

OUTPUT SCHEMA (-o csv):
  A1_COD,A1_NOME
  000001,CLIENTE TESTE
  (raw RFC 4180 CSV, no envelope wrapper)

OUTPUT SCHEMA (-o toon):
  ok: true
  command: "mapj protheus query"
  result[N]{A1_COD,A1_NOME}:
    000001,CLIENTE TESTE
    ...

OUTPUT SCHEMA (--output-file, any format):
  stdout → {"ok":true,"result":{"rows":1500,"columns":45,"format":"json","output_file":"./r.json"}}
  file   → full result (json or csv)
  Use when result > ~200 rows to avoid saturating LLM context.

SECURITY — SELECT-only enforcement:
  Blocked keywords: INSERT UPDATE DELETE MERGE CREATE ALTER DROP TRUNCATE
                    EXEC EXECUTE INTO REPLACE GRANT REVOKE BACKUP RESTORE
  Only SELECT and WITH (CTEs) are allowed. Error code: USAGE_ERROR, exit 2.
  Tip: avoid SELECT INTO #temp — use CTEs: WITH t AS (SELECT ...) SELECT * FROM t

FLAGS:
  --connection NAME    Run against specific profile WITHOUT switching active
                       Use to compare data across environments:
                       mapj protheus query "SELECT COUNT(*) FROM SA1010" --connection TOTALPEC_PRD
  -o, --output FORMAT  Output format: llm (default), json, csv, toon
  --max-rows N         Client-side row cap, default 10000. Prefer TOP N in SQL.
  --output-file PATH   Write result to file, stdout gets summary only.

EXAMPLES:
  mapj protheus query "SELECT TOP 10 A1_COD, A1_NOME FROM SA1010"
  mapj protheus query "SELECT DB_NAME() AS db, @@SERVERNAME AS srv"
  mapj protheus query "SELECT COUNT(*) AS total FROM SA1010" --connection TOTALPEC_PRD
  mapj protheus query "SELECT * FROM SA1010" --output-file ./sa1010.json
  mapj protheus query "SELECT * FROM SA1010" -o csv --output-file ./sa1010.csv
  mapj protheus query "SELECT * FROM SA1010" -o toon`,
	Args: cobra.ExactArgs(1),
	RunE: protheusQueryRun,
}

var protheusMaxRows int
var protheusConnection string // --connection: run against specific profile without switching
var protheusOutputFile string // --output-file: write result to file instead of stdout

func init() {
	protheusCmd.AddCommand(protheusQueryCmd)
	protheusQueryCmd.Flags().IntVar(&protheusMaxRows, "max-rows", 10000, "Max rows to return (0 = no limit)")
	protheusQueryCmd.Flags().StringVar(&protheusConnection, "connection", "", "Run against this named profile without switching the active connection")
	protheusQueryCmd.Flags().StringVar(&protheusOutputFile, "output-file", "", "Write query result to this file path instead of stdout (useful for large result sets)")

	protheusCmd.AddCommand(protheusSchemaCmd)
	protheusSchemaCmd.Flags().StringVar(&protheusConnection, "connection", "", "Run against this named profile without switching the active connection")
}

var protheusSchemaCmd = &cobra.Command{
	Use:   "schema <table_name>",
	Short: "Get the schema (columns and types) for a specific table",
	Long: `Quickly discover the structure of a Protheus SQL Server table.
This helps AI agents formulate correct SQL queries without hallucinations.

OUTPUT SCHEMA:
  Returns a TOON or JSON response containing column names, data types, and max lengths.

EXAMPLE:
  mapj protheus schema SA1010
  mapj protheus schema SA1010 --connection TOTALPEC_PRD`,
	Args: cobra.ExactArgs(1),
	RunE: protheusSchemaRun,
}

func protheusSchemaRun(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Validate basic table name to prevent SQL injection in the schema query itself
	if strings.ContainsAny(tableName, " ;'\"-") {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "invalid table name format", false)
		fmt.Println(GetFormatter().Format(env))
		return errors.New("USAGE_ERROR")
	}

	sqlQuery := fmt.Sprintf(`
		SELECT 
			COLUMN_NAME, 
			DATA_TYPE, 
			CHARACTER_MAXIMUM_LENGTH
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION
	`, tableName)

	// Reuse the query logic but override the SQL
	protheusMaxRows = 1000 // Schema queries are small, but add a safety net
	protheusOutputFile = "" // Never write schema to file, always stdout
	
	// Temporarily override the args to use our generated SQL
	return protheusQueryRun(cmd, []string{sqlQuery})
}

func protheusQueryRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sqlQuery := args[0]
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	creds, err := store.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Resolve which profile to use
	var profile *auth.ProtheusProfile
	if protheusConnection != "" {
		// --connection flag: use specific profile without changing active
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[protheusConnection] == nil {
			msg := fmt.Sprintf("profile '%s' not found. Use 'mapj protheus connection list' to see available profiles", protheusConnection)
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND", msg, false)
			fmt.Println(formatter.Format(env))
			return errors.New(msg)
		}
		profile = creds.ProtheusProfiles[protheusConnection]
	} else {
		// Use active profile (with v1 migration)
		profile = creds.ActiveProtheusProfile()
	}

	if profile == nil {
		msg := "No Protheus connection configured. Run:\n  mapj protheus connection add <name> --server S --database D --user U --password P"
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", msg, false)
		fmt.Println(formatter.Format(env))
		return errors.New("NOT_AUTHENTICATED")
	}

	client := protheus.NewClient(profile.Server, profile.Port, profile.Database, profile.User, profile.Password)

	result, err := client.Query(ctx, sqlQuery, protheusMaxRows)
	if err != nil {
		if strings.Contains(err.Error(), "validation error") {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", err.Error(), false)
			fmt.Println(formatter.Format(env))
			return err
		}

		// Connection / network error — add VPN hint
		msg := err.Error()
		hint := protheusVPNHint(profile.Server)
		fullMsg := msg + "\n" + hint
		env := output.NewErrorEnvelope(cmd.CommandPath(), "QUERY_ERROR", fullMsg, true)
		fmt.Println(formatter.Format(env))
		return err
	}

	// ── Safety Tripwire (Protect LLM Context) ──────────────────────────────────
	// If the result is very large and no output file was specified, auto-fallback to file
	tripwireThreshold := 500
	if protheusOutputFile == "" && result.Count > tripwireThreshold {
		tempFile := fmt.Sprintf("mapj_overflow_%d.toon", time.Now().UnixNano())
		protheusOutputFile = tempFile
		
		// Let the user/agent know we diverted the output
		fmt.Fprintf(os.Stderr, "⚠️  Safety Tripwire: Result exceeded %d rows. Auto-saving to %s to protect context window.\n", tripwireThreshold, tempFile)
		
		// Force TOON format for the file since it's the most efficient
		formatter = output.NewFormatter("toon")
	}

	// ── Build output payload ──────────────────────────────────────────────────
	var resultPayload any
	var fileFormatter output.Formatter

	_, isToon := formatter.(output.TOONFormatter)
	_, isAuto := formatter.(output.AutoFormatter)
	if isToon || isAuto {
		resultPayload = buildToonPayload(result)
	} else {
		resultPayload = result
	}
	
	fileFormatter = formatter

	// ── --output-file: write to file, print summary to stdout ─────────────────
	if protheusOutputFile != "" {
		env := output.NewEnvelope(cmd.CommandPath(), resultPayload)
		content := fileFormatter.Format(env)

		if err := output.WriteToFile(protheusOutputFile, content); err != nil {
			errEnv := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(), "FILE_WRITE_ERROR", err.Error(),
				fmt.Sprintf("Check that the directory exists and you have write access: %s", protheusOutputFile),
				false,
			)
			fmt.Println(formatter.Format(errEnv))
			return err
		}

		// Print a minimal summary to stdout (not the data)
		format := "auto"
		if _, isLLM := formatter.(output.LLMFormatter); isLLM {
			format = "llm"
		}
		summary := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"rows":        result.Count,
			"columns":     len(result.Columns),
			"format":      format,
			"output_file": protheusOutputFile,
		})
		fmt.Println(formatter.Format(summary))
		return nil
	}

	// ── Default: print to stdout ──────────────────────────────────────────────
	env := output.NewEnvelope(cmd.CommandPath(), resultPayload)
	fmt.Println(fileFormatter.Format(env))
	return nil
}

// protheusVPNHint returns a contextual VPN hint based on the server IP range.
func protheusVPNHint(server string) string {
	switch {
	case strings.HasPrefix(server, "192.168.99."):
		return "💡 VPN: This is a TOTALPEC server. Verify the TOTALPEC VPN is active."
	case strings.HasPrefix(server, "192.168.7."):
		return "💡 VPN: This is a UNION server. Verify the UNION VPN is active."
	default:
		return fmt.Sprintf("💡 VPN: Verify the VPN for server %s is active.", server)
	}
}

// buildToonPayload converts a QueryResult into a slice of maps for tabular TOON output.
func buildToonPayload(result *protheus.QueryResult) []map[string]any {
	payload := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		obj := make(map[string]any, len(result.Columns))
		for i, col := range result.Columns {
			obj[col] = row[i]
		}
		payload = append(payload, obj)
	}
	return payload
}
