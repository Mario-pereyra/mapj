package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

var advplMonitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor connected users and server state",
	Long: `Monitor TOTVS Application Server connections and users.

SUBCOMMANDS:
  monitor users    List connected users
  monitor kill     Terminate a user connection
  monitor lock     Lock server (prevent new connections)
  monitor unlock   Unlock server (allow new connections)`,
}

var monitorConnection string

// ---- USERS ----

var monitorUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List connected users on the AppServer",
	Long: `List all currently connected users on the TOTVS Application Server.

OUTPUT SCHEMA:
  {"ok":true,"result":{"total":15,"users":[
    {"username":"admin","computerName":"SRV01","threadId":12345,
     "environment":"producao","mainName":"SIGAFAT","loginTime":"08:00:00"}
  ]}}

EXAMPLES:
  mapj advpl monitor users
  mapj advpl monitor users --connection PRODUCAO`,
	RunE: monitorUsersRun,
}

func monitorUsersRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, monitorConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Fetching connected users...\n")
		users, err := client.GetUsers(context.Background())
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "MONITOR_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"total": len(users),
			"users": users,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- KILL ----

var monitorKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Terminate a user connection",
	Long: `Terminate a specific user connection on the AppServer by thread ID.

OUTPUT SCHEMA:
  {"ok":true,"result":{"threadId":12345,"message":"User disconnected"}}

EXAMPLES:
  mapj advpl monitor kill --thread-id 12345 --username admin --computer SRV01`,
	RunE: monitorKillRun,
}

var (
	killThreadID int
	killUsername string
	killComputer string
	killServer   string
)

func monitorKillRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if killThreadID == 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --thread-id", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	return withTDSClient(cmd, monitorConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Killing thread %d...\n", killThreadID)
		message, err := client.KillUser(context.Background(), killUsername, killComputer, killThreadID, killServer)
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "MONITOR_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"threadId": killThreadID,
			"username": killUsername,
			"message":  message,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- LOCK ----

var monitorLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock server (prevent new connections)",
	Long: `Lock the TOTVS Application Server to prevent new user connections.

OUTPUT SCHEMA:
  {"ok":true,"result":{"locked":true}}`,
	RunE: monitorLockRun,
}

func monitorLockRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, monitorConnection, func(client *tds.Client) error {
		if err := client.SetConnectionStatus(context.Background(), false); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "MONITOR_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"locked": true,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- UNLOCK ----

var monitorUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock server (allow new connections)",
	Long: `Unlock the TOTVS Application Server to allow new user connections.

OUTPUT SCHEMA:
  {"ok":true,"result":{"locked":false}}`,
	RunE: monitorUnlockRun,
}

func monitorUnlockRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, monitorConnection, func(client *tds.Client) error {
		if err := client.SetConnectionStatus(context.Background(), true); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "MONITOR_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"locked": false,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ======================== INIT ========================

func init() {
	advplCmd.AddCommand(advplMonitorCmd)
	advplMonitorCmd.AddCommand(
		monitorUsersCmd,
		monitorKillCmd,
		monitorLockCmd,
		monitorUnlockCmd,
	)

	// Shared flags
	advplMonitorCmd.PersistentFlags().StringVar(&monitorConnection, "connection", "", "Use specific profile without switching active")

	// Kill flags
	monitorKillCmd.Flags().IntVar(&killThreadID, "thread-id", 0, "Thread ID to kill")
	monitorKillCmd.Flags().StringVar(&killUsername, "username", "", "Username of the connection")
	monitorKillCmd.Flags().StringVar(&killComputer, "computer", "", "Computer name of the connection")
	monitorKillCmd.Flags().StringVar(&killServer, "server", "", "Server name")
	monitorKillCmd.MarkFlagRequired("thread-id")
}
