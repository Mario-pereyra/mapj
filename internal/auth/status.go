package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for all services",
	RunE:  statusRun,
}

func statusRun(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}

	creds, err := store.Load()
	if err != nil {
		return err
	}

	fmt.Println("Authentication Status:")
	fmt.Printf("  TDN:        %s\n", boolStr(creds.TDN != nil && creds.TDN.Token != ""))
	fmt.Printf("  Confluence: %s\n", boolStr(creds.Confluence != nil && creds.Confluence.Token != ""))

	// Protheus: show active profile name and total count
	if creds.HasProtheusProfiles() {
		active := creds.ActiveProtheusProfile()
		total := len(creds.ProtheusProfiles)
		if creds.Protheus != nil && total == 0 {
			// Legacy v1 — show as "default (legacy)"
			fmt.Printf("  Protheus:   ✓ authenticated  [active: default (legacy) → %s/%s]\n",
				creds.Protheus.Server, creds.Protheus.Database)
		} else if active != nil {
			fmt.Printf("  Protheus:   ✓ authenticated  [active: %s → %s/%s | %d profile(s) registered]\n",
				active.Name, active.Server, active.Database, total)
		} else {
			fmt.Printf("  Protheus:   ✓ authenticated  [%d profile(s) registered, no active set]\n", total)
		}
	} else {
		fmt.Println("  Protheus:   ✗ not configured")
	}

	return nil
}

func boolStr(b bool) string {
	if b {
		return "✓ authenticated"
	}
	return "✗ not configured"
}
