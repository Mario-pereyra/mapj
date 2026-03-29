package auth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to a service",
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var tdnLoginCmd = &cobra.Command{
	Use:   "tdn --url URL --token TOKEN",
	Short: "Login to TDN (TOTVS Developer Network)",
	RunE:  tdnLogin,
}
var tdnURL, tdnToken string

var confluenceLoginCmd = &cobra.Command{
	Use:   "confluence --url URL --token TOKEN",
	Short: "Login to Confluence",
	Long: `Login to Confluence.

For Confluence Server / Data Center (e.g. tdninterno.totvs.com):
  Use Bearer auth (PAT token). Do NOT set --username.

  mapj auth login confluence --url https://tdninterno.totvs.com --token YOUR_PAT

For Confluence Cloud (e.g. company.atlassian.net):
  Use Basic auth (email + API token).

  mapj auth login confluence --url https://company.atlassian.net --username you@company.com --token YOUR_API_TOKEN

The auth type is auto-detected from the URL. Override with --auth-type bearer|basic.`,
	RunE: confluenceLogin,
}
var confluenceURL, confluenceToken, confluenceUsername, confluenceAuthType string

var protheusLoginCmd = &cobra.Command{
	Use:   "protheus --server S --port P --database D --user U --password PASS",
	Short: "Login to Protheus database",
	RunE:  protheusLogin,
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

	fmt.Println(authJSON(cmd.CommandPath(), "tdn", ""))
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

	fmt.Println(authJSON(cmd.CommandPath(), "confluence", authType))
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

	fmt.Println(authJSON(cmd.CommandPath(), "protheus", ""))
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
// Cloud uses Basic Auth (email + API token). Everything else (Server, DC, intranet) uses Bearer PAT.
func isCloudURL(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), "atlassian.net")
}

// authJSON produces a compact JSON response for auth operations.
func authJSON(cmdPath, service, authType string) string {
	payload := map[string]any{
		"ok":            true,
		"command":       cmdPath,
		"result": map[string]any{
			"service":       service,
			"authenticated": true,
		},
	}
	if authType != "" {
		payload["result"].(map[string]any)["authType"] = authType
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
