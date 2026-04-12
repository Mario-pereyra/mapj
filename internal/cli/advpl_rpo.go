package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

var advplRpoCmd = &cobra.Command{
	Use:   "rpo",
	Short: "Inspect and manage the RPO (Repository of Objects)",
	Long: `Operations on the TOTVS RPO (Repository of Objects).

SUBCOMMANDS:
  rpo inspect      List objects in the RPO
  rpo functions    List functions in the RPO
  rpo integrity    Check RPO integrity
  rpo defrag       Defragment the RPO
  rpo revalidate   Revalidate the RPO
  rpo delete       Delete programs from the RPO
  rpo info         Get RPO information`,
}

var rpoConnection string

// ---- INSPECT ----

var rpoInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "List objects (sources/resources) in the RPO",
	Long: `List all objects in the RPO. Use --filter to search by name pattern.

OUTPUT SCHEMA:
  {"ok":true,"result":{"total":1523,"objects":[
    {"source":"MATA110.PRW","date":"15/03/2024","sourceStatus":"N","rpoStatus":"R"}
  ]}}

Safety Tripwire: results > 500 objects auto-save to file.`,
	RunE: rpoInspectRun,
}

var rpoFilter string

func rpoInspectRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	profile, err := resolveTDSProfile(rpoConnection)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NO_CONNECTION", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	advplsPath, err := tds.FindAdvplsBinary()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_NOT_FOUND", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_START_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer client.Close()

	if err := client.ConnectAndAuth(ctx, profile.Server, profile.Port, profile.Environment, profile.User, profile.Password, profile.Secure); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	fmt.Fprintf(os.Stderr, "Inspecting RPO objects...\n")
	objects, err := client.InspectObjects(ctx, true)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Apply filter
	if rpoFilter != "" {
		filter := strings.ToUpper(rpoFilter)
		filtered := make([]tds.ObjectData, 0)
		for _, obj := range objects {
			if matchWildcard(strings.ToUpper(obj.Source), filter) {
				filtered = append(filtered, obj)
			}
		}
		objects = filtered
	}

	// Safety Tripwire
	tripwireThreshold := 500
	if len(objects) > tripwireThreshold {
		tempFile := fmt.Sprintf("mapj_rpo_inspect_%d.toon", time.Now().UnixNano())
		toonFormatter := output.NewFormatter("toon")

		env := output.NewEnvelope(cmd.CommandPath(), objects)
		content := toonFormatter.Format(env)
		if err := output.WriteToFile(tempFile, content); err == nil {
			fmt.Fprintf(os.Stderr, "Safety Tripwire: %d objects auto-saved to %s\n", len(objects), tempFile)
			summary := output.NewEnvelope(cmd.CommandPath(), map[string]any{
				"total":      len(objects),
				"outputFile": tempFile,
				"format":     "toon",
			})
			fmt.Println(formatter.Format(summary))
			return nil
		}
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"total":   len(objects),
		"objects": objects,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- FUNCTIONS ----

var rpoFunctionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "List functions in the RPO",
	Long: `List all functions/methods in the RPO.

OUTPUT SCHEMA:
  {"ok":true,"result":{"total":5230,"functions":[
    {"function":"MATA110","source":"MATA110.PRW","line":1,"sourceStatus":"N","rpoStatus":"R"}
  ]}}`,
	RunE: rpoFunctionsRun,
}

func rpoFunctionsRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	profile, err := resolveTDSProfile(rpoConnection)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "NO_CONNECTION", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	advplsPath, err := tds.FindAdvplsBinary()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_NOT_FOUND", err.Error(), false)
		fmt.Println(formatter.Format(env))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_START_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer client.Close()

	if err := client.ConnectAndAuth(ctx, profile.Server, profile.Port, profile.Environment, profile.User, profile.Password, profile.Secure); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	fmt.Fprintf(os.Stderr, "Inspecting RPO functions...\n")
	functions, err := client.InspectFunctions(ctx)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Apply filter
	if rpoFilter != "" {
		filter := strings.ToUpper(rpoFilter)
		filtered := make([]tds.FunctionData, 0)
		for _, fn := range functions {
			if matchWildcard(strings.ToUpper(fn.Function), filter) || matchWildcard(strings.ToUpper(fn.Source), filter) {
				filtered = append(filtered, fn)
			}
		}
		functions = filtered
	}

	// Safety Tripwire
	tripwireThreshold := 500
	if len(functions) > tripwireThreshold {
		tempFile := fmt.Sprintf("mapj_rpo_functions_%d.toon", time.Now().UnixNano())
		toonFormatter := output.NewFormatter("toon")

		env := output.NewEnvelope(cmd.CommandPath(), functions)
		content := toonFormatter.Format(env)
		if err := output.WriteToFile(tempFile, content); err == nil {
			fmt.Fprintf(os.Stderr, "Safety Tripwire: %d functions auto-saved to %s\n", len(functions), tempFile)
			summary := output.NewEnvelope(cmd.CommandPath(), map[string]any{
				"total":      len(functions),
				"outputFile": tempFile,
				"format":     "toon",
			})
			fmt.Println(formatter.Format(summary))
			return nil
		}
	}

	env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
		"total":     len(functions),
		"functions": functions,
	})
	fmt.Println(formatter.Format(env))
	return nil
}

// ---- INTEGRITY ----

var rpoIntegrityCmd = &cobra.Command{
	Use:   "integrity",
	Short: "Check RPO integrity",
	RunE:  rpoIntegrityRun,
}

func rpoIntegrityRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, rpoConnection, func(client *tds.Client) error {
		result, err := client.RpoCheckIntegrity(context.Background())
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"integrity": result.Integrity,
			"message":   result.Message,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- DEFRAG ----

var rpoDefragCmd = &cobra.Command{
	Use:   "defrag",
	Short: "Defragment the RPO",
	RunE:  rpoDefragRun,
}

var rpoCleanHistory bool

func rpoDefragRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, rpoConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Defragmenting RPO (this may take some time)...\n")
		if err := client.DefragRpo(context.Background(), "", rpoCleanHistory); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"success":      true,
			"cleanHistory": rpoCleanHistory,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- REVALIDATE ----

var rpoRevalidateCmd = &cobra.Command{
	Use:   "revalidate",
	Short: "Revalidate the RPO",
	RunE:  rpoRevalidateRun,
}

func rpoRevalidateRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, rpoConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Revalidating RPO...\n")
		if err := client.RevalidateRpo(context.Background()); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"success": true,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- DELETE ----

var rpoDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete programs from the RPO",
	RunE:  rpoDeleteRun,
}

var rpoDeletePrograms []string

func rpoDeleteRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if len(rpoDeletePrograms) == 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --programs", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	return withTDSClient(cmd, rpoConnection, func(client *tds.Client) error {
		result, err := client.DeletePrograms(context.Background(), rpoDeletePrograms, "")
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"deleted":    rpoDeletePrograms,
			"returnCode": result.ReturnCode,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- INFO ----

var rpoInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get RPO information",
	RunE:  rpoInfoRun,
}

func rpoInfoRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	return withTDSClient(cmd, rpoConnection, func(client *tds.Client) error {
		result, err := client.RpoInfo(context.Background())
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "RPO_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		// Pass raw JSON through
		var parsed any
		if err := json.Unmarshal(result, &parsed); err != nil {
			parsed = string(result)
		}

		env := output.NewEnvelope(cmd.CommandPath(), parsed)
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ======================== INIT ========================

func init() {
	advplCmd.AddCommand(advplRpoCmd)
	advplRpoCmd.AddCommand(
		rpoInspectCmd,
		rpoFunctionsCmd,
		rpoIntegrityCmd,
		rpoDefragCmd,
		rpoRevalidateCmd,
		rpoDeleteCmd,
		rpoInfoCmd,
	)

	// Shared flags
	advplRpoCmd.PersistentFlags().StringVar(&rpoConnection, "connection", "", "Use specific profile without switching active")

	rpoInspectCmd.Flags().StringVar(&rpoFilter, "filter", "", "Filter objects by name pattern (supports * wildcard)")
	rpoFunctionsCmd.Flags().StringVar(&rpoFilter, "filter", "", "Filter functions by name pattern (supports * wildcard)")
	rpoDefragCmd.Flags().BoolVar(&rpoCleanHistory, "clean-history", false, "Clean patch history during defrag")
	rpoDeleteCmd.Flags().StringSliceVar(&rpoDeletePrograms, "programs", nil, "Programs to delete (comma-separated)")
	rpoDeleteCmd.MarkFlagRequired("programs")
}

// ======================== HELPERS ========================

// withTDSClient is a convenience wrapper that creates a connected TDS client,
// runs the provided function, and cleans up.
func withTDSClient(cmd *cobra.Command, connectionName string, fn func(client *tds.Client) error) error {
	formatter := GetFormatter()

	profile, err := resolveTDSProfile(connectionName)
	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "NO_CONNECTION", err.Error(),
			"Run: mapj advpl connection add <name> --server ... --environment ... --use",
			false,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	advplsPath, err := tds.FindAdvplsBinary()
	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "ADVPLS_NOT_FOUND", err.Error(),
			"Install: npm install -g @totvs/tds-ls",
			false,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_START_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer client.Close()

	if err := client.ConnectAndAuth(ctx, profile.Server, profile.Port, profile.Environment, profile.User, profile.Password, profile.Secure); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	return fn(client)
}

// matchWildcard performs a simple wildcard match (* only).
func matchWildcard(s, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(s, suffix)
	}
	if strings.Contains(pattern, "*") {
		parts := strings.SplitN(pattern, "*", 2)
		return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1])
	}
	return s == pattern
}
