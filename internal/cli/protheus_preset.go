package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/internal/preset"
	"github.com/spf13/cobra"
)

// presetStore is the global preset store for CLI commands.
var presetStore *preset.PresetStore

// SetPresetStoreForTest allows injecting a custom preset store for testing.
func SetPresetStoreForTest(store *preset.PresetStore) {
	presetStore = store
}

// ResetPresetStore resets the preset store to nil (for testing cleanup).
func ResetPresetStore() {
	presetStore = nil
}

// initPresetStore initializes the preset store if not already initialized.
func initPresetStore() error {
	if presetStore != nil {
		return nil
	}

	var err error
	presetStore, err = preset.NewPresetStore()
	if err != nil {
		return fmt.Errorf("failed to initialize preset store: %w", err)
	}

	return nil
}

// ======================== PRESET SUBCOMMAND ========================

var protheusPresetCmd = &cobra.Command{
	Use:   "preset",
	Short: "Manage saved query presets with parameters",
	Long: `Manage saved query presets with parameters.

Presets allow you to save frequently used queries with parameter definitions.
Parameters are automatically detected from :placeholder syntax in queries.

STORAGE: ~/.config/mapj/presets.json (JSON format)

SUBCOMMANDS:
  preset add <name>       Create a new preset with query and parameters
  preset list             List all saved presets
  preset show [name]      Show preset details (defaults to active)
  preset run <name>       Execute a preset with parameters
  preset edit <name>      Modify an existing preset
  preset remove <name>    Delete a preset
  preset use <name>       Set a preset as active

Run 'mapj protheus preset <command> --help' for full output schema.`,
}

// ======================== ADD SUBCOMMAND ========================

var presetAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new query preset",
	Long: `Create a new query preset with optional parameters.

Parameters are automatically detected from :placeholder syntax in the query.
You can define parameter metadata using --param-def to specify type, default,
and description.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset add","result":{
    "name":"my-preset",
    "query":"SELECT :name FROM users WHERE id = :id",
    "detectedParameters":["name","id"],
    "parameters":[{"name":"id","type":"int","required":true}],
    "createdAt":"2024-01-15T10:30:00Z",
    "updatedAt":"2024-01-15T10:30:00Z"
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"PRESET_EXISTS","message":"...","hint":"..."}}

FLAGS:
  --query TEXT         (required) The SQL query with :parameter placeholders
  --description TEXT   Optional description of the preset
  --connection NAME    Optional default connection profile to use
  --max-rows N         Optional default max rows limit
  --param-def DEF      Parameter definition (repeatable)
                       Format: name:type[:default][:description]
                       Types: string, int, date, datetime, bool, list
  --tags TAGS          Comma-separated tags (e.g., "report,daily")
  --use                Set this preset as active immediately

EXAMPLES:
  mapj protheus preset add myquery --query "SELECT * FROM SA1010"
  
  mapj protheus preset add user-query \\
    --query "SELECT :name FROM users WHERE id = :id" \\
    --param-def "name:string::User name" \\
    --param-def "id:int:0:User ID" \\
    --description "Query users by name or ID" \\
    --tags "report,users" \\
    --use`,
	Args: cobra.ExactArgs(1),
	RunE: presetAddRun,
}

var (
	presetAddQuery       string
	presetAddDescription string
	presetAddConnection  string
	presetAddMaxRows     int
	presetAddParamDefs   []string
	presetAddTags        string
	presetAddUse         bool
)

func presetAddRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	name := strings.TrimSpace(args[0])
	out := cmd.OutOrStdout()

	// Validate required fields
	// VAL-CLI-003: Error MISSING_REQUIRED_FIELD if falta --query
	if presetAddQuery == "" {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"MISSING_REQUIRED_FIELD",
			"the --query flag is required",
			"Provide a query: --query \"SELECT ...\"",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Validate name is not empty
	if name == "" {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"MISSING_REQUIRED_FIELD",
			"preset name cannot be empty",
			"Provide a name: mapj protheus preset add <name> --query ...",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load existing presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-004: Error PRESET_EXISTS si nombre duplicado con hint de edit
	if existing := presetFile.GetPreset(name); existing != nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"PRESET_EXISTS",
			fmt.Sprintf("preset '%s' already exists", name),
			fmt.Sprintf("Use 'mapj protheus preset edit %s' to modify the existing preset", name),
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Detect parameters from query
	detectedParams := preset.DetectParameters(presetAddQuery)

	// Validate parameter names
	// VAL-PARAM-002: Valid Parameter Names Only
	invalidNames := preset.ValidateParamNames(detectedParams)
	if len(invalidNames) > 0 {
		var msgs []string
		for _, e := range invalidNames {
			msgs = append(msgs, e.Error())
		}
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"INVALID_PARAM_NAME",
			fmt.Sprintf("invalid parameter names detected: %s", strings.Join(msgs, "; ")),
			"Parameter names must start with a letter or underscore and contain only letters, digits, and underscores",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Parse param-def flags
	paramDefs := []preset.ParamDef{}
	for _, defStr := range presetAddParamDefs {
		def, err := parseParamDef(defStr)
		if err != nil {
			// VAL-CLI-005: Error INVALID_PARAM_DEF si formato de param-def inválido
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"INVALID_PARAM_DEF",
				fmt.Sprintf("invalid --param-def format: %s", err.Error()),
				"Format: name:type[:default][:description] (types: string, int, date, datetime, bool, list)",
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}
		paramDefs = append(paramDefs, def)
	}

	// Build parameter map from definitions for quick lookup
	defMap := make(map[string]preset.ParamDef)
	for _, def := range paramDefs {
		defMap[def.Name] = def
	}

	// Mark detected parameters that aren't defined as required by default
	finalParams := make([]preset.ParamDef, 0)
	for _, detected := range detectedParams {
		if def, exists := defMap[detected]; exists {
			// Use the defined parameter
			finalParams = append(finalParams, def)
		} else {
			// Auto-create parameter definition (required string by default)
			finalParams = append(finalParams, preset.ParamDef{
				Name:     detected,
				Type:     preset.ParamTypeString,
				Required: true,
			})
		}
	}

	// Add any defined parameters that weren't detected (for documentation purposes)
	for _, def := range paramDefs {
		found := false
		for _, detected := range detectedParams {
			if detected == def.Name {
				found = true
				break
			}
		}
		if !found {
			// Parameter defined but not found in query - still include it
			// (might be used in interpolation or documentation)
		}
	}

	// Parse tags
	tags := []string{}
	if presetAddTags != "" {
		for _, tag := range strings.Split(presetAddTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Create the preset
	now := time.Now().UTC()
	newPreset := &preset.QueryPreset{
		Name:        name,
		Description: presetAddDescription,
		Query:       presetAddQuery,
		Connection:  presetAddConnection,
		MaxRows:     presetAddMaxRows,
		Parameters:  finalParams,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to store
	presetFile.SetPreset(newPreset)

	// Set as active if --use flag
	if presetAddUse {
		presetFile.SetActivePreset(name)
	}

	if err := presetStore.Save(presetFile); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build response
	// VAL-CLI-001: Output JSON con ok:true y preset creado
	response := map[string]any{
		"name":               newPreset.Name,
		"query":              newPreset.Query,
		"detectedParameters": detectedParams,
		"parameters":         newPreset.Parameters,
		"createdAt":          newPreset.CreatedAt.Format(time.RFC3339),
		"updatedAt":          newPreset.UpdatedAt.Format(time.RFC3339),
	}

	if newPreset.Description != "" {
		response["description"] = newPreset.Description
	}
	if newPreset.Connection != "" {
		response["connection"] = newPreset.Connection
	}
	if newPreset.MaxRows > 0 {
		response["maxRows"] = newPreset.MaxRows
	}
	if len(newPreset.Tags) > 0 {
		response["tags"] = newPreset.Tags
	}
	if presetAddUse {
		response["active"] = true
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// parseParamDef parses a parameter definition string.
// Format: name:type[:default][:description]
// VAL-CLI-005: Invalid param definition returns error
func parseParamDef(s string) (preset.ParamDef, error) {
	parts := strings.Split(s, ":")

	// Must have at least name and type
	if len(parts) < 2 {
		return preset.ParamDef{}, fmt.Errorf("missing type component")
	}

	name := strings.TrimSpace(parts[0])
	paramType := strings.TrimSpace(parts[1])

	// Validate name is not empty
	if name == "" {
		return preset.ParamDef{}, fmt.Errorf("parameter name cannot be empty")
	}

	// Validate type
	if !preset.IsValidParamType(paramType) {
		return preset.ParamDef{}, fmt.Errorf("invalid type '%s', valid types: %s", paramType, strings.Join(preset.ValidParamTypes(), ", "))
	}

	// Build the ParamDef
	def := preset.ParamDef{
		Name:     name,
		Type:     paramType,
		Required: true, // Default to required
	}

	// Parse optional components
	if len(parts) >= 3 {
		def.Default = strings.TrimSpace(parts[2])
		if def.Default != "" {
			def.Required = false // Has default, so not required
		}
	}

	if len(parts) >= 4 {
		def.Description = strings.TrimSpace(parts[3])
	}

	return def, nil
}

// ======================== INIT ========================

func init() {
	protheusCmd.AddCommand(protheusPresetCmd)
	protheusPresetCmd.AddCommand(presetAddCmd)

	// Add flags for preset add command
	presetAddCmd.Flags().StringVar(&presetAddQuery, "query", "", "SQL query with :parameter placeholders (required)")
	presetAddCmd.Flags().StringVar(&presetAddDescription, "description", "", "Description of the preset")
	presetAddCmd.Flags().StringVar(&presetAddConnection, "connection", "", "Default connection profile to use")
	presetAddCmd.Flags().IntVar(&presetAddMaxRows, "max-rows", 0, "Default max rows limit (0 = no limit)")
	presetAddCmd.Flags().StringArrayVar(&presetAddParamDefs, "param-def", nil, "Parameter definition (repeatable): name:type[:default][:description]")
	presetAddCmd.Flags().StringVar(&presetAddTags, "tags", "", "Comma-separated tags")
	presetAddCmd.Flags().BoolVar(&presetAddUse, "use", false, "Set this preset as active immediately")

	// Mark query as required (for help text, actual validation done in RunE for better error messages)
	presetAddCmd.MarkFlagRequired("query")
}
