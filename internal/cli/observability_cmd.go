package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// observabilityCmd is the parent command for observability features.
var observabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "Observability features: metrics, tracing",
	Long: `Commands for observing mapj CLI behavior.

EXAMPLES:
  mapj observability metrics    # Show command metrics in Prometheus format`,
}

// metricsCmd shows the current metrics in Prometheus text format.
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show command metrics",
	Long: `Display collected command metrics in Prometheus text format.

Shows:
  - mapj_command_total{cmd,exit_code}: Counter of commands by name and exit code
  - mapj_command_duration_ms{cmd}: Histogram of command durations by command name

EXAMPLES:
  mapj observability metrics`,
	Run: func(cmd *cobra.Command, args []string) {
		output := GetAllMetricsPrometheus()
		fmt.Println(output)
	},
}

func init() {
	rootCmd.AddCommand(observabilityCmd)
	observabilityCmd.AddCommand(metricsCmd)
}
