package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

// Health command to verify connectivity to all configured services

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check connectivity to all configured services",
	Long: `Verify connectivity to TDN, Confluence, Protheus SQL Server, and TDS AppServer.

Returns health status for each service, including latency where available.

Exit codes:
  0 = all healthy
  1 = general error
  2 = usage error
  3 = auth error
  4 = retryable error

EXAMPLES:
  mapj health                    # check all services
  mapj health --service=tdn     # check single service
  mapj health --service=protheus
  mapj health --service=tds`,
	RunE: healthRun,
}

var healthService string

func init() {
	rootCmd.AddCommand(healthCmd)
	healthCmd.Flags().StringVar(&healthService, "service", "", "Service to check: tdn, confluence, protheus, tds (default: all)")
}

// healthResult is the structured output for health check results.
type healthResult struct {
	Services map[string]serviceHealth `json:"services"`
	AllHealthy bool `json:"allHealthy"`
}

type serviceHealth struct {
	Healthy       bool   `json:"healthy"`
	Authenticated bool   `json:"authenticated,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
	Server        string `json:"server,omitempty"`
	Database      string `json:"database,omitempty"`
	Build         string `json:"build,omitempty"`
	Secure        bool   `json:"secure,omitempty"`
	Error         string `json:"error,omitempty"`
	Hint          string `json:"hint,omitempty"`
}

func healthRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	result := healthResult{
		Services: make(map[string]serviceHealth),
	}

	// Determine which services to check
	services := []string{"tdn", "confluence", "protheus", "tds"}
	if healthService != "" {
		services = []string{healthService}
	}

	allHealthy := true

	for _, svc := range services {
		var health serviceHealth
		var err error

		switch svc {
		case "tdn":
			health, err = checkTDN()
		case "confluence":
			health, err = checkConfluence()
		case "protheus":
			health, err = checkProtheus()
		case "tds":
			health, err = checkTDS()
		default:
			env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR",
				fmt.Sprintf("unknown service: %s. Valid services: tdn, confluence, protheus, tds", svc), false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("unknown service: %s", svc)
		}

		if err != nil {
			allHealthy = false
		}
		result.Services[svc] = health
	}

	result.AllHealthy = allHealthy

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))

	if !allHealthy {
		return fmt.Errorf("one or more services unhealthy")
	}
	return nil
}

// checkTDN verifies connectivity to TDN (public API, no auth required).
func checkTDN() (serviceHealth, error) {
	health := serviceHealth{Healthy: false}

	store, err := auth.NewStore()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	creds, err := store.Load()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	// Check if TDN credentials are configured
	if creds.TDN != nil && creds.TDN.Token != "" {
		health.Authenticated = true
	} else {
		health.Authenticated = false
	}

	// Try to ping TDN (public endpoint works without auth)
	baseURL := "https://tdn.totvs.com"
	if creds.TDN != nil && creds.TDN.BaseURL != "" {
		baseURL = creds.TDN.BaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	client := confluence.NewClient(baseURL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err = client.Ping(ctx)
	health.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		health.Healthy = false
		health.Error = err.Error()
		return health, err
	}

	health.Healthy = true
	return health, nil
}

// checkConfluence verifies connectivity to Confluence.
func checkConfluence() (serviceHealth, error) {
	health := serviceHealth{Healthy: false}

	store, err := auth.NewStore()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	creds, err := store.Load()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	// Check if Confluence credentials are configured
	if creds.Confluence != nil && creds.Confluence.Token != "" {
		health.Authenticated = true
	} else {
		health.Authenticated = false
		health.Healthy = false
		health.Error = "not configured"
		health.Hint = "Run: mapj auth login confluence --url URL --token TOKEN"
		return health, fmt.Errorf("confluence not configured")
	}

	client := confluence.NewClient(creds.Confluence.BaseURL, creds.Confluence.Token)
	if creds.Confluence.AuthType == "basic" {
		client.SetBasicAuth(creds.Confluence.Username, creds.Confluence.Token)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err = client.Ping(ctx)
	health.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		health.Healthy = false
		health.Error = err.Error()
		return health, err
	}

	health.Healthy = true
	return health, nil
}

// checkProtheus verifies connectivity to Protheus SQL Server.
func checkProtheus() (serviceHealth, error) {
	health := serviceHealth{Healthy: false}

	store, err := auth.NewStore()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	creds, err := store.Load()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	// Check if Protheus profile is configured
	profile := creds.ActiveProtheusProfile()
	if profile == nil {
		health.Healthy = false
		health.Error = "no active profile"
		health.Hint = "Run: mapj protheus connection add <name> --server ... --database ... --user ... --password ... --use"
		return health, fmt.Errorf("no active Protheus profile")
	}

	health.Server = profile.Server
	health.Database = profile.Database
	health.Authenticated = true

	client := protheus.NewClient(profile.Server, profile.Port, profile.Database, profile.User, profile.Password)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	latency, err := client.Ping(ctx)
	health.LatencyMs = latency

	if err != nil {
		health.Healthy = false
		health.Error = err.Error()
		health.Hint = protheusVPNHint(profile.Server)
		return health, err
	}

	health.Healthy = true
	return health, nil
}

// checkTDS verifies connectivity to TDS/AppServer.
func checkTDS() (serviceHealth, error) {
	health := serviceHealth{Healthy: false}

	store, err := auth.NewStore()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	creds, err := store.Load()
	if err != nil {
		health.Error = err.Error()
		return health, err
	}

	// Check if TDS profile is configured
	profile := creds.ActiveTDSProfile()
	if profile == nil {
		health.Healthy = false
		health.Error = "no active profile"
		health.Hint = "Run: mapj advpl connection add <name> --server ... --port ... --environment ... --user ... --password ... --use"
		return health, fmt.Errorf("no active TDS profile")
	}

	health.Server = profile.Server
	health.Authenticated = true

	// Find advpls binary
	advplsPath, err := tds.FindAdvplsBinary()
	if err != nil {
		health.Healthy = false
		health.Error = err.Error()
		health.Hint = "Install advpls: npm install -g @totvs/tds-ls, or place the binary in PATH"
		return health, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		health.Healthy = false
		health.Error = fmt.Sprintf("failed to start advpls: %s", err.Error())
		health.Hint = "Verify advpls binary is valid and executable"
		return health, err
	}
	defer client.Close()

	valResult, err := client.Validate(ctx, profile.Server, profile.Port)
	health.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		health.Healthy = false
		health.Error = fmt.Sprintf("connection to %s:%d failed: %s", profile.Server, profile.Port, err.Error())
		health.Hint = tdsVPNHint(profile.Server)
		return health, err
	}

	health.Healthy = true
	health.Build = valResult.Build
	health.Secure = valResult.Secure != 0

	return health, nil
}

// tdsVPNHint returns a contextual VPN hint based on the server IP range.
func tdsVPNHint(server string) string {
	switch {
	case strings.HasPrefix(server, "192.168.99."):
		return "💡 VPN: This is a TOTALPEC server. Verify the TOTALPEC VPN is active."
	case strings.HasPrefix(server, "192.168.7."):
		return "💡 VPN: This is a UNION server. Verify the UNION VPN is active."
	default:
		return fmt.Sprintf("💡 VPN: Verify the VPN for server %s is active.", server)
	}
}
