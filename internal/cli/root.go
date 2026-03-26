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
	noColor      bool
)

var rootCmd = &cobra.Command{
	Use:   "mapj",
	Short: "CLI for LLM agents to search TDN, export Confluence, and query Protheus",
	Long: `mapj is an agentic CLI tool designed for LLM/AI agents.

Commands:
  mapj tdn search <query>       Search TDN documentation
  mapj confluence export <url>  Export Confluence page
  mapj protheus query <sql>     Query Protheus database
  mapj auth login <service>     Authenticate a service
  mapj auth status              Show auth status`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tdnCmd, confluenceCmd, protheusCmd)
}

func Execute() int {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json, table)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return errors.ExitError
	}
	return errors.ExitSuccess
}

func GetFormatter() output.Formatter {
	return output.NewFormatter(outputFormat)
}
