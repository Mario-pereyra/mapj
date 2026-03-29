package auth

import (
	"fmt"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Remove stored credentials for a service",
	Long: `Remove stored credentials for a service. Does not affect the server.
Valid services: tdn, confluence, protheus

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj auth logout","result":{"service":"confluence","authenticated":false}}

EXAMPLE:
  mapj auth logout confluence
  mapj auth logout protheus

Note: For Protheus, this only removes the legacy v1 credential.
To remove named profiles use: mapj protheus connection remove <name>`,
	Args:  cobra.ExactArgs(1),
	RunE:  logoutRun,
}

func logoutRun(cmd *cobra.Command, args []string) error {
	service := args[0]
	formatter := output.NewFormatter(outputFlagFromArgs())

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
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "INVALID_SERVICE",
			"unknown service: "+service,
			"Valid services: tdn, confluence, protheus",
			false,
		)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("unknown service: %s", service)
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"service":       service,
		"authenticated": false,
	})
	fmt.Println(formatter.Format(env))
	return nil
}
