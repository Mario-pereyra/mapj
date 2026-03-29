package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for all services",
	Long: `Show the current authentication state for TDN, Confluence, and Protheus.
Run this first to check what needs to be configured.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj auth status","result":{
    "tdn":        {"authenticated":false},
    "confluence": {"authenticated":true, "url":"https://tdninterno.totvs.com"},
    "protheus":   {"authenticated":true,
                   "activeProfile":"TOTALPEC_BIB",
                   "server":"192.168.99.102",
                   "database":"P1212410_BIB",
                   "totalProfiles":7}
  }}

  authenticated:true  = credentials stored and present (not validated against server)
  authenticated:false = not configured, run 'mapj auth login <service>'
  totalProfiles       = number of registered Protheus connection profiles

See also:
  mapj protheus connection list   # detailed view of all Protheus profiles`,
	RunE:  statusRun,
}

// authStatusResult is the structured output for LLM consumption.
type authStatusResult struct {
	TDN        serviceStatus  `json:"tdn"`
	Confluence serviceStatus  `json:"confluence"`
	Protheus   protheusStatus `json:"protheus"`
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

	// Use the correct formatter based on global --output flag
	// Read from os.Args since we can't import cli (circular). Auth flag is always -o / --output.
	formatter := output.NewFormatter(outputFlagFromArgs())
	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}

// outputFlagFromArgs reads the -o / --output flag value from os.Args.
// This avoids a circular import between auth and cli packages.
func outputFlagFromArgs() string {
	args := os.Args
	for i, a := range args {
		if (a == "-o" || a == "--output") && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--output=") {
			return strings.TrimPrefix(a, "--output=")
		}
		if strings.HasPrefix(a, "-o=") {
			return strings.TrimPrefix(a, "-o=")
		}
	}
	return "llm" // default
}
