package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
	"github.com/spf13/cobra"
)

var protheusCmd = &cobra.Command{
	Use:   "protheus",
	Short: "Protheus database commands",
}

var protheusQueryCmd = &cobra.Command{
	Use:   "query <sql>",
	Short: "Execute SELECT query on Protheus database",
	Long: `Execute a SELECT query on the Protheus SQL Server database.

Examples:
  mapj protheus query "SELECT TOP 10 * FROM SPED050"
  mapj protheus query "SELECT COUNT(*) FROM SA1010" --format csv
  mapj protheus query "SELECT TOP 5 A1_COD FROM SA1010" --connection TOTALPEC_PRD

Note: Only SELECT queries are allowed for security reasons.`,
	Args: cobra.ExactArgs(1),
	RunE: protheusQueryRun,
}

var protheusFormat string
var protheusMaxRows int
var protheusConnection string // --connection: run against specific profile without switching

func init() {
	protheusCmd.AddCommand(protheusQueryCmd)
	protheusQueryCmd.Flags().StringVar(&protheusFormat, "format", "json", "Output format (json, csv)")
	protheusQueryCmd.Flags().IntVar(&protheusMaxRows, "max-rows", 10000, "Max rows to return (0 = no limit)")
	protheusQueryCmd.Flags().StringVar(&protheusConnection, "connection", "", "Run against this named profile without switching the active connection")
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

	result, err := client.Query(ctx, sqlQuery)
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

	if protheusMaxRows > 0 && result.Count > protheusMaxRows {
		result.Rows = result.Rows[:protheusMaxRows]
		result.Count = protheusMaxRows
	}

	if protheusFormat == "csv" {
		csvOutput := protheusResultToCSV(result)
		env := output.NewEnvelope(cmd.CommandPath(), map[string]interface{}{
			"format":  "csv",
			"content": csvOutput,
		})
		fmt.Println(formatter.Format(env))
		return nil
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}

func protheusResultToCSV(result *protheus.QueryResult) string {
	var lines []string

	lines = append(lines, strings.Join(result.Columns, ","))

	for _, row := range result.Rows {
		var fields []string
		for _, f := range row {
			fields = append(fields, fmt.Sprintf("%v", f))
		}
		lines = append(lines, strings.Join(fields, ","))
	}

	return strings.Join(lines, "\n")
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
