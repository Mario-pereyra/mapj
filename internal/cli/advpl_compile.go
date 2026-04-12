package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/pkg/tds"
	"github.com/spf13/cobra"
)

var advplCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile AdvPL/TLPP source files on the connected AppServer",
	Long: `Compile source files into the RPO via the TOTVS Language Server.

OUTPUT SCHEMA:
  {"ok":true,"command":"mapj advpl compile","result":{
    "compiled":2,"succeeded":1,"failed":1,
    "files":[
      {"file":"src/MATA110.PRW","status":"error","diagnostics":[
        {"line":42,"severity":"error","message":"Variable 'cNome' not declared"}
      ]},
      {"file":"src/MATA120.PRW","status":"success","diagnostics":[]}
    ]
  }}

AGENTIC USE: Read diagnostics → fix code → recompile in a loop.

EXAMPLES:
  mapj advpl compile --files src/MATA110.PRW --includes /path/to/includes
  mapj advpl compile --dir ./src --includes /includes --recompile
  mapj advpl compile --files src/prog1.prw,src/prog2.prw --includes /includes --connection HOMOLOGACAO`,
	RunE: advplCompileRun,
}

var (
	compileFiles      []string
	compileDir        string
	compileIncludes   []string
	compileRecompile  bool
	compileConnection string
)

func init() {
	advplCmd.AddCommand(advplCompileCmd)

	advplCompileCmd.Flags().StringSliceVar(&compileFiles, "files", nil, "Source files to compile (comma-separated)")
	advplCompileCmd.Flags().StringVar(&compileDir, "dir", "", "Directory of sources to compile (recursive)")
	advplCompileCmd.Flags().StringSliceVar(&compileIncludes, "includes", nil, "Include directories (comma-separated)")
	advplCompileCmd.Flags().BoolVar(&compileRecompile, "recompile", false, "Force full recompile")
	advplCompileCmd.Flags().StringVar(&compileConnection, "connection", "", "Use specific profile without switching active")
}

func advplCompileRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()

	if len(compileFiles) == 0 && compileDir == "" {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR",
			"specify --files or --dir", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("USAGE_ERROR")
	}

	// Resolve files
	files := compileFiles
	if compileDir != "" {
		dirFiles, err := collectSourceFiles(compileDir)
		if err != nil {
			env := output.NewErrorEnvelope(cmd.CommandPath(), "FILE_ERROR", err.Error(), false)
			fmt.Println(formatter.Format(env))
			return err
		}
		files = append(files, dirFiles...)
	}

	if len(files) == 0 {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR",
			"no source files found", false)
		fmt.Println(formatter.Format(env))
		return fmt.Errorf("no source files found")
	}

	// Resolve connection profile
	profile, err := resolveTDSProfile(compileConnection)
	if err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "NO_CONNECTION", err.Error(),
			"Run: mapj advpl connection add <name> --server ... --environment ... --use",
			false,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Find advpls
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start LSP client
	fmt.Fprintf(os.Stderr, "Connecting to %s:%d...\n", profile.Server, profile.Port)
	client, err := tds.NewClient(ctx, advplsPath)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "ADVPLS_START_FAILED", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}
	defer client.Close()

	// Connect and authenticate
	if err := client.ConnectAndAuth(ctx, profile.Server, profile.Port, profile.Environment, profile.User, profile.Password, profile.Secure); err != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(), "AUTH_FAILED", err.Error(),
			fmt.Sprintf("Verify AppServer at %s:%d is running. Check user/password for environment '%s'", profile.Server, profile.Port, profile.Environment),
			true,
		)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Convert file paths to URIs
	fileUris := make([]string, len(files))
	for i, f := range files {
		abs, _ := filepath.Abs(f)
		fileUris[i] = "file:///" + filepath.ToSlash(abs)
	}

	includeUris := make([]string, len(compileIncludes))
	for i, inc := range compileIncludes {
		abs, _ := filepath.Abs(inc)
		includeUris[i] = "file:///" + filepath.ToSlash(abs)
	}

	// Compile
	fmt.Fprintf(os.Stderr, "Compiling %d file(s)...\n", len(files))
	result, err := client.Compile(ctx, fileUris, includeUris, "", compileRecompile)
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "COMPILE_ERROR", err.Error(), true)
		fmt.Println(formatter.Format(env))
		return err
	}

	// Build structured output
	succeeded := 0
	failed := 0
	type fileResult struct {
		File        string                `json:"file"`
		Status      string                `json:"status"`
		Diagnostics []tds.CompileDiagnostic `json:"diagnostics"`
	}

	fileResults := make([]fileResult, 0)
	if result != nil && result.Files != nil {
		for _, fi := range result.Files {
			fr := fileResult{
				File:        fi.File,
				Status:      fi.Status,
				Diagnostics: fi.Detail,
			}
			if fr.Diagnostics == nil {
				fr.Diagnostics = []tds.CompileDiagnostic{}
			}
			if fi.Status == "error" {
				failed++
			} else {
				succeeded++
			}
			fileResults = append(fileResults, fr)
		}
	}

	ok := failed == 0
	env := &output.Envelope{
		OK:      ok,
		Command: cmd.CommandPath(),
		Result: map[string]any{
			"compiled":  len(files),
			"succeeded": succeeded,
			"failed":    failed,
			"files":     fileResults,
		},
	}
	fmt.Println(formatter.Format(env))

	if !ok {
		return fmt.Errorf("compilation failed: %d errors", failed)
	}
	return nil
}

// resolveTDSProfile resolves a TDS profile by name or returns the active one.
func resolveTDSProfile(connectionName string) (*auth.TDSProfile, error) {
	store, err := auth.NewStore()
	if err != nil {
		return nil, err
	}
	creds, err := store.Load()
	if err != nil {
		return nil, err
	}

	if connectionName != "" {
		if creds.TDSProfiles == nil || creds.TDSProfiles[connectionName] == nil {
			return nil, fmt.Errorf("profile '%s' not found", connectionName)
		}
		return creds.TDSProfiles[connectionName], nil
	}

	p := creds.ActiveTDSProfile()
	if p == nil {
		return nil, fmt.Errorf("no active TOTVS AppServer profile configured")
	}
	return p, nil
}

// collectSourceFiles walks a directory and collects AdvPL/TLPP source files.
func collectSourceFiles(dir string) ([]string, error) {
	validExts := map[string]bool{
		".prw": true, ".prx": true, ".prg": true, ".aph": true,
		".tlpp": true, ".4gl": true, ".per": true, ".apl": true,
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if validExts[ext] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
