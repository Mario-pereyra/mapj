package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
	"github.com/spf13/cobra"
)

// ======================== CONNECTION SUBCOMMAND ========================

var protheusConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Manage named Protheus SQL Server connection profiles",
	Long: `Manage named, encrypted connection profiles for Protheus SQL Server.

PROFILE STORAGE: encrypted at ~/.config/mapj/credentials.enc
ACTIVE PROFILE: queries use the active profile by default.

SUBCOMMANDS:
  connection add <name>    Register new profile (--server --database --user --password --use)
  connection list          List all profiles (JSON with active field)
  connection use <name>    Switch active profile
  connection ping [name]   Test connectivity (returns latencyMs)
  connection show [name]   Show profile details (password masked)
  connection remove <name> Delete profile

Run 'mapj protheus connection <cmd> --help' for output schema.`,
}

// ---- ADD ----

var connAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Register a named Protheus connection profile",
	Long: `Register a named Protheus SQL Server connection profile.
The profile is stored encrypted. Use 'connection use <name>' to switch to it.

Examples:
  mapj protheus connection add TOTALPEC_BIB --server 192.168.99.102 --database P1212410_BIB --user P1212410_BIB --password P1212410_BIB
  mapj protheus connection add UNION_PRD --server 192.168.7.215 --database P1212410_PRD --user P1212410_PRD --password P1212410_PRD --use`,
	Args: cobra.ExactArgs(1),
	RunE: connAddRun,
}

var (
	connAddServer   string
	connAddPort     int
	connAddDatabase string
	connAddUser     string
	connAddPassword string
	connAddUse      bool
)

func connAddRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	formatter := GetFormatter()

	if name == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "profile name cannot be empty", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("profile name cannot be empty")
	}

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	profile := &auth.ProtheusProfile{
		Name:     name,
		Server:   connAddServer,
		Port:     connAddPort,
		Database: connAddDatabase,
		User:     connAddUser,
		Password: connAddPassword,
	}

	isFirst := !creds.HasProtheusProfiles()
	setActive := connAddUse || isFirst
	creds.SetProtheusProfile(profile, setActive)

	if err := store.Save(creds); err != nil {
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"name":      name,
		"server":    connAddServer,
		"port":      connAddPort,
		"database":  connAddDatabase,
		"setActive": setActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- LIST ----

var connListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered Protheus profiles (shows which is active)",
	Long: `List all registered Protheus connection profiles.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj protheus connection list","result":{
    "profiles": [
      {"name":"TOTALPEC_BIB","server":"192.168.99.102","port":1433,
       "database":"P1212410_BIB","user":"P1212410_BIB","active":true},
      {"name":"TOTALPEC_PRD",...,"active":false}
    ],
    "count":  7,
    "active": "TOTALPEC_BIB"
  }}`,
	RunE: connListRun,
}

func connListRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if !creds.HasProtheusProfiles() {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "NO_PROFILES",
			"no Protheus profiles registered",
			"Run: mapj protheus connection add <name> --server ... --database ... --user ... --password ...",
			false,
		)
		fmt.Println(formatter.Format(env))
		return nil
	}

	// Build structured list
	type profileEntry struct {
		Name     string `json:"name"`
		Server   string `json:"server"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		User     string `json:"user"`
		Active   bool   `json:"active"`
	}

	profiles := []profileEntry{}

	// Handle legacy v1
	if len(creds.ProtheusProfiles) == 0 && creds.Protheus != nil {
		profiles = append(profiles, profileEntry{
			Name: "default (legacy)", Server: creds.Protheus.Server,
			Port: creds.Protheus.Port, Database: creds.Protheus.Database,
			User: creds.Protheus.User, Active: true,
		})
	} else {
		for _, name := range creds.ProtheusProfileNames() {
			p := creds.ProtheusProfiles[name]
			profiles = append(profiles, profileEntry{
				Name: name, Server: p.Server, Port: p.Port,
				Database: p.Database, User: p.User,
				Active: name == creds.ProtheusActive,
			})
		}
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"profiles": profiles,
		"count":    len(profiles),
		"active":   creds.ProtheusActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- USE ----

var connUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active Protheus connection (affects all future queries)",
	Long: `Switch the active connection profile. Future queries will use this profile.
Does NOT affect queries using --connection flag.

OUTPUT SCHEMA:
  {"ok":true,"result":{"previous":"TOTALPEC_BIB","active":"TOTALPEC_PRD",
                        "server":"192.168.99.102","port":1433,"database":"P1212410_PRD"}}`,
	Args: cobra.ExactArgs(1),
	RunE: connUseRun,
}

func connUseRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
		hint := "Available profiles: " + strings.Join(creds.ProtheusProfileNames(), ", ")
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "PROFILE_NOT_FOUND",
			fmt.Sprintf("profile '%s' not found", name),
			hint, false,
		)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("profile not found: %s", name)
	}

	prev := creds.ProtheusActive
	creds.ProtheusActive = name

	if err := store.Save(creds); err != nil {
		return err
	}

	p := creds.ProtheusProfiles[name]
	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"previous": prev,
		"active":   name,
		"server":   p.Server,
		"port":     p.Port,
		"database": p.Database,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- REMOVE ----

var connRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a registered Protheus connection profile",
	Args:  cobra.ExactArgs(1),
	RunE:  connRemoveRun,
}

func connRemoveRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
			fmt.Sprintf("profile '%s' not found", name), false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("profile '%s' not found", name)
	}

	wasActive := creds.ProtheusActive == name
	delete(creds.ProtheusProfiles, name)

	newActive := ""
	if wasActive {
		creds.ProtheusActive = ""
		remaining := creds.ProtheusProfileNames()
		if len(remaining) > 0 {
			newActive = remaining[0]
			creds.ProtheusActive = newActive
		}
	}

	if err := store.Save(creds); err != nil {
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"removed":   name,
		"wasActive": wasActive,
		"newActive": newActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- SHOW ----

var connShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show details of a profile (password masked). Defaults to active.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  connShowRun,
}

func connShowRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	var p *auth.ProtheusProfile
	var name string

	if len(args) == 0 {
		p = creds.ActiveProtheusProfile()
		if p == nil {
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(), "NO_ACTIVE_PROFILE",
				"no active Protheus profile",
				"Run: mapj protheus connection list && mapj protheus connection use <name>",
				false,
			)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("no active Protheus profile")
		}
		name = creds.ProtheusActive
	} else {
		name = args[0]
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
				fmt.Sprintf("profile '%s' not found", name), false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.ProtheusProfiles[name]
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"name":     p.Name,
		"server":   p.Server,
		"port":     p.Port,
		"database": p.Database,
		"user":     p.User,
		"password": maskPassword(p.Password),
		"active":   name == creds.ProtheusActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- PING ----

var connPingCmd = &cobra.Command{
	Use:   "ping [name]",
	Short: "Test connectivity to a Protheus profile (returns latencyMs)",
	Long: `Test TCP+SQL connection to a Protheus profile.
Defaults to active profile if no name given.

OUTPUT SCHEMA (success):
  {"ok":true,"result":{"profile":"TOTALPEC_BIB","server":"192.168.99.102",
                        "port":1433,"database":"P1212410_BIB","latencyMs":147,"ok":true}}

OUTPUT SCHEMA (failure — with VPN hint):
  {"ok":false,"error":{"code":"PING_FAILED","retryable":true,
                        "hint":"Connect to TOTALPEC VPN for server 192.168.99.102. ..."}}

VPN auto-detection:
  192.168.99.x → TOTALPEC VPN
  192.168.7.x  → UNION VPN

EXAMPLES:
  mapj protheus connection ping              # test active
  mapj protheus connection ping TOTALPEC_BIB # test specific`,
	Args: cobra.MaximumNArgs(1),
	RunE: connPingRun,
}

func connPingRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	var p *auth.ProtheusProfile
	if len(args) == 0 {
		p = creds.ActiveProtheusProfile()
		if p == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED",
				"no active Protheus profile configured", false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("no active Protheus profile configured")
		}
	} else {
		name := args[0]
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
				fmt.Sprintf("profile '%s' not found", name), false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.ProtheusProfiles[name]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := protheus.NewClient(p.Server, p.Port, p.Database, p.User, p.Password)
	latency, err := client.Ping(ctx)
	if err != nil {
		hint := vpnHintForServer(p.Server, p.User, p.Database)
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "PING_FAILED",
			fmt.Sprintf("connection to %s:%d failed: %s", p.Server, p.Port, err.Error()),
			hint, true,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"profile":  p.Name,
		"server":   p.Server,
		"port":     p.Port,
		"database": p.Database,
		"latencyMs": latency,
		"ok":       true,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ======================== INIT ========================

func init() {
	protheusCmd.AddCommand(protheusConnectionCmd)
	protheusConnectionCmd.AddCommand(
		connAddCmd,
		connListCmd,
		connUseCmd,
		connRemoveCmd,
		connShowCmd,
		connPingCmd,
	)

	connAddCmd.Flags().StringVar(&connAddServer, "server", "", "SQL Server host/IP")
	connAddCmd.Flags().IntVar(&connAddPort, "port", 1433, "SQL Server port")
	connAddCmd.Flags().StringVar(&connAddDatabase, "database", "", "Database name")
	connAddCmd.Flags().StringVar(&connAddUser, "user", "", "Database user")
	connAddCmd.Flags().StringVar(&connAddPassword, "password", "", "Database password")
	connAddCmd.Flags().BoolVar(&connAddUse, "use", false, "Set as active connection immediately")
	connAddCmd.MarkFlagRequired("server")
	connAddCmd.MarkFlagRequired("database")
	connAddCmd.MarkFlagRequired("user")
	connAddCmd.MarkFlagRequired("password")
}

// ======================== HELPERS ========================

func maskPassword(pwd string) string {
	if len(pwd) <= 4 {
		return strings.Repeat("*", len(pwd))
	}
	return pwd[:2] + strings.Repeat("*", len(pwd)-4) + pwd[len(pwd)-2:]
}

func vpnHintForServer(server, user, database string) string {
	var vpnName string
	switch {
	case strings.HasPrefix(server, "192.168.99."):
		vpnName = "TOTALPEC"
	case strings.HasPrefix(server, "192.168.7."):
		vpnName = "UNION"
	default:
		return fmt.Sprintf("Verify VPN for server %s is active. Credentials: user=%s, database=%s", server, user, database)
	}
	return fmt.Sprintf("Connect to %s VPN for server %s. Credentials: user=%s, database=%s", vpnName, server, user, database)
}
