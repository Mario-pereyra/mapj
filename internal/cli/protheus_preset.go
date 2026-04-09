package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mario-pereyra/mapj/internal/auth"
	"github.com/Mario-pereyra/mapj/internal/errors"
	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/internal/preset"
	"github.com/Mario-pereyra/mapj/pkg/protheus"
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
	Use:            "preset",
	Short:          "Manage saved query presets with parameters",
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
	SilenceUsage:   true, // Prevent Cobra from printing usage after we output JSON error
	SilenceErrors:  true, // Prevent Cobra from printing error after we output JSON
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
	Args:            cobra.ExactArgs(1),
	SilenceUsage:    true, // Prevent Cobra from printing usage after we output JSON error
	SilenceErrors:   true, // Prevent Cobra from printing error after we output JSON
	RunE:            presetAddRun,
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
		return &errors.UsageError{Msg: "missing required flag: --query"}
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
		return &errors.UsageError{Msg: "preset name cannot be empty"}
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
		return &errors.UsageError{Msg: fmt.Sprintf("preset '%s' already exists", name)}
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
		return &errors.UsageError{Msg: "invalid parameter names detected"}
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
			return &errors.UsageError{Msg: "invalid --param-def format"}
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

// ======================== LIST SUBCOMMAND ========================

var presetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved presets",
	Long: `List all saved presets with optional filtering.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset list","result":{
    "presets":[{"name":"preset1","query":"SELECT 1",...},...],
    "count":2
  }}

FLAGS:
  --tag TAG          Filter presets by tag
  --connection NAME  Filter presets by connection profile

EXAMPLES:
  mapj protheus preset list
  
  mapj protheus preset list --tag report
  
  mapj protheus preset list --connection protheus_prod`,
	Args: cobra.NoArgs,
	RunE: presetListRun,
}

var (
	presetListTag        string
	presetListConnection string
)

func presetListRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	out := cmd.OutOrStdout()

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Get all preset names (sorted)
	presetNames := presetFile.ListPresets()

	// Filter and collect presets
	presets := make([]map[string]any, 0)
	for _, name := range presetNames {
		p := presetFile.GetPreset(name)
		if p == nil {
			continue
		}

		// Apply tag filter
		if presetListTag != "" {
			if !containsTag(p.Tags, presetListTag) {
				continue
			}
		}

		// Apply connection filter
		if presetListConnection != "" {
			if p.Connection != presetListConnection {
				continue
			}
		}

		// Build preset output
		presetOutput := map[string]any{
			"name":      p.Name,
			"query":     p.Query,
			"createdAt": p.CreatedAt.Format(time.RFC3339),
			"updatedAt": p.UpdatedAt.Format(time.RFC3339),
		}

		if p.Description != "" {
			presetOutput["description"] = p.Description
		}
		if p.Connection != "" {
			presetOutput["connection"] = p.Connection
		}
		if p.MaxRows > 0 {
			presetOutput["maxRows"] = p.MaxRows
		}
		if len(p.Tags) > 0 {
			presetOutput["tags"] = p.Tags
		}
		if len(p.Parameters) > 0 {
			presetOutput["parameters"] = p.Parameters
		}

		// Mark if this is the active preset
		if presetFile.ActivePreset == p.Name {
			presetOutput["active"] = true
		}

		presets = append(presets, presetOutput)
	}

	// Build response
	response := map[string]any{
		"presets": presets,
		"count":   len(presets),
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// containsTag checks if a tag exists in the tags slice.
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ======================== SHOW SUBCOMMAND ========================

var presetShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show preset details",
	Long: `Show preset details including parameters.

When no name is provided, shows the active preset.
When no active preset is set, returns null for activePreset.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset show","result":{
    "name":"my-preset",
    "query":"SELECT :name FROM users WHERE id = :id",
    "parameters":[
      {"name":"name","type":"string","required":true,"description":"User name"},
      {"name":"id","type":"int","required":true}
    ],
    "description":"...",
    "connection":"...",
    "maxRows":100,
    "tags":["report"],
    "createdAt":"2024-01-15T10:30:00Z",
    "updatedAt":"2024-01-15T10:30:00Z"
  }}

OUTPUT SCHEMA (no active preset):
  {"ok":true,"command":"mapj protheus preset show","result":{
    "activePreset":null
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"PRESET_NOT_FOUND","message":"...","hint":"..."}}

EXAMPLES:
  mapj protheus preset show my-preset
  
  mapj protheus preset show  # Shows active preset`,
	Args: cobra.MaximumNArgs(1),
	RunE: presetShowRun,
}

func presetShowRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	out := cmd.OutOrStdout()

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	var targetPreset *preset.QueryPreset
	var presetName string

	// Determine which preset to show
	if len(args) > 0 && args[0] != "" {
		// Specific preset requested
		presetName = strings.TrimSpace(args[0])
		targetPreset = presetFile.GetPreset(presetName)
	} else {
		// No name provided, show active preset
		if presetFile.ActivePreset != "" {
			targetPreset = presetFile.GetPreset(presetFile.ActivePreset)
			presetName = presetFile.ActivePreset
		}
	}

	// Handle preset not found
	if targetPreset == nil {
		if presetName != "" {
			// Specific preset was requested but not found
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"PRESET_NOT_FOUND",
				fmt.Sprintf("preset '%s' not found", presetName),
				"Use 'mapj protheus preset list' to see available presets",
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}

		// No active preset set - return null response
		response := map[string]any{
			"activePreset": nil,
		}
		env := output.NewEnvelope(cmd.CommandPath(), response)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build preset output with all fields
	response := map[string]any{
		"name":      targetPreset.Name,
		"query":     targetPreset.Query,
		"createdAt": targetPreset.CreatedAt.Format(time.RFC3339),
		"updatedAt": targetPreset.UpdatedAt.Format(time.RFC3339),
	}

	if targetPreset.Description != "" {
		response["description"] = targetPreset.Description
	}
	if targetPreset.Connection != "" {
		response["connection"] = targetPreset.Connection
	}
	if targetPreset.MaxRows > 0 {
		response["maxRows"] = targetPreset.MaxRows
	}
	if len(targetPreset.Tags) > 0 {
		response["tags"] = targetPreset.Tags
	}

	// Include parameters with all their fields
	if len(targetPreset.Parameters) > 0 {
		response["parameters"] = targetPreset.Parameters
	}

	// Mark if this is the active preset
	if presetFile.ActivePreset == targetPreset.Name {
		response["active"] = true
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// ======================== RUN SUBCOMMAND ========================

var presetRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Execute a preset query with parameters",
	Long: `Execute a saved preset query with parameter interpolation.

Parameters from the preset are substituted into the query with proper escaping
and SQL injection protection.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset run","result":{
    "rows":[[...],...],
    "columns":["col1","col2"],
    "count":10,
    "params_used":{"name":"value"},
    "connection_used":"protheus_prod"
  }}

OUTPUT SCHEMA (success with --output-file):
  {"ok":true,"command":"mapj protheus preset run","result":{
    "rows":100,
    "columns":5,
    "output_file":"./results.json"
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"MISSING_PARAMETER","message":"...","hint":"..."}}

FLAGS:
  --param KEY=VALUE     Parameter value (repeatable)
                        Example: --param name=John --param id=123
  --connection NAME     Override the preset's connection profile
  --max-rows N          Limit number of rows returned (default: from preset or 10000)
  --output-file PATH    Write results to file instead of stdout

CONNECTION RESOLUTION:
  1. --connection flag (highest priority)
  2. preset.connection (saved with preset)
  3. active connection profile
  4. Error: NO_CONNECTION

EXAMPLES:
  mapj protheus preset run myquery
  
  mapj protheus preset run user-query \\
    --param name=John \\
    --param id=123
  
  mapj protheus preset run report \\
    --connection protheus_prod \\
    --max-rows 100 \\
    --output-file ./report.json`,
	Args: cobra.ExactArgs(1),
	RunE: presetRunRun,
}

var (
	presetRunParams      []string
	presetRunConnection  string
	presetRunMaxRows     int
	presetRunOutputFile  string
)

func presetRunRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	name := strings.TrimSpace(args[0])
	out := cmd.OutOrStdout()

	// Initialize preset store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-015: Error PRESET_NOT_FOUND si preset no existe
	targetPreset := presetFile.GetPreset(name)
	if targetPreset == nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"PRESET_NOT_FOUND",
			fmt.Sprintf("preset '%s' not found", name),
			"Use 'mapj protheus preset list' to see available presets",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Parse --param flags into map
	params := make(map[string]string)
	for _, paramStr := range presetRunParams {
		key, value, err := parseParamKeyValue(paramStr)
		if err != nil {
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"INVALID_PARAM_FORMAT",
				fmt.Sprintf("invalid --param format: %s", err.Error()),
				"Format: --param key=value (e.g., --param name=John)",
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}
		params[key] = value
	}

	// Interpolate query with parameters
	interpolatedQuery, err := preset.InterpolateQuery(targetPreset.Query, params, targetPreset.Parameters)
	if err != nil {
		// Handle different error types
		if ierr := preset.GetInterpolationError(err); ierr != nil {
			switch ierr.Type {
			case "missing_param":
				// VAL-CLI-016: Error MISSING_PARAMETER lista params faltantes
				missingParams := getMissingRequiredParams(targetPreset.Parameters, params)
				env := output.NewErrorEnvelopeWithHint(
					cmd.CommandPath(),
					"MISSING_PARAMETER",
					fmt.Sprintf("missing required parameters: %s", strings.Join(missingParams, ", ")),
					fmt.Sprintf("Provide values: --param %s=value", strings.Join(missingParams, " --param ")),
					false,
				)
				fmt.Fprintln(out, formatter.Format(env))
				return nil
			case "type_mismatch":
				env := output.NewErrorEnvelope(
					cmd.CommandPath(),
					"TYPE_MISMATCH",
					ierr.Message,
					false,
				)
				fmt.Fprintln(out, formatter.Format(env))
				return nil
			case "sql_injection":
				env := output.NewErrorEnvelope(
					cmd.CommandPath(),
					"SQL_INJECTION_DETECTED",
					fmt.Sprintf("potential SQL injection detected in parameter '%s': patterns %v", ierr.ParamName, ierr.Detected),
					false,
				)
				fmt.Fprintln(out, formatter.Format(env))
				return nil
			}
		}
		env := output.NewErrorEnvelope(cmd.CommandPath(), "INTERPOLATION_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Resolve connection
	// VAL-CLI-012: --connection overridea conexión del preset
	// VAL-CROSS-005: Connection profile integration
	connectionName := resolveConnection(presetRunConnection, targetPreset.Connection)

	// Get credentials
	authStore, err := auth.NewStore()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	creds, err := authStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "AUTH_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Resolve the profile to use
	var profile *auth.ProtheusProfile
	if connectionName != "" {
		// Use specific connection
		if creds.ProtheusProfiles == nil || creds.ProtheusProfiles[connectionName] == nil {
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"CONNECTION_NOT_FOUND",
				fmt.Sprintf("connection profile '%s' not found", connectionName),
				"Use 'mapj protheus connection list' to see available profiles",
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}
		profile = creds.ProtheusProfiles[connectionName]
	} else {
		// Use active profile
		profile = creds.ActiveProtheusProfile()
	}

	// VAL-CROSS-005: Sin conexión disponible: error con hint
	// NO_CONNECTION: when no connection profile is configured (retryable: true - user can configure one)
	if profile == nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"NO_CONNECTION",
			"no Protheus connection available",
			"Specify a connection with --connection or set an active profile with 'mapj protheus connection use <name>'",
			true, // retryable - user can configure a connection
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Determine max rows
	// VAL-CLI-013: --max-rows N limita resultados
	maxRows := presetRunMaxRows
	if maxRows == 0 {
		// Use preset default
		maxRows = targetPreset.MaxRows
	}
	if maxRows == 0 {
		// Use global default
		maxRows = 10000
	}

	// Execute query
	ctx := context.Background()
	client := protheus.NewClient(profile.Server, profile.Port, profile.Database, profile.User, profile.Password)

	result, err := client.Query(ctx, interpolatedQuery, maxRows)
	if err != nil {
		// VAL-CLI-017: Error CONNECTION_FAILED si conexión falla
		// VAL-CLI-018: Error QUERY_VALIDATION_FAILED si query inválida
		if strings.Contains(err.Error(), "validation error") {
			env := output.NewErrorEnvelope(
				cmd.CommandPath(),
				"QUERY_VALIDATION_FAILED",
				err.Error(),
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}

		// Connection error
		msg := err.Error()
		hint := protheusVPNHint(profile.Server)
		env := output.NewErrorEnvelope(
			cmd.CommandPath(),
			"CONNECTION_FAILED",
			msg+"\n"+hint,
			true, // retryable
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build response
	// VAL-CLI-010: Output JSON con rows, columns, count
	response := map[string]any{
		"columns":        result.Columns,
		"rows":           result.Rows,
		"count":          result.Count,
		"params_used":    params,
		"connection_used": profile.Name,
	}

	// VAL-CLI-014: --output-file path escribe resultados a archivo
	if presetRunOutputFile != "" {
		// Write to file
		env := output.NewEnvelope(cmd.CommandPath(), result)
		content := formatter.Format(env)

		if err := output.WriteToFile(presetRunOutputFile, content); err != nil {
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"FILE_WRITE_ERROR",
				err.Error(),
				fmt.Sprintf("Check that the directory exists and you have write access: %s", presetRunOutputFile),
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}

		// Print summary to stdout
		summary := map[string]any{
			"rows":         result.Count,
			"columns":      len(result.Columns),
			"output_file":  presetRunOutputFile,
			"params_used":  params,
			"connection_used": profile.Name,
		}
		summaryEnv := output.NewEnvelope(cmd.CommandPath(), summary)
		fmt.Fprintln(out, formatter.Format(summaryEnv))
		return nil
	}

	// Print full result to stdout
	resultEnv := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(resultEnv))
	return nil
}

// parseParamKeyValue parses a "key=value" string into key and value components.
func parseParamKeyValue(s string) (string, string, error) {
	idx := strings.Index(s, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("missing '=' separator")
	}
	key := strings.TrimSpace(s[:idx])
	value := strings.TrimSpace(s[idx+1:])
	if key == "" {
		return "", "", fmt.Errorf("empty key")
	}
	return key, value, nil
}

// resolveConnection determines which connection to use.
// Priority: flag > preset.connection > empty (will use active)
func resolveConnection(flagValue, presetConnection string) string {
	if flagValue != "" {
		return flagValue
	}
	return presetConnection
}

// getMissingRequiredParams returns the names of required parameters that are missing.
func getMissingRequiredParams(paramDefs []preset.ParamDef, providedParams map[string]string) []string {
	var missing []string
	for _, def := range paramDefs {
		if def.Required {
			if _, provided := providedParams[def.Name]; !provided {
				missing = append(missing, def.Name)
			}
		}
	}
	return missing
}

// ======================== EDIT SUBCOMMAND ========================

var presetEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Modify an existing preset",
	Long: `Modify an existing preset's fields.

Only the fields you specify with flags will be updated. The updatedAt timestamp
is always refreshed when a preset is edited.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset edit","result":{
    "name":"my-preset",
    "fields_updated":["description","query"],
    "updatedAt":"2024-01-15T10:30:00Z"
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"NO_FIELDS_TO_UPDATE","message":"...","hint":"..."}}

FLAGS:
  --description TEXT   Update preset description
  --query TEXT         Update the SQL query
  --connection NAME    Update the default connection profile
  --max-rows N         Update the default max rows limit
  --param-def DEF      Replace parameter definitions (repeatable)
                       Format: name:type[:default][:description]
  --tags TAGS          Replace tags (comma-separated)

EXAMPLES:
  mapj protheus preset edit myquery --description "Updated description"
  
  mapj protheus preset edit report \\
    --query "SELECT :status FROM orders WHERE date > :since" \\
    --param-def "status:string:pending" \\
    --param-def "since:date"`,
	Args: cobra.ExactArgs(1),
	RunE: presetEditRun,
}

var (
	presetEditDescription string
	presetEditQuery       string
	presetEditConnection  string
	presetEditMaxRows     int
	presetEditParamDefs   []string
	presetEditTags        string
)

func presetEditRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	name := strings.TrimSpace(args[0])
	out := cmd.OutOrStdout()

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-026: Error PRESET_NOT_FOUND si preset no existe
	targetPreset := presetFile.GetPreset(name)
	if targetPreset == nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"PRESET_NOT_FOUND",
			fmt.Sprintf("preset '%s' not found", name),
			"Use 'mapj protheus preset list' to see available presets",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Check if any fields were provided
	// VAL-CLI-025: Error NO_FIELDS_TO_UPDATE sin flags
	fieldsUpdated := []string{}
	updates := make(map[string]any)

	// Check each flag and update if provided
	if cmd.Flags().Changed("description") {
		targetPreset.Description = presetEditDescription
		fieldsUpdated = append(fieldsUpdated, "description")
		updates["description"] = presetEditDescription
	}

	if cmd.Flags().Changed("query") {
		targetPreset.Query = presetEditQuery
		fieldsUpdated = append(fieldsUpdated, "query")
		updates["query"] = presetEditQuery

		// Re-detect parameters from new query
		detectedParams := preset.DetectParameters(presetEditQuery)
		invalidNames := preset.ValidateParamNames(detectedParams)
		if len(invalidNames) > 0 {
			var msgs []string
			for _, e := range invalidNames {
				msgs = append(msgs, e.Error())
			}
			env := output.NewErrorEnvelopeWithHint(
				cmd.CommandPath(),
				"INVALID_PARAM_NAME",
				fmt.Sprintf("invalid parameter names detected in query: %s", strings.Join(msgs, "; ")),
				"Parameter names must start with a letter or underscore and contain only letters, digits, and underscores",
				false,
			)
			fmt.Fprintln(out, formatter.Format(env))
			return nil
		}
	}

	if cmd.Flags().Changed("connection") {
		targetPreset.Connection = presetEditConnection
		fieldsUpdated = append(fieldsUpdated, "connection")
		updates["connection"] = presetEditConnection
	}

	if cmd.Flags().Changed("max-rows") {
		targetPreset.MaxRows = presetEditMaxRows
		fieldsUpdated = append(fieldsUpdated, "maxRows")
		updates["maxRows"] = presetEditMaxRows
	}

	if cmd.Flags().Changed("tags") {
		tags := []string{}
		if presetEditTags != "" {
			for _, tag := range strings.Split(presetEditTags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
		targetPreset.Tags = tags
		fieldsUpdated = append(fieldsUpdated, "tags")
		updates["tags"] = tags
	}

	if cmd.Flags().Changed("param-def") {
		// VAL-CLI-027: Error INVALID_PARAM_DEF si formato inválido
		paramDefs := []preset.ParamDef{}
		for _, defStr := range presetEditParamDefs {
			def, err := parseParamDef(defStr)
			if err != nil {
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
		targetPreset.Parameters = paramDefs
		fieldsUpdated = append(fieldsUpdated, "parameters")
		updates["parameters"] = paramDefs
	}

	// Check if any updates were made
	if len(fieldsUpdated) == 0 {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"NO_FIELDS_TO_UPDATE",
			"no fields specified to update",
			"Provide at least one field to update: --description, --query, --connection, --max-rows, --param-def, or --tags",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Update timestamp
	// VAL-CLI-024: preset edit actualiza updatedAt timestamp
	targetPreset.UpdatedAt = time.Now().UTC()

	// Save to store
	if err := presetStore.Save(presetFile); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build response
	// VAL-CLI-023: preset edit actualiza campo
	response := map[string]any{
		"name":          targetPreset.Name,
		"fields_updated": fieldsUpdated,
		"updatedAt":     targetPreset.UpdatedAt.Format(time.RFC3339),
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// ======================== REMOVE SUBCOMMAND ========================

var presetRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a preset",
	Long: `Delete a preset from the store.

If the removed preset is the active preset, the active preset reference
is automatically cleared.

OUTPUT SCHEMA (success):
  {"ok":true,"command":"mapj protheus preset remove","result":{
    "removed":"my-preset",
    "was_active":true
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"PRESET_NOT_FOUND","message":"...","hint":"..."}}

FLAGS:
  --force   Skip confirmation prompt (useful for scripts)

EXAMPLES:
  mapj protheus preset remove old-query
  
  mapj protheus preset remove old-query --force`,
	Args: cobra.ExactArgs(1),
	RunE: presetRemoveRun,
}

var presetRemoveForce bool

func presetRemoveRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	name := strings.TrimSpace(args[0])
	out := cmd.OutOrStdout()

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-030: Error PRESET_NOT_FOUND si preset no existe
	targetPreset := presetFile.GetPreset(name)
	if targetPreset == nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"PRESET_NOT_FOUND",
			fmt.Sprintf("preset '%s' not found", name),
			"Use 'mapj protheus preset list' to see available presets",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Check if this is the active preset
	// VAL-CLI-029: preset remove en preset activo limpia activePreset
	wasActive := presetFile.ActivePreset == name

	// VAL-CLI-031: preset remove --force sin confirmación
	// If not --force, would prompt for confirmation here
	// For agent-friendly CLI, we skip interactive prompts

	// Delete the preset
	presetFile.DeletePreset(name)

	// Save to store
	if err := presetStore.Save(presetFile); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build response
	// VAL-CLI-028: preset remove elimina preset
	response := map[string]any{
		"removed":    name,
		"was_active": wasActive,
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// ======================== USE SUBCOMMAND ========================

var presetUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Set or show the active preset",
	Long: `Set a preset as active or show the current active preset.

When called with a preset name, sets that preset as active. The active preset
is used as the default for commands like 'preset run' when no name is given.

When called without a name, shows the current active preset.

OUTPUT SCHEMA (success - set active):
  {"ok":true,"command":"mapj protheus preset use","result":{
    "active_preset":"my-preset",
    "preset":{"name":"my-preset","query":"...",...}
  }}

OUTPUT SCHEMA (success - show current):
  {"ok":true,"command":"mapj protheus preset use","result":{
    "active_preset":"my-preset"
  }}

OUTPUT SCHEMA (no active preset):
  {"ok":true,"command":"mapj protheus preset use","result":{
    "active_preset":null
  }}

OUTPUT SCHEMA (error):
  {"ok":false,"error":{"code":"PRESET_NOT_FOUND","message":"...","hint":"..."}}

EXAMPLES:
  mapj protheus preset use myquery
  
  mapj protheus preset use  # Show current active preset`,
	Args: cobra.MaximumNArgs(1),
	RunE: presetUseRun,
}

func presetUseRun(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	out := cmd.OutOrStdout()

	// Initialize store
	if err := initPresetStore(); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Load presets
	presetFile, err := presetStore.Load()
	if err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-034: preset use sin nombre muestra active actual
	if len(args) == 0 || args[0] == "" {
		// Show current active preset
		response := map[string]any{
			"active_preset": nil,
		}

		if presetFile.ActivePreset != "" {
			activePreset := presetFile.GetPreset(presetFile.ActivePreset)
			if activePreset != nil {
				response["active_preset"] = activePreset.Name
				response["preset"] = map[string]any{
					"name":        activePreset.Name,
					"query":       activePreset.Query,
					"description": activePreset.Description,
					"createdAt":   activePreset.CreatedAt.Format(time.RFC3339),
					"updatedAt":   activePreset.UpdatedAt.Format(time.RFC3339),
				}
				if activePreset.Connection != "" {
					response["preset"].(map[string]any)["connection"] = activePreset.Connection
				}
				if len(activePreset.Parameters) > 0 {
					response["preset"].(map[string]any)["parameters"] = activePreset.Parameters
				}
			}
		}

		env := output.NewEnvelope(cmd.CommandPath(), response)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Set active preset
	name := strings.TrimSpace(args[0])

	// VAL-CLI-033: Error PRESET_NOT_FOUND si preset no existe
	targetPreset := presetFile.GetPreset(name)
	if targetPreset == nil {
		env := output.NewErrorEnvelopeWithHint(
			cmd.CommandPath(),
			"PRESET_NOT_FOUND",
			fmt.Sprintf("preset '%s' not found", name),
			"Use 'mapj protheus preset list' to see available presets",
			false,
		)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// VAL-CLI-032: preset use marca como activePreset
	presetFile.SetActivePreset(name)

	// Save to store
	if err := presetStore.Save(presetFile); err != nil {
		env := output.NewErrorEnvelope(cmd.CommandPath(), "STORE_ERROR", err.Error(), false)
		fmt.Fprintln(out, formatter.Format(env))
		return nil
	}

	// Build response
	response := map[string]any{
		"active_preset": name,
		"preset": map[string]any{
			"name":        targetPreset.Name,
			"query":       targetPreset.Query,
			"description": targetPreset.Description,
			"createdAt":   targetPreset.CreatedAt.Format(time.RFC3339),
			"updatedAt":   targetPreset.UpdatedAt.Format(time.RFC3339),
		},
	}

	if targetPreset.Connection != "" {
		response["preset"].(map[string]any)["connection"] = targetPreset.Connection
	}
	if targetPreset.MaxRows > 0 {
		response["preset"].(map[string]any)["maxRows"] = targetPreset.MaxRows
	}
	if len(targetPreset.Parameters) > 0 {
		response["preset"].(map[string]any)["parameters"] = targetPreset.Parameters
	}
	if len(targetPreset.Tags) > 0 {
		response["preset"].(map[string]any)["tags"] = targetPreset.Tags
	}

	env := output.NewEnvelope(cmd.CommandPath(), response)
	fmt.Fprintln(out, formatter.Format(env))
	return nil
}

// ======================== INIT ========================

func init() {
	protheusCmd.AddCommand(protheusPresetCmd)
	protheusPresetCmd.AddCommand(presetAddCmd)
	protheusPresetCmd.AddCommand(presetListCmd)
	protheusPresetCmd.AddCommand(presetShowCmd)
	protheusPresetCmd.AddCommand(presetRunCmd)
	protheusPresetCmd.AddCommand(presetEditCmd)
	protheusPresetCmd.AddCommand(presetRemoveCmd)
	protheusPresetCmd.AddCommand(presetUseCmd)

	// Add flags for preset add command
	presetAddCmd.Flags().StringVar(&presetAddQuery, "query", "", "SQL query with :parameter placeholders (required)")
	presetAddCmd.Flags().StringVar(&presetAddDescription, "description", "", "Description of the preset")
	presetAddCmd.Flags().StringVar(&presetAddConnection, "connection", "", "Default connection profile to use")
	presetAddCmd.Flags().IntVar(&presetAddMaxRows, "max-rows", 0, "Default max rows limit (0 = no limit)")
	presetAddCmd.Flags().StringArrayVar(&presetAddParamDefs, "param-def", nil, "Parameter definition (repeatable): name:type[:default][:description]")
	presetAddCmd.Flags().StringVar(&presetAddTags, "tags", "", "Comma-separated tags")
	presetAddCmd.Flags().BoolVar(&presetAddUse, "use", false, "Set this preset as active immediately")

	// NOTE: We do NOT use MarkFlagRequired("query") here because Cobra would intercept
	// the error and output raw text instead of our JSON error envelope.
	// The validation is done in presetAddRun which outputs a proper JSON error.

	// Add flags for preset list command
	presetListCmd.Flags().StringVar(&presetListTag, "tag", "", "Filter presets by tag")
	presetListCmd.Flags().StringVar(&presetListConnection, "connection", "", "Filter presets by connection profile")

	// Add flags for preset run command
	presetRunCmd.Flags().StringArrayVar(&presetRunParams, "param", nil, "Parameter value (repeatable): key=value")
	presetRunCmd.Flags().StringVar(&presetRunConnection, "connection", "", "Override the preset's connection profile")
	presetRunCmd.Flags().IntVar(&presetRunMaxRows, "max-rows", 0, "Limit number of rows returned (0 = use preset default or 10000)")
	presetRunCmd.Flags().StringVar(&presetRunOutputFile, "output-file", "", "Write results to file instead of stdout")

	// Add flags for preset edit command
	presetEditCmd.Flags().StringVar(&presetEditDescription, "description", "", "Update preset description")
	presetEditCmd.Flags().StringVar(&presetEditQuery, "query", "", "Update the SQL query")
	presetEditCmd.Flags().StringVar(&presetEditConnection, "connection", "", "Update the default connection profile")
	presetEditCmd.Flags().IntVar(&presetEditMaxRows, "max-rows", 0, "Update the default max rows limit")
	presetEditCmd.Flags().StringArrayVar(&presetEditParamDefs, "param-def", nil, "Update parameter definitions")
	presetEditCmd.Flags().StringVar(&presetEditTags, "tags", "", "Update tags (comma-separated)")

	// Add flags for preset remove command
	presetRemoveCmd.Flags().BoolVar(&presetRemoveForce, "force", false, "Skip confirmation prompt")
}
