package auth

import (
	"encoding/json"
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
		b, _ := json.Marshal(map[string]any{
			"ok":      false,
			"command": cmd.CommandPath(),
			"error": map[string]any{
				"code":    "INVALID_SERVICE",
				"message": "unknown service: " + service,
				"hint":    "Valid services: tdn, confluence, protheus",
			},
		})
		fmt.Println(string(b))
		return fmt.Errorf("unknown service: %s", service)
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	b, _ := json.Marshal(map[string]any{
		"ok":      true,
		"command": cmd.CommandPath(),
		"result": map[string]any{
			"service":       service,
			"authenticated": false,
		},
	})
	fmt.Println(string(b))
	return nil
}
