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
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		return err
	}

	fmt.Println("Authentication Status:")
	fmt.Printf("  TDN:        %s\n", boolStr(creds.TDN != nil && creds.TDN.Token != ""))
	fmt.Printf("  Confluence: %s\n", boolStr(creds.Confluence != nil && creds.Confluence.Token != ""))
	fmt.Printf("  Protheus:   %s\n", boolStr(creds.Protheus != nil && creds.Protheus.Server != ""))

	return nil
}

func boolStr(b bool) string {
	if b {
		return "✓ authenticated"
	}
	return "✗ not configured"
}
