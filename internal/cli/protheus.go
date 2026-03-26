package cli

import (
	"context"
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

Note: Only SELECT queries are allowed for security reasons.`,
	Args: cobra.ExactArgs(1),
	RunE: protheusQueryRun,
}

var protheusFormat string

func init() {
	protheusCmd.AddCommand(protheusQueryCmd)
	protheusQueryCmd.Flags().StringVar(&protheusFormat, "format", "json", "Output format (json, csv)")
}

func protheusQueryRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sqlQuery := args[0]
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

	if creds.Protheus == nil || creds.Protheus.Server == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", "Run 'mapj auth login protheus --server S --database D --user U --password P' first", false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	client := protheus.NewClient(
		creds.Protheus.Server,
		creds.Protheus.Port,
		creds.Protheus.Database,
		creds.Protheus.User,
		creds.Protheus.Password,
	)

	result, err := client.Query(ctx, sqlQuery)
	if err != nil {
		if strings.Contains(err.Error(), "validation error") {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", err.Error(), false)
			fmt.Println(formatter.Format(env))
			return nil
		}
		env := output.NewErrorEnvelope(cmd.CommandPath(), "QUERY_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return nil
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
