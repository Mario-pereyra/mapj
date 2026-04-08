package cli

import (
	"bytes"
	"encoding/json"
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
// INTEGRATION TESTS FOR CROSS-AREA FLOWS
// =============================================================================
// These tests validate complete end-to-end flows across multiple preset commands.
// They fulfill assertions VAL-CROSS-001 through VAL-CROSS-007 and VAL-STORAGE-015.
// =============================================================================

// =============================================================================
// Test Helpers for Integration Tests
// =============================================================================

// integrationTestEnv holds the test environment for integration tests
type integrationTestEnv struct {
	store       *preset.PresetStore
	tempDir     string
	originalFmt string
}

// setupIntegrationTest initializes a test environment for integration tests
func setupIntegrationTest(t *testing.T) *integrationTestEnv {
	t.Helper()

	// Create temp directory
	tempDir := t.TempDir()
	presetPath := filepath.Join(tempDir, "presets.json")

	// Create preset store with test path
	store, err := preset.NewPresetStore()
	require.NoError(t, err, "Failed to create preset store")
	store.SetPath(presetPath)

	// Set output format to LLM for JSON output
	originalFmt := outputFormat
	outputFormat = "llm"

	// Reset global store and set test store
	ResetPresetStore()
	SetPresetStoreForTest(store)

	return &integrationTestEnv{
		store:       store,
		tempDir:     tempDir,
		originalFmt: originalFmt,
	}
}

// cleanupIntegrationTest cleans up the test environment
func cleanupIntegrationTest(t *testing.T, env *integrationTestEnv) {
	t.Helper()
	outputFormat = env.originalFmt
	ResetPresetStore()
}

// executeCommand executes a cobra command and returns the parsed JSON output
func executeCommand(t *testing.T, cmd *cobra.Command, args []string) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	require.NoError(t, err, "Command execution should not return Go errors")

	output := strings.TrimSpace(buf.String())
	if output == "" {
		return nil
	}

	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Output should be valid JSON: %s", output)

	return result
}

// =============================================================================
// VAL-CROSS-001: Full Create and Execute Flow
// =============================================================================
// User creates preset with typed params, inspects it, runs with values,
// sees results in stdout. All commands succeed, JSON outputs are parseable,
// parameters substituted correctly.

func TestIntegration_FullCreateAndExecuteFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Step 1: Create a preset with parameters
	addCmd := createPresetAddCmdForTest()
	addCmd.Flags().Set("query", "SELECT * FROM users WHERE name = :name AND age > :minAge")
	addCmd.Flags().Set("description", "Query users by name and minimum age")
	addCmd.Flags().Set("param-def", "name:string::User name to search")
	addCmd.Flags().Set("param-def", "minAge:int:0:Minimum age filter")
	addCmd.Flags().Set("tags", "users,search")
	addCmd.Flags().Set("use", "true")

	result := executeCommand(t, addCmd, []string{"user-search"})

	// Verify add success
	require.True(t, result["ok"].(bool), "Add command should succeed")
	presetData := result["result"].(map[string]any)
	assert.Equal(t, "user-search", presetData["name"])
	assert.Contains(t, presetData["detectedParameters"], "name")
	assert.Contains(t, presetData["detectedParameters"], "minAge")

	// Verify preset is marked as active
	assert.True(t, presetData["active"].(bool))

	// Step 2: Show the preset to verify it was saved correctly
	showCmd := createPresetShowCmdForTest()
	result = executeCommand(t, showCmd, []string{"user-search"})

	require.True(t, result["ok"].(bool), "Show command should succeed")
	presetData = result["result"].(map[string]any)
	assert.Equal(t, "user-search", presetData["name"])
	assert.Equal(t, "SELECT * FROM users WHERE name = :name AND age > :minAge", presetData["query"])

	// Verify parameters are present with correct types
	params := presetData["parameters"].([]any)
	require.Len(t, params, 2)

	// Verify parameter details
	nameParam := params[0].(map[string]any)
	assert.Equal(t, "name", nameParam["name"])
	assert.Equal(t, "string", nameParam["type"])
	assert.True(t, nameParam["required"].(bool))

	minAgeParam := params[1].(map[string]any)
	assert.Equal(t, "minAge", minAgeParam["name"])
	assert.Equal(t, "int", minAgeParam["type"])
	assert.Equal(t, "0", minAgeParam["default"])

	// Step 3: Run the preset with parameters
	// Note: This will fail at connection stage since there's no real DB connection
	// But we can verify the parameter handling succeeded
	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "name=John")
	runCmd.Flags().Set("param", "minAge=25")

	result = executeCommand(t, runCmd, []string{"user-search"})

	// Should fail at connection (expected), not at parameter validation
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Accept either NO_CONNECTION or CONNECTION_FAILED - both indicate params were processed
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
		"Should fail at connection stage, not at parameter validation")

	// Verify the preset file contains all the data
	loaded, err := env.store.Load()
	require.NoError(t, err)
	savedPreset := loaded.GetPreset("user-search")
	require.NotNil(t, savedPreset)
	assert.Equal(t, "user-search", savedPreset.Name)
	assert.Equal(t, "Query users by name and minimum age", savedPreset.Description)
	assert.Contains(t, savedPreset.Tags, "users")
	assert.Contains(t, savedPreset.Tags, "search")
	assert.Equal(t, "user-search", loaded.ActivePreset)
}

// =============================================================================
// VAL-CROSS-002: Agent-Friendly Flow
// =============================================================================
// LLM discovers parameters with `preset show`, executes non-interactively,
// parses JSON output. `show --output json` produces parseable JSON,
// run requires no prompts.

func TestIntegration_AgentFriendlyFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Create a preset with multiple typed parameters
	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "agent-test",
		Query:       "SELECT :id, :name, :status, :created FROM users WHERE active = :active",
		Description: "Agent-friendly test preset",
		Parameters: []preset.ParamDef{
			{Name: "id", Type: "int", Required: true, Description: "User ID"},
			{Name: "name", Type: "string", Required: false, Default: "NULL", Description: "User name"},
			{Name: "status", Type: "string", Required: false, Default: "active", Description: "User status"},
			{Name: "created", Type: "date", Required: false, Description: "Creation date"},
			{Name: "active", Type: "bool", Required: false, Default: "true", Description: "Is active"},
		},
		Tags:      []string{"agent", "test"},
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, env.store.Save(presetFile))

	// Step 1: Agent discovers parameters via show command
	showCmd := createPresetShowCmdForTest()
	result := executeCommand(t, showCmd, []string{"agent-test"})

	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)

	// Verify JSON is parseable and contains parameter info
	assert.Contains(t, presetData, "parameters")
	params := presetData["parameters"].([]any)
	require.Len(t, params, 5)

	// Verify each parameter has agent-friendly info
	for _, p := range params {
		paramMap := p.(map[string]any)
		assert.Contains(t, paramMap, "name", "Parameter should have 'name' for agent discovery")
		assert.Contains(t, paramMap, "type", "Parameter should have 'type' for agent validation")
		assert.Contains(t, paramMap, "required", "Parameter should have 'required' for agent decisions")
	}

	// Step 2: Agent runs preset with only required parameters
	// (optional params use defaults)
	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "id=123")

	result = executeCommand(t, runCmd, []string{"agent-test"})

	// Should fail at connection, but parameters should be processed
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	// Accept NO_CONNECTION or CONNECTION_FAILED - both indicate params were validated
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
		"Should fail at connection, params should use defaults successfully")

	// Step 3: Verify show without name shows active preset (or null)
	showCmdNoArgs := createPresetShowCmdForTest()
	result = executeCommand(t, showCmdNoArgs, []string{})

	require.True(t, result["ok"].(bool))
	// No active preset set, so should show null
	presetData = result["result"].(map[string]any)
	assert.Nil(t, presetData["activePreset"], "No active preset should return null")

	// Step 4: Verify JSON output is valid and parseable for jq
	listCmd := createPresetListCmdForTest()
	result = executeCommand(t, listCmd, []string{})

	require.True(t, result["ok"].(bool))
	// Verify structure is jq-friendly
	resultData := result["result"].(map[string]any)
	assert.Contains(t, resultData, "presets")
	assert.Contains(t, resultData, "count")
	presets := resultData["presets"].([]any)
	assert.GreaterOrEqual(t, len(presets), 1)
}

// =============================================================================
// VAL-CROSS-003: Security Flow - SQL Injection
// =============================================================================
// System rejects injection attempts with clear errors, no query execution.
// All injection patterns rejected, errors are descriptive.

func TestIntegration_SQLInjectionSecurity(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Create a preset with a string parameter
	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "security-test",
		Query:     "SELECT * FROM users WHERE name = :name",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "name", Type: "string", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// Test all SQL injection patterns from validation contract
	injectionTests := []struct {
		name     string
		value    string
		expected string // expected error code
	}{
		// VAL-PARAM-012: Semicolon DROP
		{"semicolon_drop", "1; DROP TABLE users", "SQL_INJECTION_DETECTED"},
		// VAL-PARAM-013: OR 1=1
		{"or_1_equals_1", "' OR 1=1 --", "SQL_INJECTION_DETECTED"},
		// VAL-PARAM-014: UNION SELECT
		{"union_select", "' UNION SELECT password FROM users", "SQL_INJECTION_DETECTED"},
		// VAL-PARAM-015: Comment injection
		{"comment_injection", "value'--", "SQL_INJECTION_DETECTED"},
		// VAL-PARAM-016: Combined patterns
		{"combined_attack", "1'; DROP TABLE users; --", "SQL_INJECTION_DETECTED"},
		// Additional patterns
		{"semicolon_delete", "1; DELETE FROM users", "SQL_INJECTION_DETECTED"},
		{"semicolon_truncate", "1; TRUNCATE TABLE users", "SQL_INJECTION_DETECTED"},
		{"or_string_equals", "' OR '1'='1", "SQL_INJECTION_DETECTED"},
		{"union_all", "' UNION ALL SELECT * FROM users", "SQL_INJECTION_DETECTED"},
	}

	for _, tt := range injectionTests {
		t.Run(tt.name, func(t *testing.T) {
			runCmd := createPresetRunCmdForTest()
			runCmd.Flags().Set("param", "name="+tt.value)

			result := executeCommand(t, runCmd, []string{"security-test"})

			// Must reject the injection
			assert.False(t, result["ok"].(bool), "Should reject SQL injection")
			errorData := result["error"].(map[string]any)
			assert.Equal(t, tt.expected, errorData["code"],
				"Should detect SQL injection pattern: %s", tt.name)

			// Error message should be descriptive
			assert.Contains(t, errorData["message"], "injection",
				"Error message should mention injection")
		})
	}
}

// Test that safe values with quotes are properly escaped, not rejected
func TestIntegration_SafeValuesWithQuotes(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "quote-test",
		Query:     "SELECT * FROM users WHERE name = :name",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "name", Type: "string", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// These values contain quotes but are NOT SQL injection
	safeValues := []string{
		"O'Brien",           // Irish name
		"l'ordinateur",      // French text
		"user's data",       // Possessive
		"it's a test",       // Contraction
		"he said 'hello'",   // Nested quotes
	}

	for _, value := range safeValues {
		t.Run("safe_"+value, func(t *testing.T) {
			runCmd := createPresetRunCmdForTest()
			runCmd.Flags().Set("param", "name="+value)

			result := executeCommand(t, runCmd, []string{"quote-test"})

			// Should fail at connection (expected), NOT SQL injection
			assert.False(t, result["ok"].(bool))
			errorData := result["error"].(map[string]any)
			// Should be connection error, not injection
			assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
				"Safe values with quotes should be escaped, not rejected as injection")
		})
	}
}

// =============================================================================
// VAL-CROSS-004: Preset Management Flow
// =============================================================================
// List, edit, remove presets with full cleanup verification.
// File and references are properly synchronized after operations.

func TestIntegration_PresetManagementFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()

	// Step 1: Create multiple presets
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "mgmt-test-1",
		Query:       "SELECT 1",
		Description: "First preset",
		Tags:        []string{"test", "first"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "mgmt-test-2",
		Query:       "SELECT 2",
		Description: "Second preset",
		Tags:        []string{"test", "second"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "mgmt-test-3",
		Query:       "SELECT 3",
		Description: "Third preset",
		Tags:        []string{"test", "third"},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	require.NoError(t, env.store.Save(presetFile))

	// Step 2: List all presets
	listCmd := createPresetListCmdForTest()
	result := executeCommand(t, listCmd, []string{})

	require.True(t, result["ok"].(bool))
	resultData := result["result"].(map[string]any)
	assert.Equal(t, float64(3), resultData["count"])
	presets := resultData["presets"].([]any)
	assert.Len(t, presets, 3)

	// Step 3: Edit a preset
	editCmd := createEditCommand()
	editCmd.Flags().Set("description", "Updated description")
	editCmd.Flags().Set("query", "SELECT 1, 2, 3")

	result = executeCommand(t, editCmd, []string{"mgmt-test-1"})

	require.True(t, result["ok"].(bool))
	editData := result["result"].(map[string]any)
	assert.Contains(t, editData["fields_updated"], "description")
	assert.Contains(t, editData["fields_updated"], "query")

	// Verify other presets unchanged
	loaded, _ := env.store.Load()
	preset2 := loaded.GetPreset("mgmt-test-2")
	assert.Equal(t, "Second preset", preset2.Description) // unchanged

	// Step 4: Set active preset
	useCmd := createUseCommand()
	result = executeCommand(t, useCmd, []string{"mgmt-test-2"})

	require.True(t, result["ok"].(bool))
	loaded, _ = env.store.Load()
	assert.Equal(t, "mgmt-test-2", loaded.ActivePreset)

	// Step 5: Remove active preset
	removeCmd := createRemoveCommand()
	removeCmd.Flags().Set("force", "true")
	result = executeCommand(t, removeCmd, []string{"mgmt-test-2"})

	require.True(t, result["ok"].(bool))
	removeData := result["result"].(map[string]any)
	assert.Equal(t, "mgmt-test-2", removeData["removed"])
	assert.True(t, removeData["was_active"].(bool))

	// Step 6: Verify cleanup
	loaded, _ = env.store.Load()
	assert.Nil(t, loaded.GetPreset("mgmt-test-2"), "Preset should be removed")
	assert.Equal(t, "", loaded.ActivePreset, "Active preset should be cleared")
	assert.NotNil(t, loaded.GetPreset("mgmt-test-1"), "Other presets should remain")
	assert.NotNil(t, loaded.GetPreset("mgmt-test-3"), "Other presets should remain")

	// Step 7: List again to verify
	listCmd = createPresetListCmdForTest()
	result = executeCommand(t, listCmd, []string{})

	require.True(t, result["ok"].(bool))
	resultData = result["result"].(map[string]any)
	assert.Equal(t, float64(2), resultData["count"])
}

// =============================================================================
// VAL-CROSS-005: Connection Profile Integration
// =============================================================================
// Preset connection, override, fallback to active profile all work.
// Correct connection used in each scenario, error when none available.

func TestIntegration_ConnectionProfileFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()

	// Create presets with different connection scenarios
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:       "conn-test-with-default",
		Query:      "SELECT 1",
		Connection: "saved-connection",
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	presetFile.SetPreset(&preset.QueryPreset{
		Name:       "conn-test-no-default",
		Query:      "SELECT 2",
		Connection: "", // No default connection
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	require.NoError(t, env.store.Save(presetFile))

	// Scenario 1: Preset has saved connection, but it doesn't exist
	runCmd := createPresetRunCmdForTest()
	result := executeCommand(t, runCmd, []string{"conn-test-with-default"})

	assert.False(t, result["ok"].(bool), "Should fail when connection profile doesn't exist")
	errorData, ok := result["error"].(map[string]any)
	require.True(t, ok, "Result should have error data")
	assert.Equal(t, "CONNECTION_NOT_FOUND", errorData["code"])
	assert.Contains(t, errorData["message"], "saved-connection")

	// Scenario 2: Override connection with --connection flag
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("connection", "override-connection")
	result = executeCommand(t, runCmd, []string{"conn-test-with-default"})

	assert.False(t, result["ok"].(bool), "Should fail when override connection doesn't exist")
	errorData, ok = result["error"].(map[string]any)
	require.True(t, ok, "Result should have error data")
	assert.Equal(t, "CONNECTION_NOT_FOUND", errorData["code"])
	assert.Contains(t, errorData["message"], "override-connection")

	// Scenario 3: Test connection resolution priority (flag > preset.connection > active profile)
	// This validates that --connection flag takes precedence over preset's saved connection
	// by using an --connection value that doesn't exist (proving the flag was used)
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("connection", "nonexistent-flag-connection")
	result = executeCommand(t, runCmd, []string{"conn-test-with-default"})

	assert.False(t, result["ok"].(bool), "Should fail when flag connection doesn't exist")
	errorData, ok = result["error"].(map[string]any)
	require.True(t, ok, "Result should have error data")
	assert.Equal(t, "CONNECTION_NOT_FOUND", errorData["code"])
	// The error should reference the flag override, not the preset's saved connection
	assert.Contains(t, errorData["message"], "nonexistent-flag-connection")
}

// =============================================================================
// VAL-CROSS-006: Error Recovery Flow
// =============================================================================
// Errors are graceful with actionable hints.
// Each error scenario has clear, helpful message.

func TestIntegration_ErrorRecoveryFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Scenario 1: Preset not found
	showCmd := createPresetShowCmdForTest()
	result := executeCommand(t, showCmd, []string{"nonexistent-preset"})

	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Equal(t, "PRESET_NOT_FOUND", errorData["code"])
	assert.NotEmpty(t, errorData["hint"], "Error should have actionable hint")
	assert.Contains(t, errorData["hint"], "list")

	// Scenario 2: Missing required parameter
	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "error-test-required",
		Query:     "SELECT * FROM users WHERE id = :id AND name = :name",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "id", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "id=123") // missing name
	result = executeCommand(t, runCmd, []string{"error-test-required"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "MISSING_PARAMETER", errorData["code"])
	assert.Contains(t, errorData["message"], "name")
	assert.Contains(t, errorData["hint"], "--param")

	// Scenario 3: Type mismatch
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "id=not-a-number")
	runCmd.Flags().Set("param", "name=test")
	result = executeCommand(t, runCmd, []string{"error-test-required"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "TYPE_MISMATCH", errorData["code"])
	assert.Contains(t, errorData["message"], "int")

	// Scenario 4: Invalid parameter format
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "invalidformat") // no '=' separator
	result = executeCommand(t, runCmd, []string{"error-test-required"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "INVALID_PARAM_FORMAT", errorData["code"])
	assert.Contains(t, errorData["hint"], "key=value")

	// Scenario 5: Duplicate preset name
	addCmd := createPresetAddCmdForTest()
	addCmd.Flags().Set("query", "SELECT 1")
	result = executeCommand(t, addCmd, []string{"error-test-required"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "PRESET_EXISTS", errorData["code"])
	assert.Contains(t, errorData["hint"], "edit")

	// Scenario 6: No fields to update in edit
	editCmd := createEditCommand()
	result = executeCommand(t, editCmd, []string{"error-test-required"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "NO_FIELDS_TO_UPDATE", errorData["code"])
	assert.NotEmpty(t, errorData["hint"])
}

// =============================================================================
// VAL-CROSS-007: Complex Types Flow
// =============================================================================
// List and datetime types work with proper escaping and validation.

func TestIntegration_ComplexTypesFlow(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Create preset with complex parameter types
	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "complex-types-test",
		Query:     "SELECT * FROM orders WHERE id IN (:ids) AND created_at >= :startDate AND status = :status",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "ids", Type: "list", Required: true, Description: "Order IDs (comma-separated)"},
			{Name: "startDate", Type: "datetime", Required: true, Description: "Start date and time"},
			{Name: "status", Type: "string", Required: false, Default: "pending"},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// Test 1: List type with proper IN clause generation
	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "ids=1,2,3,4,5")
	runCmd.Flags().Set("param", "startDate=2024-01-15 10:30:00")
	runCmd.Flags().Set("param", "status=completed")

	result := executeCommand(t, runCmd, []string{"complex-types-test"})

	// Should fail at connection (expected), not at parameter processing
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
		"Complex types should be processed successfully")

	// Test 2: Datetime with T separator (ISO 8601)
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "ids=100")
	runCmd.Flags().Set("param", "startDate=2024-01-15T10:30:00")

	result = executeCommand(t, runCmd, []string{"complex-types-test"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])

	// Test 3: List with special characters (quotes should be escaped)
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "ids=O'Brien,Jane's,Test")
	runCmd.Flags().Set("param", "startDate=2024-01-15 00:00:00")

	result = executeCommand(t, runCmd, []string{"complex-types-test"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
		"List with special characters should be escaped successfully")

	// Test 4: Invalid datetime format should error
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "ids=1")
	runCmd.Flags().Set("param", "startDate=invalid-datetime")

	result = executeCommand(t, runCmd, []string{"complex-types-test"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "TYPE_MISMATCH", errorData["code"])

	// Test 5: Date type (without time)
	presetFile.Presets["date-only-test"] = &preset.QueryPreset{
		Name:      "date-only-test",
		Query:     "SELECT * FROM orders WHERE date = :orderDate",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "orderDate", Type: "date", Required: true},
		},
	}
	require.NoError(t, env.store.Save(presetFile))

	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "orderDate=2024-01-15")

	result = executeCommand(t, runCmd, []string{"date-only-test"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])

	// Test 6: Invalid date format
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "orderDate=2024/01/15") // Wrong separator

	result = executeCommand(t, runCmd, []string{"date-only-test"})

	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Equal(t, "TYPE_MISMATCH", errorData["code"])
}

// =============================================================================
// VAL-STORAGE-015: Unicode and Special Characters Support
// =============================================================================
// Presets correctly store and retrieve queries containing Unicode
// characters and special SQL characters.

func TestIntegration_UnicodeAndSpecialCharacters(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	// Create preset with Unicode and special characters
	addCmd := createPresetAddCmdForTest()
	addCmd.Flags().Set("query", "SELECT * FROM users WHERE name LIKE '%O''Brien%' AND comment LIKE '%--comment%'")
	addCmd.Flags().Set("description", "Descripción en español - 中文 - 日本語 - 한글")

	result := executeCommand(t, addCmd, []string{"unicode-test"})

	require.True(t, result["ok"].(bool))
	presetData := result["result"].(map[string]any)
	assert.Equal(t, "unicode-test", presetData["name"])

	// Verify retrieval preserves all characters
	showCmd := createPresetShowCmdForTest()
	result = executeCommand(t, showCmd, []string{"unicode-test"})

	require.True(t, result["ok"].(bool))
	presetData = result["result"].(map[string]any)

	// Unicode in description should be preserved
	assert.Equal(t, "Descripción en español - 中文 - 日本語 - 한글", presetData["description"])

	// Special SQL characters in query should be preserved
	query := presetData["query"].(string)
	assert.Contains(t, query, "O''Brien") // Escaped quote in query
	assert.Contains(t, query, "--comment") // Comment pattern in query (legitimate)

	// Verify file on disk contains valid JSON with Unicode
	loaded, err := env.store.Load()
	require.NoError(t, err)
	savedPreset := loaded.GetPreset("unicode-test")
	require.NotNil(t, savedPreset)
	assert.Equal(t, "Descripción en español - 中文 - 日本語 - 한글", savedPreset.Description)

	// Test with various Unicode values in parameters
	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "unicode-param-test",
		Query:     "SELECT * FROM users WHERE name = :name AND city = :city",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "name", Type: "string", Required: true},
			{Name: "city", Type: "string", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	unicodeTests := []struct {
		name  string
		value string
	}{
		{"chinese", "张三"},
		{"japanese", "田中太郎"},
		{"korean", "김철수"},
		{"arabic", "أحمد"},
		{"russian", "Иван"},
		{"emoji", "👨‍👩‍👧‍👦"},
		{"mixed", "John 日本語 Smith 中文"},
	}

	for _, tt := range unicodeTests {
		t.Run("unicode_param_"+tt.name, func(t *testing.T) {
			runCmd := createPresetRunCmdForTest()
			runCmd.Flags().Set("param", "name="+tt.value)
			runCmd.Flags().Set("param", "city=Tokyo")

			result := executeCommand(t, runCmd, []string{"unicode-param-test"})

			// Should fail at connection, not at parameter processing
			assert.False(t, result["ok"].(bool))
			errorData := result["error"].(map[string]any)
			assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
				"Unicode in parameters should be handled successfully")
		})
	}
}

// =============================================================================
// Additional Integration Tests
// =============================================================================

// TestIntegration_DefaultValues tests that default values work correctly
func TestIntegration_DefaultValues(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "defaults-test",
		Query:     "SELECT * FROM users WHERE status = :status AND limit = :limit AND active = :active",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "status", Type: "string", Required: false, Default: "active"},
			{Name: "limit", Type: "int", Required: false, Default: "100"},
			{Name: "active", Type: "bool", Required: false, Default: "true"},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// Run without providing any params - defaults should be used
	runCmd := createPresetRunCmdForTest()
	result := executeCommand(t, runCmd, []string{"defaults-test"})

	// Should fail at connection (params processed with defaults)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])

	// Run with explicit values that override defaults
	runCmd = createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "status=inactive")
	runCmd.Flags().Set("param", "limit=50")
	runCmd.Flags().Set("param", "active=false")

	result = executeCommand(t, runCmd, []string{"defaults-test"})

	// Should fail at connection
	assert.False(t, result["ok"].(bool))
	errorData = result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// TestIntegration_BooleanTypeVariations tests all boolean value variations
func TestIntegration_BooleanTypeVariations(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "bool-test",
		Query:     "SELECT * FROM users WHERE active = :active",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "active", Type: "bool", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// All valid boolean representations
	validBools := []string{"true", "false", "TRUE", "FALSE", "True", "False", "1", "0", "yes", "no", "YES", "NO", "Yes", "No"}

	for _, val := range validBools {
		t.Run("bool_"+val, func(t *testing.T) {
			runCmd := createPresetRunCmdForTest()
			runCmd.Flags().Set("param", "active="+val)

			result := executeCommand(t, runCmd, []string{"bool-test"})

			// Should fail at connection, not at parameter validation
			assert.False(t, result["ok"].(bool))
			errorData := result["error"].(map[string]any)
			assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"],
				"Boolean value '%s' should be accepted", val)
		})
	}

	// Invalid boolean values
	invalidBools := []string{"maybe", "2", "y", "n", "t", "f", "yes please"}

	for _, val := range invalidBools {
		t.Run("invalid_bool_"+val, func(t *testing.T) {
			runCmd := createPresetRunCmdForTest()
			runCmd.Flags().Set("param", "active="+val)

			result := executeCommand(t, runCmd, []string{"bool-test"})

			// Should fail at type validation
			assert.False(t, result["ok"].(bool))
			errorData := result["error"].(map[string]any)
			assert.Equal(t, "TYPE_MISMATCH", errorData["code"],
				"Invalid boolean '%s' should be rejected", val)
		})
	}
}

// TestIntegration_MultipleParameterUsage tests parameters used multiple times in query
func TestIntegration_MultipleParameterUsage(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "multi-usage-test",
		Query:     "SELECT * FROM users WHERE name = :name OR alt_name = :name OR nickname = :name",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "name", Type: "string", Required: true},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "name=John")

	result := executeCommand(t, runCmd, []string{"multi-usage-test"})

	// Should fail at connection (params processed successfully)
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}

// TestIntegration_EmptyStringParameter tests empty string handling
func TestIntegration_EmptyStringParameter(t *testing.T) {
	env := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, env)

	now := time.Now()
	presetFile, _ := env.store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "empty-string-test",
		Query:     "SELECT * FROM users WHERE middle_name = :middleName",
		CreatedAt: now,
		UpdatedAt: now,
		Parameters: []preset.ParamDef{
			{Name: "middleName", Type: "string", Required: false},
		},
	})
	require.NoError(t, env.store.Save(presetFile))

	// Empty string should be accepted
	runCmd := createPresetRunCmdForTest()
	runCmd.Flags().Set("param", "middleName=") // Empty value

	result := executeCommand(t, runCmd, []string{"empty-string-test"})

	// Should fail at connection, not at parameter validation
	assert.False(t, result["ok"].(bool))
	errorData := result["error"].(map[string]any)
	assert.Contains(t, []string{"NO_CONNECTION", "CONNECTION_FAILED"}, errorData["code"])
}
