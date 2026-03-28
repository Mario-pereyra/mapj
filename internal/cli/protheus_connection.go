package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
	"github.com/spf13/cobra"
)

// ======================== CONNECTION SUBCOMMAND ========================

var protheusConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Manage Protheus connection profiles (add, list, use, remove, ping)",
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
	if name == "" {
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
	creds.SetProtheusProfile(profile, connAddUse || isFirst)

	if err := store.Save(creds); err != nil {
		return err
	}

	active := ""
	if connAddUse || isFirst {
		active = " (set as active)"
	}
	fmt.Printf("✓ Profile '%s' registered%s → %s:%d/%s\n", name, active, connAddServer, connAddPort, connAddDatabase)
	return nil
}

// ---- LIST ----

var connListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered Protheus connection profiles",
	RunE:  connListRun,
}

func connListRun(cmd *cobra.Command, args []string) error {
	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	// Handle legacy v1 migration
	if !creds.HasProtheusProfiles() {
		fmt.Println("No Protheus profiles registered.")
		fmt.Println("Use: mapj protheus connection add <name> --server ... --database ... --user ... --password ...")
		return nil
	}

	// Show legacy profile if only v1 exists
	if len(creds.ProtheusProfiles) == 0 && creds.Protheus != nil {
		fmt.Println("Registered profiles:")
		fmt.Printf("  * default (legacy)  →  %s:%d / %s / user: %s\n",
			creds.Protheus.Server, creds.Protheus.Port,
			creds.Protheus.Database, creds.Protheus.User)
		fmt.Println("\n💡 Migrate to named profiles: mapj protheus connection add <name> --server ...")
		return nil
	}

	fmt.Println("Registered profiles:")
	for _, name := range creds.ProtheusProfileNames() {
		p := creds.ProtheusProfiles[name]
		marker := "  "
		activeTag := ""
		if name == creds.ProtheusActive {
			marker = "* "
			activeTag = "  ← ACTIVE"
		}
		fmt.Printf("  %s%-20s  %s:%d / %s / user: %s%s\n",
			marker, name, p.Server, p.Port, p.Database, p.User, activeTag)
	}
	fmt.Printf("\nTotal: %d profile(s). Active: %s\n", len(creds.ProtheusProfiles), creds.ProtheusActive)
	return nil
}

// ---- USE ----

var connUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active Protheus connection profile",
	Args:  cobra.ExactArgs(1),
	RunE:  connUseRun,
}

func connUseRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
		fmt.Printf("✗ Profile '%s' not found.\n", name)
		fmt.Println("  Available profiles:")
		for _, n := range creds.ProtheusProfileNames() {
			fmt.Printf("    - %s\n", n)
		}
		return fmt.Errorf("profile not found: %s", name)
	}

	prev := creds.ProtheusActive
	creds.ProtheusActive = name

	if err := store.Save(creds); err != nil {
		return err
	}

	p := creds.ProtheusProfiles[name]
	fmt.Printf("✓ Switched active profile: %s → %s\n", prev, name)
	fmt.Printf("  Server: %s:%d / Database: %s\n", p.Server, p.Port, p.Database)
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

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(creds.ProtheusProfiles, name)

	wasActive := creds.ProtheusActive == name
	if wasActive {
		creds.ProtheusActive = ""
		// Auto-select next profile if any remain
		remaining := creds.ProtheusProfileNames()
		if len(remaining) > 0 {
			creds.ProtheusActive = remaining[0]
			fmt.Printf("✓ Profile '%s' removed. Auto-switched active to: %s\n", name, remaining[0])
		} else {
			fmt.Printf("✓ Profile '%s' removed. No profiles remaining — configure one with 'connection add'.\n", name)
		}
	} else {
		fmt.Printf("✓ Profile '%s' removed.\n", name)
	}

	return store.Save(creds)
}

// ---- SHOW ----

var connShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show details of a profile (password masked). Defaults to active.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  connShowRun,
}

func connShowRun(cmd *cobra.Command, args []string) error {
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
		// Show active
		p = creds.ActiveProtheusProfile()
		if p == nil {
			return fmt.Errorf("no active Protheus profile. Use 'connection use <name>' or 'connection add'")
		}
		name = creds.ProtheusActive
	} else {
		name = args[0]
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.ProtheusProfiles[name]
	}

	activeTag := ""
	if name == creds.ProtheusActive || (creds.ProtheusActive == "" && name == "default") {
		activeTag = "  ← ACTIVE"
	}

	fmt.Printf("Profile: %s%s\n", p.Name, activeTag)
	fmt.Printf("  Server:   %s\n", p.Server)
	fmt.Printf("  Port:     %d\n", p.Port)
	fmt.Printf("  Database: %s\n", p.Database)
	fmt.Printf("  User:     %s\n", p.User)
	fmt.Printf("  Password: %s\n", maskPassword(p.Password))
	return nil
}

// ---- PING ----

var connPingCmd = &cobra.Command{
	Use:   "ping [name]",
	Short: "Test connectivity to a Protheus connection profile",
	Long: `Test the connection to a Protheus SQL Server profile.
If no name is given, tests the active connection.

Examples:
  mapj protheus connection ping              # test active
  mapj protheus connection ping TOTALPEC_BIB # test specific profile`,
	Args: cobra.MaximumNArgs(1),
	RunE: connPingRun,
}

func connPingRun(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("no active Protheus profile configured")
		}
	} else {
		name := args[0]
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[name] == nil {
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.ProtheusProfiles[name]
	}

	fmt.Printf("Pinging %s → %s:%d/%s ...\n", p.Name, p.Server, p.Port, p.Database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := protheus.NewClient(p.Server, p.Port, p.Database, p.User, p.Password)
	latency, err := client.Ping(ctx)
	if err != nil {
		fmt.Printf("✗ FAILED (%s)\n", err.Error())
		fmt.Println()
		fmt.Println("💡 Suggestions:")
		fmt.Println("   1. Verify you are connected to the VPN for this server:")
		if strings.HasPrefix(p.Server, "192.168.99.") {
			fmt.Printf("      TOTALPEC servers (%s) — connect to the TOTALPEC VPN\n", p.Server)
		} else if strings.HasPrefix(p.Server, "192.168.7.") {
			fmt.Printf("      UNION servers (%s) — connect to the UNION VPN\n", p.Server)
		} else {
			fmt.Printf("      Server %s — verify the appropriate VPN is active\n", p.Server)
		}
		fmt.Printf("   2. Verify credentials: user='%s', database='%s'\n", p.User, p.Database)
		fmt.Printf("   3. Verify the server is reachable: ping %s\n", p.Server)
		return err
	}

	fmt.Printf("✓ OK — %dms  [%s:%d / %s]\n", latency, p.Server, p.Port, p.Database)
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

	// add flags
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
