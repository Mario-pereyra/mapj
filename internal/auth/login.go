package auth

import (
	"fmt"

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
	Long: `Login to Confluence Cloud with either:
  1. Basic Auth: --username EMAIL --password API_TOKEN
  2. Bearer Token: --token PAT (Personal Access Token)`,
	RunE: confluenceLogin,
}
var confluenceURL, confluenceToken, confluenceUsername string

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
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		return err
	}
	creds.TDN = &TDNCreds{BaseURL: tdnURL, Token: tdnToken}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Println("TDN login successful")
	return nil
}

func confluenceLogin(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		return err
	}
	creds.Confluence = &ConfluenceCreds{
		BaseURL:  confluenceURL,
		Username: confluenceUsername,
		Token:    confluenceToken,
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	fmt.Println("Confluence login successful")
	return nil
}

func protheusLogin(cmd *cobra.Command, args []string) error {
	store, err := NewStore()
	if err != nil {
		return err
	}
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

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

	fmt.Println("Protheus login successful")
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
	confluenceLoginCmd.Flags().StringVar(&confluenceUsername, "username", "", "Confluence username (email for Cloud API token auth)")
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
