package auth

import (
	"fmt"
	"strings"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to a service and store credentials",
	Long: `Authenticate to a service. Credentials stored encrypted at ~/.config/mapj/credentials.enc.

SERVICES:
  tdn         Public TDN (optional — public content works without auth)
  confluence  Confluence Server/DC or Cloud (required for export)
  protheus    Protheus SQL Server (legacy — prefer: mapj protheus connection add)

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj auth login <service>","result":{
    "service":"confluence","authenticated":true,"authType":"bearer"
  }}

Run 'mapj auth login <service> --help' for service-specific flags.`,
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication for TDN, Confluence, and Protheus",
	Long: `Manage credentials for all services used by mapj.

Subcommands:
  mapj auth status              Show auth state for all services (run this first)
  mapj auth login <service>     Authenticate a service
  mapj auth logout <service>    Remove stored credentials

SERVICES:
  tdn         tdn.totvs.com   — public docs, auth optional
  confluence  Confluence Server/DC or Cloud  — required for export
  protheus    SQL Server  — use 'mapj protheus connection' for multi-profile management

Run 'mapj auth status' first to see what is already authenticated.`,
}

var tdnLoginCmd = &cobra.Command{
	Use:   "tdn --url URL --token TOKEN",
	Short: "Login to TDN (optional — public content works without auth)",
	Long: `Authenticate to TDN (tdn.totvs.com). This is OPTIONAL.
Public TDN content is accessible without authentication.
Only needed for private/internal TDN instances.

  mapj auth login tdn --url https://tdn.totvs.com --token YOUR_PAT

OUTPUT SCHEMA:
  {"ok":true,"result":{"service":"tdn","authenticated":true}}`,
	RunE: tdnLogin,
}
var tdnURL, tdnToken string

var confluenceLoginCmd = &cobra.Command{
	Use:   "confluence --url URL --token TOKEN",
	Short: "Authenticate Confluence (required before export)",
	Long: `Authenticate to Confluence. Required before running 'mapj confluence export'.

AUTH TYPE AUTO-DETECTION from URL:
  *.atlassian.net  → Basic Auth (email + API token)
  anything else    → Bearer PAT (Server/DC)

For Confluence Server / Data Center (e.g. tdninterno.totvs.com):
  mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_PAT
  ⚠️  Do NOT use --username for Server/DC — causes 401.

For Confluence Cloud (e.g. company.atlassian.net):
  mapj auth login confluence --url https://company.atlassian.net --username you@example.com --token YOUR_API_TOKEN

OUTPUT SCHEMA:
  {"ok":true,"result":{"service":"confluence","authenticated":true,"authType":"bearer"}}`,
	RunE: confluenceLogin,
}
var confluenceURL, confluenceToken, confluenceUsername, confluenceAuthType string

var protheusLoginCmd = &cobra.Command{
	Use:   "protheus --server S --port P --database D --user U --password PASS",
	Short: "Login to Protheus (legacy — prefer: mapj protheus connection add)",
	Long: `DEPRECATED in favor of multi-profile connection management.
This command registers a single legacy connection (no profile name).

Preferred: Use 'mapj protheus connection add' for named, switchable profiles.
  mapj protheus connection add MYDB --server HOST --database DB --user U --password P --use

OUTPUT SCHEMA:
  {"ok":true,"result":{"service":"protheus","authenticated":true}}`,
	RunE: protheusLogin,
}
var protheusServer, protheusDatabase, protheusUser, protheusPassword string
var protheusPort int

func tdnLogin(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}

	creds, err := store.Load()
	if err != nil {
		return err
	}
	creds.TDN = &TDNCreds{BaseURL: tdnURL, Token: tdnToken}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Println(loginJSON(cmd.CommandPath(), "tdn", ""))
	return nil
}

func confluenceLogin(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}

	creds, err := store.Load()
	if err != nil {
		return err
	}

	// Auto-detect auth type based on URL if not explicitly set
	authType := confluenceAuthType
	if authType == "" {
		if isCloudURL(confluenceURL) {
			authType = "basic"
		} else {
			authType = "bearer"
		}
	}

	// Validate: Cloud Basic Auth needs a username
	if authType == "basic" && confluenceUsername == "" {
		return fmt.Errorf("--username is required for basic auth (Confluence Cloud). Use: --username your@email.com")
	}

	// Warn: Server Bearer Auth with username is a common mistake
	if authType == "bearer" && confluenceUsername != "" {
		// Still log the warning to stderr so it doesn't pollute the JSON output
		fmt.Fprintf(cmd.OutOrStderr(), `{"level":"warn","message":"--username is ignored for bearer auth (Server/DC). If you need Basic Auth use --auth-type basic"}%s`, "\n")
		confluenceUsername = ""
	}

	creds.Confluence = &ConfluenceCreds{
		BaseURL:  confluenceURL,
		Username: confluenceUsername,
		Token:    confluenceToken,
		AuthType: authType,
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Println(loginJSON(cmd.CommandPath(), "confluence", authType))
	return nil
}

func protheusLogin(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}

	creds, err := store.Load()
	if err != nil {
		return err
	}
	creds.Protheus = &ProtheusCreds{
		Server:   protheusServer,
		Port:     protheusPort,
		Database: protheusDatabase,
		User:     protheusUser,
		Password: protheusPassword,
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Println(loginJSON(cmd.CommandPath(), "protheus", ""))
	return nil
}

func AddCommands(root *cobra.Command) {
	loginCmd.AddCommand(tdnLoginCmd, confluenceLoginCmd, protheusLoginCmd)
	authCmd.AddCommand(loginCmd, statusCmd, logoutCmd)
	root.AddCommand(authCmd)

	tdnLoginCmd.Flags().StringVar(&tdnURL, "url", "https://tdninterno.totvs.com", "TDN base URL")
	tdnLoginCmd.Flags().StringVar(&tdnToken, "token", "", "TDN Personal Access Token")
	tdnLoginCmd.MarkFlagRequired("token")

	confluenceLoginCmd.Flags().StringVar(&confluenceURL, "url", "", "Confluence base URL")
	confluenceLoginCmd.Flags().StringVar(&confluenceToken, "token", "", "Confluence API Token or PAT")
	confluenceLoginCmd.Flags().StringVar(&confluenceUsername, "username", "", "Email for Confluence Cloud Basic Auth (not needed for Server/DC Bearer auth)")
	confluenceLoginCmd.Flags().StringVar(&confluenceAuthType, "auth-type", "", "Auth scheme: 'bearer' (Server/DC PAT) or 'basic' (Cloud email+token). Auto-detected if omitted")
	confluenceLoginCmd.MarkFlagRequired("url")
	confluenceLoginCmd.MarkFlagRequired("token")

	protheusLoginCmd.Flags().StringVar(&protheusServer, "server", "", "Protheus server")
	protheusLoginCmd.Flags().IntVar(&protheusPort, "port", 1433, "SQL Server port")
	protheusLoginCmd.Flags().StringVar(&protheusDatabase, "database", "", "Database name")
	protheusLoginCmd.Flags().StringVar(&protheusUser, "user", "", "Database user")
	protheusLoginCmd.Flags().StringVar(&protheusPassword, "password", "", "Database password")
	protheusLoginCmd.MarkFlagRequired("server")
	protheusLoginCmd.MarkFlagRequired("database")
	protheusLoginCmd.MarkFlagRequired("user")
	protheusLoginCmd.MarkFlagRequired("password")
}

// isCloudURL returns true for Confluence Cloud (Atlassian-hosted) URLs.
func isCloudURL(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), "atlassian.net")
}

// loginJSON produces a formatted auth operation response using the global --output flag.
func loginJSON(cmdPath, service, authType string) string {
	result := map[string]any{
		"service":       service,
		"authenticated": true,
	}
	if authType != "" {
		result["authType"] = authType
	}
	formatter := output.NewFormatter(outputFlagFromArgs())
	env := output.NewEnvelope(cmdPath, result)
	return formatter.Format(env)
}
