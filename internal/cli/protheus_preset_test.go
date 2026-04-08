package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mario-pereyra/mapj/internal/preset"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// testFlags holds flag values for testing
type testFlags struct {
	query       string
	description string
	connection  string
	maxRows     int
	paramDefs   []string
	tags        string
	use         bool
}

// createTestStore creates a preset store with a temp directory path.
func createTestStore(t *testing.T) *preset.PresetStore {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")
	store := &preset.PresetStore{}
	store.SetPath(testPath)
	return store
}

// executePresetAdd executes the preset add command with given args and returns the JSON output.
func executePresetAdd(t *testing.T, store *preset.PresetStore, args []string, flags testFlags) map[string]any {
	// Set output format to LLM (JSON) for testing
	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	// Set the test store
	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	// Create a new command instance for this test
	cmd := createPresetAddCmdForTest()

	// Set flag values on the command
	if flags.query != "" {
		cmd.Flags().Set("query", flags.query)
	}
	if flags.description != "" {
		cmd.Flags().Set("description", flags.description)
	}
	if flags.connection != "" {
		cmd.Flags().Set("connection", flags.connection)
	}
	if flags.maxRows > 0 {
		cmd.Flags().Set("max-rows", fmt.Sprintf("%d", flags.maxRows))
	}
	for _, pd := range flags.paramDefs {
		cmd.Flags().Set("param-def", pd)
	}
	if flags.tags != "" {
		cmd.Flags().Set("tags", flags.tags)
	}
	if flags.use {
		cmd.Flags().Set("use", "true")
	}

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(args)

	// Execute
	err := cmd.Execute()
	// Commands return nil even for "expected errors" (they output JSON errors instead)
	require.NoError(t, err, "Command execution should not return Go errors")

	// Parse JSON output
	output := strings.TrimSpace(buf.String())
	if output == "" {
		return nil
	}

	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Output should be valid JSON: %s", output)

	return result
}

// createPresetAddCmdForTest creates a fresh preset add command for testing.
// It uses local flag variables to avoid conflicts with global state.
func createPresetAddCmdForTest() *cobra.Command {
	// Use local variables for this command instance
	var query string
	var description string
	var connection string
	var maxRows int
	var paramDefs []string
	var tags string
	var use bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new query preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Copy local values to global for presetAddRun to use
			presetAddQuery = query
			presetAddDescription = description
			presetAddConnection = connection
			presetAddMaxRows = maxRows
			presetAddParamDefs = paramDefs
			presetAddTags = tags
			presetAddUse = use
			return presetAddRun(cmd, args)
		},
	}

	// Add flags using local variables
	cmd.Flags().StringVar(&query, "query", "", "SQL query with :parameter placeholders (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description of the preset")
	cmd.Flags().StringVar(&connection, "connection", "", "Default connection profile to use")
	cmd.Flags().IntVar(&maxRows, "max-rows", 0, "Default max rows limit (0 = no limit)")
	cmd.Flags().StringArrayVar(&paramDefs, "param-def", nil, "Parameter definition (repeatable): name:type[:default][:description]")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().BoolVar(&use, "use", false, "Set this preset as active immediately")

	return cmd
}

// =============================================================================
// VAL-CLI-001: Preset Add - Success
// =============================================================================

func TestPresetAddSuccess(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"test-preset"}, testFlags{
		query: "SELECT 1",
	})

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool), "Expected ok=true, got: %v", result)
	// Note: command path is "add" when testing in isolation (not "preset add")
	assert.Contains(t, result["command"], "add")

	// Verify preset was created
	presetData := result["result"].(map[string]any)
	assert.Equal(t, "test-preset", presetData["name"])
	assert.Equal(t, "SELECT 1", presetData["query"])
	assert.NotEmpty(t, presetData["createdAt"])
	assert.NotEmpty(t, presetData["updatedAt"])

	// Verify file was created
	testPath := store.GetPath()
	assert.FileExists(t, testPath)

	// Verify store contains the preset
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Contains(t, loaded.Presets, "test-preset")
}

// =============================================================================
// VAL-CLI-002: Preset Add - With Optional Fields
// =============================================================================

func TestPresetAddWithOptionalFields(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"full-preset"}, testFlags{
		query:       "SELECT :name FROM users WHERE id = :id",
		description: "Test preset with params",
		connection:  "protheus_prod",
		paramDefs:   []string{"name:string::User name", "id:int:0:User ID"},
		tags:        "report,daily",
		use:         true,
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)

	// Verify all fields
	assert.Equal(t, "full-preset", presetData["name"])
	assert.Equal(t, "Test preset with params", presetData["description"])
	assert.Equal(t, "protheus_prod", presetData["connection"])
	assert.Equal(t, "SELECT :name FROM users WHERE id = :id", presetData["query"])

	// Verify parameters
	params := presetData["parameters"].([]any)
	require.Len(t, params, 2)

	// Verify tags
	tags := presetData["tags"].([]any)
	assert.Contains(t, tags, "report")
	assert.Contains(t, tags, "daily")

	// Verify active preset set
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "full-preset", loaded.ActivePreset)
}

// =============================================================================
// VAL-CLI-003: Preset Add - Missing Required Field
// =============================================================================

func TestPresetAddMissingQuery(t *testing.T) {
	store := createTestStore(t)

	// Not providing query flag
	result := executePresetAdd(t, store, []string{"missing-query-preset"}, testFlags{})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "MISSING_REQUIRED_FIELD", errorData["code"])
	assert.Contains(t, errorData["message"], "query")
}

// =============================================================================
// VAL-CLI-004: Preset Add - Duplicate Name
// =============================================================================

func TestPresetAddDuplicateName(t *testing.T) {
	store := createTestStore(t)

	// Create initial preset
	initialFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"existing-preset": {
				Name:      "existing-preset",
				Query:     "SELECT original",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	err := store.Save(initialFile)
	require.NoError(t, err)

	// Try to add duplicate
	result := executePresetAdd(t, store, []string{"existing-preset"}, testFlags{
		query: "SELECT duplicate",
	})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "PRESET_EXISTS", errorData["code"])
	assert.Contains(t, errorData["message"], "existing-preset")

	// Verify hint mentions edit command
	hint := errorData["hint"].(string)
	assert.Contains(t, hint, "preset edit")
}

// =============================================================================
// VAL-CLI-005: Preset Add - Invalid Param Definition
// =============================================================================

func TestPresetAddInvalidParamDef(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"invalid-param-preset"}, testFlags{
		query:     "SELECT :param",
		paramDefs: []string{"invalid-format"}, // Missing type
	})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "INVALID_PARAM_DEF", errorData["code"])
	assert.Contains(t, errorData["message"], "param-def")

	// Verify hint shows correct format
	hint := errorData["hint"].(string)
	assert.Contains(t, hint, "name:type")
}

func TestPresetAddInvalidParamType(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"invalid-type-preset"}, testFlags{
		query:     "SELECT :param",
		paramDefs: []string{"param:invalidtype"}, // Invalid type
	})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "INVALID_PARAM_DEF", errorData["code"])
}

// =============================================================================
// Additional Tests
// =============================================================================

func TestPresetAddDetectsParameters(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"auto-detect-params"}, testFlags{
		query: "SELECT :name, :age FROM users WHERE :active = 1",
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)

	// Verify detected parameters
	detectedParams := presetData["detectedParameters"].([]any)
	assert.Contains(t, detectedParams, "name")
	assert.Contains(t, detectedParams, "age")
	assert.Contains(t, detectedParams, "active")
}

func TestPresetAddParameterDetectionWithInvalidNames(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"invalid-param-names"}, testFlags{
		query: "SELECT :valid_param, :invalid-param FROM table",
	})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "INVALID_PARAM_NAME", errorData["code"])
}

func TestPresetAddTagsParsing(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"tags-preset"}, testFlags{
		query: "SELECT 1",
		tags:  "tag1,tag2,tag3",
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)

	tags := presetData["tags"].([]any)
	require.Len(t, tags, 3)
	assert.Contains(t, tags, "tag1")
	assert.Contains(t, tags, "tag2")
	assert.Contains(t, tags, "tag3")
}

func TestPresetAddUseFlag(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"active-preset"}, testFlags{
		query: "SELECT 1",
		use:   true,
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)
	assert.True(t, presetData["active"].(bool))
}

func TestPresetAddEmptyName(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{""}, testFlags{
		query: "SELECT 1",
	})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "MISSING_REQUIRED_FIELD", errorData["code"])
	assert.Contains(t, errorData["message"], "name")
}

func TestPresetAddTimestamps(t *testing.T) {
	store := createTestStore(t)
	startTime := time.Now()

	result := executePresetAdd(t, store, []string{"timestamp-preset"}, testFlags{
		query: "SELECT 1",
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)

	// Verify timestamps are set and within reasonable range
	createdAtStr := presetData["createdAt"].(string)
	updatedAtStr := presetData["updatedAt"].(string)

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	require.NoError(t, err)
	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	require.NoError(t, err)

	// Timestamps should be recent (within test execution time)
	assert.True(t, createdAt.After(startTime.Add(-1*time.Second)))
	assert.True(t, updatedAt.After(startTime.Add(-1*time.Second)))
	assert.True(t, createdAt.Equal(updatedAt) || updatedAt.After(createdAt))
}

func TestPresetAddMaxRowsFlag(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"maxrows-preset"}, testFlags{
		query:   "SELECT 1",
		maxRows: 1000,
	})

	require.NotNil(t, result)
	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)
	assert.Equal(t, float64(1000), presetData["maxRows"].(float64))
}

// =============================================================================
// ParamDef Parsing Tests
// =============================================================================

func TestParseParamDefValid(t *testing.T) {
	tests := []struct {
		input    string
		expected preset.ParamDef
	}{
		{
			input: "name:string",
			expected: preset.ParamDef{
				Name:     "name",
				Type:     "string",
				Required: true,
			},
		},
		{
			input: "id:int:0",
			expected: preset.ParamDef{
				Name:     "id",
				Type:     "int",
				Required: false,
				Default:  "0",
			},
		},
		{
			input: "date:date::",
			expected: preset.ParamDef{
				Name:     "date",
				Type:     "date",
				Required: true,
			},
		},
		{
			input: "name:string::User name",
			expected: preset.ParamDef{
				Name:        "name",
				Type:        "string",
				Required:    true,
				Description: "User name",
			},
		},
		{
			input: "status:bool:true:Is active",
			expected: preset.ParamDef{
				Name:        "status",
				Type:        "bool",
				Required:    false,
				Default:     "true",
				Description: "Is active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseParamDef(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Required, result.Required)
			assert.Equal(t, tt.expected.Default, result.Default)
			assert.Equal(t, tt.expected.Description, result.Description)
		})
	}
}

func TestParseParamDefInvalid(t *testing.T) {
	tests := []string{
		"",                 // Empty
		"name",             // Missing type
		"name:",            // Empty type
		":string",          // Empty name
		"name:invalidtype", // Invalid type
	}

	for _, input := range tests {
		t.Run(fmt.Sprintf("input_%q", input), func(t *testing.T) {
			_, err := parseParamDef(input)
			assert.Error(t, err)
		})
	}
}

// =============================================================================
// Output Format Tests
// =============================================================================

func TestPresetAddOutputFormat(t *testing.T) {
	store := createTestStore(t)

	result := executePresetAdd(t, store, []string{"format-preset"}, testFlags{
		query: "SELECT 1",
	})

	require.NotNil(t, result)
	// Verify envelope structure
	assert.Contains(t, result, "ok")
	assert.Contains(t, result, "command")
	assert.Contains(t, result, "result")
}

func TestPresetAddErrorFormat(t *testing.T) {
	store := createTestStore(t)

	// Missing --query
	result := executePresetAdd(t, store, []string{"error-preset"}, testFlags{})

	require.NotNil(t, result)
	// Verify error envelope structure
	assert.Contains(t, result, "ok")
	assert.Contains(t, result, "command")
	assert.Contains(t, result, "error")

	errorData := result["error"].(map[string]any)
	assert.Contains(t, errorData, "code")
	assert.Contains(t, errorData, "message")
}
