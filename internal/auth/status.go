package auth

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for all services",
	RunE:  statusRun,
}

// authStatusResult is the structured output for LLM consumption.
type authStatusResult struct {
	TDN        serviceStatus    `json:"tdn"`
	Confluence serviceStatus    `json:"confluence"`
	Protheus   protheusStatus   `json:"protheus"`
}

type serviceStatus struct {
	Authenticated bool   `json:"authenticated"`
	URL           string `json:"url,omitempty"`
}

type protheusStatus struct {
	Authenticated bool   `json:"authenticated"`
	ActiveProfile string `json:"activeProfile,omitempty"`
	Server        string `json:"server,omitempty"`
	Database      string `json:"database,omitempty"`
	TotalProfiles int    `json:"totalProfiles,omitempty"`
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

	result := authStatusResult{}

	// TDN
	if creds.TDN != nil && creds.TDN.Token != "" {
		result.TDN = serviceStatus{Authenticated: true, URL: creds.TDN.BaseURL}
	}

	// Confluence
	if creds.Confluence != nil && creds.Confluence.Token != "" {
		result.Confluence = serviceStatus{Authenticated: true, URL: creds.Confluence.BaseURL}
	}

	// Protheus
	if creds.HasProtheusProfiles() {
		active := creds.ActiveProtheusProfile()
		total := len(creds.ProtheusProfiles)
		ps := protheusStatus{Authenticated: true, TotalProfiles: total}

		if creds.Protheus != nil && total == 0 {
			// Legacy v1
			ps.ActiveProfile = "default (legacy)"
			ps.Server = creds.Protheus.Server
			ps.Database = creds.Protheus.Database
		} else if active != nil {
			ps.ActiveProfile = active.Name
			ps.Server = active.Server
			ps.Database = active.Database
		}
		result.Protheus = ps
	}

	// Output: use structured JSON (respects --output flag via shared approach)
	// auth package doesn't get the global formatter from cli package (circular import),
	// so we write compact JSON directly — consistent with LLM mode.
	b, _ := json.Marshal(map[string]any{
		"ok":      true,
		"command": cmd.CommandPath(),
		"result":  result,
	})
	fmt.Println(string(b))
	return nil
}
