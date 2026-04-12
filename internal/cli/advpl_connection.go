package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

// ======================== CONNECTION SUBCOMMAND ========================

var advplConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Manage TOTVS Application Server connection profiles",
	Long: `Manage named, encrypted connection profiles for TOTVS Application Servers.

PROFILE STORAGE: encrypted at ~/.config/mapj/credentials.enc
ACTIVE PROFILE: commands use the active profile by default.

SUBCOMMANDS:
  connection add <name>    Register new profile (--server --port --environment --user --password)
  connection list          List all profiles (JSON with active field)
  connection use <name>    Switch active profile
  connection ping [name]   Test connectivity to AppServer
  connection show [name]   Show profile details (password masked)
  connection remove <name> Delete profile`,
}

// ---- ADD ----

var tdsConnAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Register a named TOTVS AppServer connection profile",
	Long: `Register a named TOTVS Application Server connection profile.

Examples:
  mapj advpl connection add PRODUCAO --server 192.168.1.100 --port 5025 --environment producao --user admin --password ""
  mapj advpl connection add HOMOLOGACAO --server 192.168.1.101 --port 5025 --environment homologacao --user admin --password "" --secure --use`,
	Args: cobra.ExactArgs(1),
	RunE: tdsConnAddRun,
}

var (
	tdsConnServer      string
	tdsConnPort        int
	tdsConnEnvironment string
	tdsConnUser        string
	tdsConnPassword    string
	tdsConnSecure      bool
	tdsConnUse         bool
)

func tdsConnAddRun(cmd *cobra.Command, args []string) error {
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

	profile := &auth.TDSProfile{
		Name:        name,
		Server:      tdsConnServer,
		Port:        tdsConnPort,
		Environment: tdsConnEnvironment,
		User:        tdsConnUser,
		Password:    tdsConnPassword,
		Secure:      tdsConnSecure,
	}

	isFirst := !creds.HasTDSProfiles()
	setActive := tdsConnUse || isFirst
	creds.SetTDSProfile(profile, setActive)

	if err := store.Save(creds); err != nil {
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"name":        name,
		"server":      tdsConnServer,
		"port":        tdsConnPort,
		"environment": tdsConnEnvironment,
		"secure":      tdsConnSecure,
		"setActive":   setActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- LIST ----

var tdsConnListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered TOTVS AppServer profiles",
	Long: `List all registered TOTVS Application Server profiles.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj advpl connection list","result":{
    "profiles": [{"name":"PRODUCAO","server":"192.168.1.100","port":5025,"environment":"producao","secure":false,"active":true}],
    "count": 1,
    "active": "PRODUCAO"
  }}`,
	RunE: tdsConnListRun,
}

func tdsConnListRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	if !creds.HasTDSProfiles() {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "NO_PROFILES",
			"no TOTVS AppServer profiles registered",
			"Run: mapj advpl connection add <name> --server ... --port ... --environment ... --user ... --password ...",
			false,
		)
		fmt.Println(formatter.Format(env))
		return nil
	}

	type profileEntry struct {
		Name        string `json:"name"`
		Server      string `json:"server"`
		Port        int    `json:"port"`
		Environment string `json:"environment"`
		Secure      bool   `json:"secure"`
		Active      bool   `json:"active"`
	}

	profiles := []profileEntry{}
	for _, name := range creds.TDSProfileNames() {
		p := creds.TDSProfiles[name]
		profiles = append(profiles, profileEntry{
			Name: name, Server: p.Server, Port: p.Port,
			Environment: p.Environment, Secure: p.Secure,
			Active: name == creds.TDSActive,
		})
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"profiles": profiles,
		"count":    len(profiles),
		"active":   creds.TDSActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- USE ----

var tdsConnUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active TOTVS AppServer connection",
	Args:  cobra.ExactArgs(1),
	RunE:  tdsConnUseRun,
}

func tdsConnUseRun(cmd *cobra.Command, args []string) error {
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

	if creds.TDSProfiles == nil || creds.TDSProfiles[name] == nil {
		hint := "Available profiles: " + strings.Join(creds.TDSProfileNames(), ", ")
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "PROFILE_NOT_FOUND",
			fmt.Sprintf("profile '%s' not found", name),
			hint, false,
		)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("profile not found: %s", name)
	}

	prev := creds.TDSActive
	creds.TDSActive = name

	if err := store.Save(creds); err != nil {
		return err
	}

	p := creds.TDSProfiles[name]
	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"previous":    prev,
		"active":      name,
		"server":      p.Server,
		"port":        p.Port,
		"environment": p.Environment,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- REMOVE ----

var tdsConnRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a registered TOTVS AppServer profile",
	Args:  cobra.ExactArgs(1),
	RunE:  tdsConnRemoveRun,
}

func tdsConnRemoveRun(cmd *cobra.Command, args []string) error {
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

	if creds.TDSProfiles == nil || creds.TDSProfiles[name] == nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
			fmt.Sprintf("profile '%s' not found", name), false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("profile '%s' not found", name)
	}

	wasActive := creds.TDSActive == name
	delete(creds.TDSProfiles, name)

	newActive := ""
	if wasActive {
		creds.TDSActive = ""
		remaining := creds.TDSProfileNames()
		if len(remaining) > 0 {
			newActive = remaining[0]
			creds.TDSActive = newActive
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

var tdsConnShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show details of a TOTVS AppServer profile (password masked). Defaults to active.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  tdsConnShowRun,
}

func tdsConnShowRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	var p *auth.TDSProfile
	var name string

	if len(args) == 0 {
		p = creds.ActiveTDSProfile()
		if p == nil {
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(), "NO_ACTIVE_PROFILE",
				"no active TOTVS AppServer profile",
				"Run: mapj advpl connection list && mapj advpl connection use <name>",
				false,
			)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("no active TOTVS AppServer profile")
		}
		name = creds.TDSActive
	} else {
		name = args[0]
		if creds.TDSProfiles == nil || creds.TDSProfiles[name] == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
				fmt.Sprintf("profile '%s' not found", name), false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.TDSProfiles[name]
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"name":        p.Name,
		"server":      p.Server,
		"port":        p.Port,
		"environment": p.Environment,
		"user":        p.User,
		"password":    maskPassword(p.Password),
		"secure":      p.Secure,
		"active":      name == creds.TDSActive,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- PING ----

var tdsConnPingCmd = &cobra.Command{
	Use:   "ping [name]",
	Short: "Test connectivity to a TOTVS AppServer (validates build version)",
	Long: `Test connectivity to a TOTVS Application Server by calling $totvsserver/validation.
Returns server build version and secure status.

OUTPUT SCHEMA (success):
  {"ok":true,"result":{"profile":"PRODUCAO","server":"192.168.1.100","port":5025,
                        "build":"7.00.210324A","secure":false,"latencyMs":245}}`,
	Args: cobra.MaximumNArgs(1),
	RunE: tdsConnPingRun,
}

func tdsConnPingRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	store, err := auth.NewStore()
	if err != nil {
		return err
	}
	creds, err := store.Load()
	if err != nil {
		return err
	}

	var p *auth.TDSProfile
	if len(args) == 0 {
		p = creds.ActiveTDSProfile()
		if p == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "NOT_AUTHENTICATED",
				"no active TOTVS AppServer profile configured", false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("no active TOTVS AppServer profile configured")
		}
	} else {
		name := args[0]
		if creds.TDSProfiles == nil || creds.TDSProfiles[name] == nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PROFILE_NOT_FOUND",
				fmt.Sprintf("profile '%s' not found", name), false)
			fmt.Println(formatter.Format(env))
			return fmt.Errorf("profile '%s' not found", name)
		}
		p = creds.TDSProfiles[name]
	}

	// Find advpls binary
	advplsPath, err := tds.FindAdvplsBinary()
	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "ADVPLS_NOT_FOUND",
			err.Error(),
			"Install advpls: npm install -g @totvs/tds-ls, or place the binary in PATH",
			false,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "ADVPLS_START_FAILED",
			fmt.Sprintf("failed to start advpls: %s", err.Error()),
			"Verify advpls binary is valid and executable",
			true,
		)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer client.Close()

	valResult, err := client.Validate(ctx, p.Server, p.Port)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "PING_FAILED",
			fmt.Sprintf("connection to %s:%d failed: %s", p.Server, p.Port, err.Error()),
			fmt.Sprintf("Verify AppServer is running at %s:%d and network/VPN is active", p.Server, p.Port),
			true,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"profile":   p.Name,
		"server":    p.Server,
		"port":      p.Port,
		"build":     valResult.Build,
		"secure":    valResult.Secure != 0,
		"latencyMs": latencyMs,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ======================== INIT ========================

func init() {
	advplCmd.AddCommand(advplConnectionCmd)
	advplConnectionCmd.AddCommand(
		tdsConnAddCmd,
		tdsConnListCmd,
		tdsConnUseCmd,
		tdsConnRemoveCmd,
		tdsConnShowCmd,
		tdsConnPingCmd,
	)

	tdsConnAddCmd.Flags().StringVar(&tdsConnServer, "server", "", "AppServer host/IP")
	tdsConnAddCmd.Flags().IntVar(&tdsConnPort, "port", 5025, "AppServer port")
	tdsConnAddCmd.Flags().StringVar(&tdsConnEnvironment, "environment", "", "Target environment")
	tdsConnAddCmd.Flags().StringVar(&tdsConnUser, "user", "admin", "Username")
	tdsConnAddCmd.Flags().StringVar(&tdsConnPassword, "password", "", "Password")
	tdsConnAddCmd.Flags().BoolVar(&tdsConnSecure, "secure", false, "Use secure connection (SSL)")
	tdsConnAddCmd.Flags().BoolVar(&tdsConnUse, "use", false, "Set as active connection immediately")
	tdsConnAddCmd.MarkFlagRequired("server")
	tdsConnAddCmd.MarkFlagRequired("environment")
}
