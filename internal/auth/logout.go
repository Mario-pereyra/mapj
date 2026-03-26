package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Logout from a service",
	Args:  cobra.ExactArgs(1),
	RunE:  logoutRun,
}

func logoutRun(cmd *cobra.Command, args []string) error {
	service := args[0]

	store, err := NewStore()
	if err != nil {
		return err
	}
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		return err
	}

	switch service {
	case "tdn":
		creds.TDN = nil
	case "confluence":
		creds.Confluence = nil
	case "protheus":
		creds.Protheus = nil
	default:
		return fmt.Errorf("unknown service: %s", service)
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Printf("Logged out from %s\n", service)
	return nil
}
