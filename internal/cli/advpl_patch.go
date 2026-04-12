package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

var advplPatchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Generate, validate, apply, and inspect patches",
	Long: `Operations on TOTVS RPO patches (PTM/UPD/PAK).

SUBCOMMANDS:
  patch generate   Generate a patch from RPO resources
  patch apply      Apply a patch to the RPO
  patch info       Inspect contents of a patch file`,
}

var patchConnection string

// ---- GENERATE ----

var patchGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a patch from RPO resources",
	Long: `Generate a patch file from specified RPO resources.

Patch types: PTM (1), UPD (2), PAK (3)

OUTPUT SCHEMA:
  {"ok":true,"result":{"patchFile":"./out/fix.ptm","type":"PTM","resources":["MATA110.PRW"]}}

EXAMPLES:
  mapj advpl patch generate --resources MATA110.PRW,MATA120.PRW --type PTM --output ./patches/
  mapj advpl patch generate --resources MATA110.PRW --type UPD --output ./patches/ --name fix_vendas`,
	RunE: patchGenerateRun,
}

var (
	patchResources []string
	patchType      string
	patchOutput    string
	patchName      string
)

func patchGenerateRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if len(patchResources) == 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --resources", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}
	if patchOutput == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --output directory", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	patchTypeInt := 1 // default PTM
	switch patchType {
	case "PTM", "ptm", "1":
		patchTypeInt = 1
	case "UPD", "upd", "2":
		patchTypeInt = 2
	case "PAK", "pak", "3":
		patchTypeInt = 3
	default:
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR",
			fmt.Sprintf("invalid patch type '%s', use PTM/UPD/PAK", patchType), false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	absOutput, _ := filepath.Abs(patchOutput)
	saveLocal := "file:///" + filepath.ToSlash(absOutput)

	return withTDSClient(cmd, patchConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Generating %s patch with %d resource(s)...\n", patchType, len(patchResources))
		if err := client.PatchGenerate(context.Background(), patchResources, patchTypeInt, saveLocal, "", patchName); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PATCH_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"type":      patchType,
			"output":    patchOutput,
			"name":      patchName,
			"resources": patchResources,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- APPLY ----

var patchApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a patch to the RPO",
	Long: `Apply a patch file (PTM/UPD/PAK) to the connected RPO.

OUTPUT SCHEMA:
  {"ok":true,"result":{"file":"./patches/fix.ptm","applied":true}}

EXAMPLES:
  mapj advpl patch apply --file ./patches/fix_vendas.ptm
  mapj advpl patch apply --file ./patches/fix_vendas.ptm --connection HOMOLOGACAO`,
	RunE: patchApplyRun,
}

var patchApplyFile string

func patchApplyRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if patchApplyFile == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --file", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	absFile, _ := filepath.Abs(patchApplyFile)
	patchUri := "file:///" + filepath.ToSlash(absFile)

	return withTDSClient(cmd, patchConnection, func(client *tds.Client) error {
		fmt.Fprintf(os.Stderr, "Applying patch %s...\n", patchApplyFile)
		if err := client.PatchApply(context.Background(), patchUri, true, ""); err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PATCH_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"file":    patchApplyFile,
			"applied": true,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ---- INFO ----

var patchInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Inspect contents of a patch file",
	Long: `Show the resources contained in a patch file.

OUTPUT SCHEMA:
  {"ok":true,"result":{"file":"fix.ptm","entries":[{"name":"MATA110.PRW","date":"15/03/2024"}]}}

EXAMPLES:
  mapj advpl patch info --file ./patches/fix_vendas.ptm`,
	RunE: patchInfoRun,
}

var patchInfoFile string

func patchInfoRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if patchInfoFile == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "specify --file", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	absFile, _ := filepath.Abs(patchInfoFile)
	patchUri := "file:///" + filepath.ToSlash(absFile)

	return withTDSClient(cmd, patchConnection, func(client *tds.Client) error {
		entries, err := client.PatchInfo(context.Background(), patchUri, "")
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "PATCH_ERROR", err.Error(), true)
			fmt.Println(formatter.Format(env))
			return err
		}

		env := output.NewEnvelope(cmd.CommandPath(), map[string]any{
			"file":    patchInfoFile,
			"total":   len(entries),
			"entries": entries,
		})
		fmt.Println(formatter.Format(env))
		return nil
	})
}

// ======================== INIT ========================

func init() {
	advplCmd.AddCommand(advplPatchCmd)
	advplPatchCmd.AddCommand(
		patchGenerateCmd,
		patchApplyCmd,
		patchInfoCmd,
	)

	// Shared flags
	advplPatchCmd.PersistentFlags().StringVar(&patchConnection, "connection", "", "Use specific profile without switching active")

	// Generate flags
	patchGenerateCmd.Flags().StringSliceVar(&patchResources, "resources", nil, "RPO resources to include (comma-separated)")
	patchGenerateCmd.Flags().StringVar(&patchType, "type", "PTM", "Patch type: PTM, UPD, PAK")
	patchGenerateCmd.Flags().StringVar(&patchOutput, "output", "", "Output directory for patch file")
	patchGenerateCmd.Flags().StringVar(&patchName, "name", "", "Optional patch name")
	patchGenerateCmd.MarkFlagRequired("resources")
	patchGenerateCmd.MarkFlagRequired("output")

	// Apply flags
	patchApplyCmd.Flags().StringVar(&patchApplyFile, "file", "", "Patch file to apply")
	patchApplyCmd.MarkFlagRequired("file")

	// Info flags
	patchInfoCmd.Flags().StringVar(&patchInfoFile, "file", "", "Patch file to inspect")
	patchInfoCmd.MarkFlagRequired("file")
}
