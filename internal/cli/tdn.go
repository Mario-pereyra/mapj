package cli

import (
	"context"
	"fmt"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/confluence"
	"github.com/spf13/cobra"
)

var tdnCmd = &cobra.Command{
	Use:   "tdn",
	Short: "TDN (TOTVS Developer Network) commands",
}

var tdnSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search TDN documentation",
	Long: `Search the TOTVS Developer Network (TDN) for documentation.

Examples:
  mapj tdn search "REST API authentication"
  mapj tdn search "invoice" --space PROT --limit 5`,
	Args: cobra.ExactArgs(1),
	RunE: tdnSearchRun,
}

var tdnSpace string
var tdnLimit int

func init() {
	tdnCmd.AddCommand(tdnSearchCmd)
	tdnSearchCmd.Flags().StringVar(&tdnSpace, "space", "", "Filter by space key")
	tdnSearchCmd.Flags().IntVar(&tdnLimit, "limit", 10, "Max results")
}

func tdnSearchRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	query := args[0]
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil
	}
	store.SetKey("mapj-cred-key-32bytes-padded!!!!")

	creds, err := store.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	if creds.TDN == nil || creds.TDN.Token == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED", "Run 'mapj auth login tdn --token TOKEN' first", false)
		fmt.Println(formatter.Format(env))
		return nil
	}

	baseURL := creds.TDN.BaseURL
	if baseURL == "" {
		baseURL = "https://tdninterno.totvs.com"
	}
	client := confluence.NewClient(baseURL, creds.TDN.Token)
	if creds.TDN.Username != "" {
		client.SetBasicAuth(creds.TDN.Username, creds.TDN.Token)
	}

	opts := &confluence.SearchOpts{
		Query: query,
		Space: tdnSpace,
		Limit: tdnLimit,
	}

	result, err := client.Search(ctx, opts)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "SEARCH_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return nil
	}

	env := output.NewEnvelope(cmd.CommandPath(), result)
	fmt.Println(formatter.Format(env))
	return nil
}
