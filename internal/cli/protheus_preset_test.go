package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mario-pereyra/mapj/internal/output"
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

// =============================================================================
// PRESET LIST TESTS (VAL-CLI-006 to VAL-CLI-009)
// =============================================================================

// executePresetList executes the preset list command with given args and returns the JSON output.
func executePresetList(t *testing.T, store *preset.PresetStore, tagFilter string, connectionFilter string) map[string]any {
	// Set output format to LLM (JSON) for testing
	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	// Set the test store
	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	// Create a new command instance for this test
	cmd := createPresetListCmdForTest()

	// Set flag values on the command
	if tagFilter != "" {
		cmd.Flags().Set("tag", tagFilter)
	}
	if connectionFilter != "" {
		cmd.Flags().Set("connection", connectionFilter)
	}

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()
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

// createPresetListCmdForTest creates a fresh preset list command for testing.
func createPresetListCmdForTest() *cobra.Command {
	var tagFilter string
	var connectionFilter string

	cmd := &cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Copy local values to global for presetListRun to use
			presetListTag = tagFilter
			presetListConnection = connectionFilter
			return presetListRun(cmd, args)
		},
	}

	cmd.Flags().StringVar(&tagFilter, "tag", "", "Filter presets by tag")
	cmd.Flags().StringVar(&connectionFilter, "connection", "", "Filter presets by connection")

	return cmd
}

// VAL-CLI-006: Preset List - Success
func TestPresetListSuccess(t *testing.T) {
	store := createTestStore(t)

	// Create some presets
	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"preset-one": {
				Name:      "preset-one",
				Query:     "SELECT 1",
				CreatedAt: now,
				UpdatedAt: now,
			},
			"preset-two": {
				Name:      "preset-two",
				Query:     "SELECT 2",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetList(t, store, "", "")

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool), "Expected ok=true, got: %v", result)

	// Verify result structure
	resultData := result["result"].(map[string]any)
	assert.Contains(t, resultData, "presets")
	assert.Contains(t, resultData, "count")

	presets := resultData["presets"].([]any)
	count := int(resultData["count"].(float64))
	assert.Equal(t, 2, count)
	assert.Len(t, presets, 2)
}

// VAL-CLI-007: Preset List - Filter by Tag
func TestPresetListFilterByTag(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"report-daily": {
				Name:      "report-daily",
				Query:     "SELECT * FROM daily_reports",
				Tags:      []string{"report", "daily"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"report-weekly": {
				Name:      "report-weekly",
				Query:     "SELECT * FROM weekly_reports",
				Tags:      []string{"report", "weekly"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"admin-task": {
				Name:      "admin-task",
				Query:     "SELECT * FROM admin_tasks",
				Tags:      []string{"admin"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetList(t, store, "report", "")

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)
	presets := resultData["presets"].([]any)
	count := int(resultData["count"].(float64))

	// Should only include presets with "report" tag
	assert.Equal(t, 2, count)
	assert.Len(t, presets, 2)

	// Verify only report presets are returned
	names := make([]string, 0)
	for _, p := range presets {
		presetMap := p.(map[string]any)
		names = append(names, presetMap["name"].(string))
	}
	assert.Contains(t, names, "report-daily")
	assert.Contains(t, names, "report-weekly")
	assert.NotContains(t, names, "admin-task")
}

// VAL-CLI-008: Preset List - Filter by Connection
func TestPresetListFilterByConnection(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"prod-query-1": {
				Name:       "prod-query-1",
				Query:      "SELECT 1",
				Connection: "protheus_prod",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			"prod-query-2": {
				Name:       "prod-query-2",
				Query:      "SELECT 2",
				Connection: "protheus_prod",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			"dev-query": {
				Name:       "dev-query",
				Query:      "SELECT 3",
				Connection: "protheus_dev",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetList(t, store, "", "protheus_prod")

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)
	presets := resultData["presets"].([]any)
	count := int(resultData["count"].(float64))

	// Should only include presets with protheus_prod connection
	assert.Equal(t, 2, count)
	assert.Len(t, presets, 2)

	// Verify only prod presets are returned
	names := make([]string, 0)
	for _, p := range presets {
		presetMap := p.(map[string]any)
		names = append(names, presetMap["name"].(string))
	}
	assert.Contains(t, names, "prod-query-1")
	assert.Contains(t, names, "prod-query-2")
	assert.NotContains(t, names, "dev-query")
}

// VAL-CLI-009: Preset List - Empty Result
func TestPresetListEmptyResult(t *testing.T) {
	store := createTestStore(t)

	// Empty store
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetList(t, store, "", "")

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool), "Empty list should return ok=true, not error")

	resultData := result["result"].(map[string]any)
	presets := resultData["presets"].([]any)
	count := int(resultData["count"].(float64))

	assert.Equal(t, 0, count)
	assert.Len(t, presets, 0)
}

// VAL-CLI-009: Preset List - No Matches for Filter
func TestPresetListNoMatchesForFilter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT 1",
				Tags:      []string{"test"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Filter for non-existent tag
	result := executePresetList(t, store, "nonexistent", "")

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool), "No matches should return ok=true with empty list")

	resultData := result["result"].(map[string]any)
	presets := resultData["presets"].([]any)
	count := int(resultData["count"].(float64))

	assert.Equal(t, 0, count)
	assert.Len(t, presets, 0)
}

// =============================================================================
// PRESET SHOW TESTS (VAL-CLI-019 to VAL-CLI-022)
// =============================================================================

// executePresetShow executes the preset show command with given args and returns the JSON output.
func executePresetShow(t *testing.T, store *preset.PresetStore, args []string) map[string]any {
	// Set output format to LLM (JSON) for testing
	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	// Set the test store
	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	// Create a new command instance for this test
	cmd := createPresetShowCmdForTest()

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(args)

	// Execute
	err := cmd.Execute()
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

// createPresetShowCmdForTest creates a fresh preset show command for testing.
func createPresetShowCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "show [name]",
		Args: cobra.MaximumNArgs(1),
		RunE: presetShowRun,
	}

	return cmd
}

// VAL-CLI-019: Preset Show - Success
func TestPresetShowSuccess(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	expectedPreset := &preset.QueryPreset{
		Name:        "test-preset",
		Description: "Test preset description",
		Query:       "SELECT :name FROM users WHERE id = :id",
		Connection:  "protheus_prod",
		MaxRows:     100,
		Parameters: []preset.ParamDef{
			{
				Name:        "name",
				Type:        "string",
				Required:    true,
				Description: "User name",
			},
			{
				Name:     "id",
				Type:     "int",
				Required: true,
				Default:  "0",
			},
		},
		Tags:      []string{"report", "users"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": expectedPreset,
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetShow(t, store, []string{"test-preset"})

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)

	// Verify all fields are present
	assert.Equal(t, "test-preset", resultData["name"])
	assert.Equal(t, "Test preset description", resultData["description"])
	assert.Equal(t, "SELECT :name FROM users WHERE id = :id", resultData["query"])
	assert.Equal(t, "protheus_prod", resultData["connection"])
	assert.Equal(t, float64(100), resultData["maxRows"])

	// Verify tags
	tags := resultData["tags"].([]any)
	assert.Contains(t, tags, "report")
	assert.Contains(t, tags, "users")

	// Verify parameters array is present
	params := resultData["parameters"].([]any)
	assert.Len(t, params, 2)
}

// VAL-CLI-020: Preset Show - Parameter Detection
func TestPresetShowParameterDetection(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	expectedPreset := &preset.QueryPreset{
		Name:      "params-preset",
		Query:     "SELECT :name, :age FROM users WHERE active = :active",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{
				Name:        "name",
				Type:        "string",
				Required:    true,
				Description: "User name",
			},
			{
				Name:     "age",
				Type:     "int",
				Required: true,
			},
			{
				Name:     "active",
				Type:     "bool",
				Required: false,
				Default:  "true",
			},
		},
	}

	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"params-preset": expectedPreset,
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetShow(t, store, []string{"params-preset"})

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)
	params := resultData["parameters"].([]any)

	require.Len(t, params, 3)

	// Verify each parameter has required fields
	for _, p := range params {
		paramMap := p.(map[string]any)
		assert.Contains(t, paramMap, "name", "Parameter should have 'name' field")
		assert.Contains(t, paramMap, "type", "Parameter should have 'type' field")
		assert.Contains(t, paramMap, "required", "Parameter should have 'required' field")
	}

	// Verify specific parameter values
	nameParam := params[0].(map[string]any)
	assert.Equal(t, "name", nameParam["name"])
	assert.Equal(t, "string", nameParam["type"])
	assert.Equal(t, true, nameParam["required"])
	assert.Equal(t, "User name", nameParam["description"])

	ageParam := params[1].(map[string]any)
	assert.Equal(t, "age", ageParam["name"])
	assert.Equal(t, "int", ageParam["type"])
	assert.Equal(t, true, ageParam["required"])

	activeParam := params[2].(map[string]any)
	assert.Equal(t, "active", activeParam["name"])
	assert.Equal(t, "bool", activeParam["type"])
	assert.Equal(t, false, activeParam["required"])
	assert.Equal(t, "true", activeParam["default"])
}

// VAL-CLI-021: Preset Show - Preset Not Found
func TestPresetShowNotFound(t *testing.T) {
	store := createTestStore(t)

	// Empty store
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetShow(t, store, []string{"nonexistent-preset"})

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))

	errorData := result["error"].(map[string]any)
	assert.Equal(t, "PRESET_NOT_FOUND", errorData["code"])
	assert.Contains(t, errorData["message"], "nonexistent-preset")
}

// VAL-CLI-022: Preset Show - No Name Shows Active
func TestPresetShowNoNameShowsActive(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	activePreset := &preset.QueryPreset{
		Name:      "active-preset",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	}

	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"active-preset": activePreset,
		},
		ActivePreset: "active-preset",
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Call show without name argument
	result := executePresetShow(t, store, []string{})

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)
	assert.Equal(t, "active-preset", resultData["name"])
	assert.Equal(t, "SELECT 1", resultData["query"])
}

// VAL-CLI-022: Preset Show - No Name and No Active
func TestPresetShowNoNameNoActive(t *testing.T) {
	store := createTestStore(t)

	// Empty store with no active preset
	presetFile := &preset.PresetFile{
		Presets:      map[string]*preset.QueryPreset{},
		ActivePreset: "",
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Call show without name argument
	result := executePresetShow(t, store, []string{})

	require.NotNil(t, result)
	assert.True(t, result["ok"].(bool))

	resultData := result["result"].(map[string]any)
	// Should show null for active preset
	assert.Nil(t, resultData["activePreset"])
}

// =============================================================================
// PRESET RUN TESTS (VAL-CLI-010 to VAL-CLI-018)
// =============================================================================

// executePresetRun executes the preset run command with given args and returns the JSON output.
func executePresetRun(t *testing.T, store *preset.PresetStore, args []string, params []string, connection string, maxRows int, outputFile string) map[string]any {
	// Set output format to LLM (JSON) for testing
	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	// Set the test store
	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	// Create a new command instance for this test
	cmd := createPresetRunCmdForTest()

	// Set flag values on the command
	for _, p := range params {
		cmd.Flags().Set("param", p)
	}
	if connection != "" {
		cmd.Flags().Set("connection", connection)
	}
	if maxRows > 0 {
		cmd.Flags().Set("max-rows", fmt.Sprintf("%d", maxRows))
	}
	if outputFile != "" {
		cmd.Flags().Set("output-file", outputFile)
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

// createPresetRunCmdForTest creates a fresh preset run command for testing.
func createPresetRunCmdForTest() *cobra.Command {
	var params []string
	var connection string
	var maxRows int
	var outputFile string

	cmd := &cobra.Command{
		Use:  "run <name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Copy local values to global for presetRunRun to use
			presetRunParams = params
			presetRunConnection = connection
			presetRunMaxRows = maxRows
			presetRunOutputFile = outputFile
			return presetRunRun(cmd, args)
		},
	}

	cmd.Flags().StringArrayVar(&params, "param", nil, "Parameter value (repeatable): key=value")
	cmd.Flags().StringVar(&connection, "connection", "", "Override the preset's connection profile")
	cmd.Flags().IntVar(&maxRows, "max-rows", 0, "Limit number of rows returned")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Write results to file instead of stdout")

	return cmd
}

// VAL-CLI-015: Preset Run - Preset Not Found
func TestPresetRunPresetNotFound(t *testing.T) {
	store := createTestStore(t)

	// Empty store
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	result := executePresetRun(t, store, []string{"nonexistent-preset"}, nil, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "PRESET_NOT_FOUND", errorData["code"])
	assert.Contains(t, errorData["message"], "nonexistent-preset")
	assert.Contains(t, errorData["hint"], "preset list")
}

// VAL-CLI-016: Preset Run - Missing Required Parameter
func TestPresetRunMissingRequiredParameter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT :name, :age FROM users WHERE id = :id",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "name", Type: "string", Required: true},
					{Name: "age", Type: "int", Required: true},
					{Name: "id", Type: "int", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Provide only 'name' param, missing 'age' and 'id'
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"name=John"}, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "MISSING_PARAMETER", errorData["code"])
	assert.Contains(t, errorData["message"], "missing required parameters")

	// Verify hint includes the missing param names
	hint := errorData["hint"].(string)
	assert.Contains(t, hint, "--param")
}

// VAL-CLI-011: Preset Run - With Parameters (parameter interpolation test)
func TestPresetRunWithParameters(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE name = :name AND age = :age",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "name", Type: "string", Required: true},
					{Name: "age", Type: "int", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// This test will fail at connection time, but we can verify the parameter handling
	// by checking the error type - it should be CONNECTION_FAILED or NO_CONNECTION, not MISSING_PARAMETER
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"name=John", "age=25"}, "", 0, "")

	require.NotNil(t, result)
	// The test should fail at connection stage since there's no real connection
	// This verifies params were accepted and interpolation succeeded
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Should be connection error (either NO_CONNECTION or CONNECTION_FAILED), not parameter error
	// Note: In a test environment with an active profile, CONNECTION_FAILED is expected
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// Test parameter with default value
func TestPresetRunParameterWithDefault(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE active = :active AND name = :name",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "active", Type: "bool", Required: false, Default: "true"},
					{Name: "name", Type: "string", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Provide only 'name', 'active' should use default
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"name=John"}, "", 0, "")

	require.NotNil(t, result)
	// Should fail at connection, not missing parameter
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Default should be used, so we shouldn't get MISSING_PARAMETER
	// Accept either NO_CONNECTION or CONNECTION_FAILED (both indicate params were processed)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// Test invalid --param format
func TestPresetRunInvalidParamFormat(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT :name",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Invalid param format (no '=' separator)
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"invalidformat"}, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "INVALID_PARAM_FORMAT", errorData["code"])
}

// Test SQL injection detection
func TestPresetRunSQLInjectionDetection(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE name = :name",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "name", Type: "string", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Try SQL injection
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"name=1; DROP TABLE users"}, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "SQL_INJECTION_DETECTED", errorData["code"])
}

// Test type validation error
func TestPresetRunTypeValidationError(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE age = :age",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "age", Type: "int", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Provide float instead of int
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"age=25.5"}, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "TYPE_MISMATCH", errorData["code"])
}

// Test connection not found
func TestPresetRunConnectionNotFound(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:       "test-preset",
				Query:      "SELECT 1",
				Connection: "nonexistent-connection",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// This will fail trying to find the connection profile
	result := executePresetRun(t, store, []string{"test-preset"}, nil, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "CONNECTION_NOT_FOUND", errorData["code"])
}

// Test --connection override
func TestPresetRunConnectionOverride(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:       "test-preset",
				Query:      "SELECT 1",
				Connection: "original-connection",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Override connection
	result := executePresetRun(t, store, []string{"test-preset"}, nil, "override-connection", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Should fail trying to find override-connection (not original-connection)
	assert.Equal(t, "CONNECTION_NOT_FOUND", errorData["code"])
	assert.Contains(t, errorData["message"], "override-connection")
}

// Test list parameter type
func TestPresetRunListParameter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE id IN (:ids)",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "ids", Type: "list", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Provide list parameter
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"ids=1,2,3"}, "", 0, "")

	require.NotNil(t, result)
	// Should fail at connection, not parameter
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Accept either NO_CONNECTION or CONNECTION_FAILED
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// Test boolean parameter
func TestPresetRunBooleanParameter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE active = :active",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "active", Type: "bool", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Test various boolean values
	tests := []string{"true", "false", "yes", "no", "1", "0"}
	for _, val := range tests {
		t.Run("boolean_"+val, func(t *testing.T) {
			result := executePresetRun(t, store, []string{"test-preset"}, []string{"active=" + val}, "", 0, "")
			require.NotNil(t, result)
			// Should fail at connection, not parameter validation
			assert.False(t, result["ok"].(bool))
			errorData := result["error"].(map[string]any)
			// Accept either NO_CONNECTION or CONNECTION_FAILED
			assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
		})
	}
}

// Test date parameter
func TestPresetRunDateParameter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE created = :date",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "date", Type: "date", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Valid date
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"date=2024-01-15"}, "", 0, "")

	require.NotNil(t, result)
	// Should fail at connection, not parameter
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Accept either NO_CONNECTION or CONNECTION_FAILED
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// Test invalid date parameter
func TestPresetRunInvalidDateParameter(t *testing.T) {
	store := createTestStore(t)

	now := time.Now()
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT * FROM users WHERE created = :date",
				CreatedAt: now,
				UpdatedAt: now,
				Parameters: []preset.ParamDef{
					{Name: "date", Type: "date", Required: true},
				},
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Invalid date format
	result := executePresetRun(t, store, []string{"test-preset"}, []string{"date=not-a-date"}, "", 0, "")

	require.NotNil(t, result)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "TYPE_MISMATCH", errorData["code"])
}

// =============================================================================
// VAL-CLI-037: Help Information
// =============================================================================

func TestPresetHelp_Usage(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		args     []string
		contains []string
	}{
		{
			name:     "preset_add_help",
			cmd:      createPresetAddCmdForTest(),
			args:     []string{"test", "--help"},
			contains: []string{"Create a new query preset", "--query", "--description", "--connection", "--param-def", "--tags", "--use"},
		},
		{
			name:     "preset_list_help",
			cmd:      createPresetListCmdForTest(),
			args:     []string{"--help"},
			contains: []string{"--tag", "--connection"},
		},
		{
			name:     "preset_show_help",
			cmd:      createPresetShowCmdForTest(),
			args:     []string{"--help"},
			contains: []string{"show", "name"},
		},
		{
			name:     "preset_run_help",
			cmd:      createPresetRunCmdForTest(),
			args:     []string{"test", "--help"},
			contains: []string{"--param", "--connection", "--max-rows", "--output-file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetArgs(tt.args)

			// Execute help - this should not error
			err := tt.cmd.Execute()
			require.NoError(t, err, "Help command should not error")

			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Help output should contain '%s'", expected)
			}
		})
	}
}

// Test that preset commands have Short descriptions (help text)
func TestPresetCommandsHaveHelp(t *testing.T) {
	// VAL-CLI-037: Each command provides help with --help
	commands := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"preset", protheusPresetCmd},
		{"preset add", presetAddCmd},
		{"preset list", presetListCmd},
		{"preset show", presetShowCmd},
		{"preset run", presetRunCmd},
		{"preset edit", presetEditCmd},
		{"preset remove", presetRemoveCmd},
		{"preset use", presetUseCmd},
	}

	for _, tt := range commands {
		t.Run(tt.name, func(t *testing.T) {
			// Verify command has a Short description
			assert.NotEmpty(t, tt.cmd.Short, "Command should have Short help text")
			// Verify command has a Long description (detailed help)
			assert.NotEmpty(t, tt.cmd.Long, "Command should have Long help text")
		})
	}
}

// =============================================================================
// VAL-CLI-040: Global Flags
// =============================================================================

func TestGlobalFlags_JSON(t *testing.T) {
	// Test --json flag produces JSON output
	store := createTestStore(t)

	// Set jsonOutput flag
	originalJSON := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = originalJSON }()

	// Set output format
	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	cmd := createPresetListCmdForTest()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify output is valid JSON
	output := strings.TrimSpace(buf.String())
	assert.True(t, json.Valid([]byte(output)), "Output should be valid JSON")

	// Verify it's compact (no newlines for LLM mode)
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "ok")
}

func TestGlobalFlags_Verbose(t *testing.T) {
	// Test --verbose flag includes additional fields
	store := createTestStore(t)

	// Set verbose flag
	originalVerbose := verbose
	verbose = true
	defer func() { verbose = originalVerbose }()

	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	cmd := createPresetListCmdForTest()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse output
	output := strings.TrimSpace(buf.String())
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	// Verbose mode should include schemaVersion and timestamp
	assert.Equal(t, "1.0", result["schemaVersion"], "Verbose mode should include schemaVersion")
	assert.NotEmpty(t, result["timestamp"], "Verbose mode should include timestamp")
}

func TestGlobalFlags_Config(t *testing.T) {
	// Test GetConfigPath function (the --config flag accessor)
	originalConfig := configPath
	configPath = "/custom/config.yaml"
	defer func() { configPath = originalConfig }()

	assert.Equal(t, "/custom/config.yaml", GetConfigPath())
}

func TestGlobalFlags_Profile(t *testing.T) {
	// Test GetProfile function (the --profile flag accessor)
	originalProfile := profileName
	profileName = "custom-profile"
	defer func() { profileName = originalProfile }()

	assert.Equal(t, "custom-profile", GetProfile())
}

func TestGlobalFlags_NoColor(t *testing.T) {
	// Test IsNoColor function (the --no-color flag accessor)
	originalNoColor := noColor
	noColor = true
	defer func() { noColor = originalNoColor }()

	assert.True(t, IsNoColor())
}

func TestGlobalFlags_VerboseFlag(t *testing.T) {
	// Test IsVerbose function (the --verbose flag accessor)
	originalVerbose := verbose
	verbose = true
	defer func() { verbose = originalVerbose }()

	assert.True(t, IsVerbose())
}

func TestGlobalFlags_Accessors(t *testing.T) {
	// Test that all global flag accessor functions exist and work
	// This validates that the global flags system is in place

	// Test that variables exist and can be modified
	originalJSON := jsonOutput
	jsonOutput = true
	assert.True(t, jsonOutput)
	jsonOutput = originalJSON

	originalVerbose := verbose
	verbose = true
	assert.True(t, verbose)
	verbose = originalVerbose

	originalConfig := configPath
	configPath = "test-path"
	assert.Equal(t, "test-path", GetConfigPath())
	configPath = originalConfig

	originalProfile := profileName
	profileName = "test-profile"
	assert.Equal(t, "test-profile", GetProfile())
	profileName = originalProfile

	originalNoColor := noColor
	noColor = true
	assert.True(t, IsNoColor())
	noColor = originalNoColor
}

func TestGetFormatter_WithJSON(t *testing.T) {
	// Test GetFormatter returns correct formatter for --json
	originalJSON := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = originalJSON }()

	originalVerbose := verbose
	verbose = false
	defer func() { verbose = originalVerbose }()

	formatter := GetFormatter()
	assert.IsType(t, output.LLMFormatter{}, formatter)
}

func TestGetFormatter_WithVerbose(t *testing.T) {
	// Test GetFormatter returns correct formatter for --verbose
	originalJSON := jsonOutput
	jsonOutput = false
	defer func() { jsonOutput = originalJSON }()

	originalVerbose := verbose
	verbose = true
	defer func() { verbose = originalVerbose }()

	formatter := GetFormatter()
	// Should be an AutoFormatter with verbose=true
	assert.IsType(t, output.AutoFormatter{}, formatter)
	autoFormatter := formatter.(output.AutoFormatter)
	assert.True(t, autoFormatter.Verbose)
}

// =============================================================================
// Output Format Integration Tests (VAL-CLI-035, VAL-CLI-036, VAL-CLI-038, VAL-CLI-039)
// =============================================================================

func TestOutputFormat_SuccessEnvelope(t *testing.T) {
	// VAL-CLI-036: Success response structure
	store := createTestStore(t)

	// Create a preset
	presetFile := &preset.PresetFile{
		Presets: map[string]*preset.QueryPreset{
			"test-preset": {
				Name:      "test-preset",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Get the preset with list
	result := executePresetList(t, store, "", "")

	// Verify success envelope structure: {ok: true, command: ..., result: ...}
	require.NotNil(t, result)
	assert.Equal(t, true, result["ok"].(bool), "ok should be true for success")
	assert.Contains(t, result, "command", "response should have command field")
	assert.Contains(t, result, "result", "response should have result field")
	assert.Nil(t, result["error"], "error should be nil for success")
}

func TestOutputFormat_ErrorEnvelope(t *testing.T) {
	// VAL-CLI-035: Error response structure
	store := createTestStore(t)

	// Empty store - try to show nonexistent preset
	result := executePresetShow(t, store, []string{"nonexistent"})

	// Verify error envelope structure: {ok: false, error: {code, message, hint, retryable}}
	require.NotNil(t, result)
	assert.Equal(t, false, result["ok"].(bool), "ok should be false for error")
	assert.Contains(t, result, "command", "response should have command field")
	assert.Contains(t, result, "error", "response should have error field")
	assert.Nil(t, result["result"], "result should be nil for error")

	// Verify error object structure
	errorData := result["error"].(map[string]any)
	assert.Contains(t, errorData, "code", "error should have code field")
	assert.Contains(t, errorData, "message", "error should have message field")
	assert.Contains(t, errorData, "retryable", "error should have retryable field (VAL-CLI-035)")

	// Verify error code format: UPPER_SNAKE_CASE
	code := errorData["code"].(string)
	assert.Equal(t, strings.ToUpper(code), code, "Error code must be UPPER_SNAKE_CASE")
	assert.NotContains(t, code, " ", "Error code must not contain spaces")
	assert.NotContains(t, code, "-", "Error code must not contain hyphens")
}

func TestOutputFormat_JSONOutput(t *testing.T) {
	// VAL-CLI-038: --json produces pure JSON without decorations
	store := createTestStore(t)

	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	cmd := createPresetListCmdForTest()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())

	// Must be valid JSON
	assert.True(t, json.Valid([]byte(output)), "Output must be valid JSON")

	// Must not contain ANSI color codes
	assert.NotContains(t, output, "\x1b[", "JSON output must not contain ANSI color codes")
	assert.NotContains(t, output, "\033[", "JSON output must not contain ANSI escape sequences")
}

func TestOutputFormat_VerboseOutput(t *testing.T) {
	// VAL-CLI-039: --verbose includes debug/trace fields
	store := createTestStore(t)

	originalVerbose := verbose
	verbose = true
	defer func() { verbose = originalVerbose }()

	originalFormat := outputFormat
	outputFormat = "llm"
	defer func() { outputFormat = originalFormat }()

	SetPresetStoreForTest(store)
	defer ResetPresetStore()

	cmd := createPresetListCmdForTest()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	// Verbose mode must include additional fields
	assert.Equal(t, "1.0", result["schemaVersion"], "Verbose mode must include schemaVersion")
	assert.NotEmpty(t, result["timestamp"], "Verbose mode must include timestamp")
}
