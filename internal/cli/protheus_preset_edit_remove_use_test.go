package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mario-pereyra/mapj/internal/output"
	"github.com/Mario-pereyra/mapj/internal/preset"
	"github.com/spf13/cobra"
)

// =============================================================================
// Test Helpers for Edit/Remove/Use Commands
// =============================================================================

func setupEditRemoveUseTest(t *testing.T) (string, *preset.PresetStore) {
	t.Helper()

	// Set output format to LLM for JSON output in tests
	originalFormat := outputFormat
	outputFormat = "llm"
	t.Cleanup(func() { outputFormat = originalFormat })

	// Create temp directory for test
	tempDir := t.TempDir()
	presetPath := filepath.Join(tempDir, "presets.json")

	// Create preset store with test path
	store, err := preset.NewPresetStore()
	if err != nil {
		t.Fatalf("failed to create preset store: %v", err)
	}
	store.SetPath(presetPath)

	// Reset global store and set test store
	ResetPresetStore()
	SetPresetStoreForTest(store)

	return presetPath, store
}

func cleanupEditRemoveUseTest(t *testing.T) {
	t.Helper()
	ResetPresetStore()
}

// createEditCommand creates a fresh edit command for testing
func createEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "edit <name>",
		Args: cobra.ExactArgs(1),
		RunE: presetEditRun,
	}

	// Reset flag variables before each test
	presetEditDescription = ""
	presetEditQuery = ""
	presetEditConnection = ""
	presetEditMaxRows = 0
	presetEditParamDefs = nil
	presetEditTags = ""

	cmd.Flags().StringVar(&presetEditDescription, "description", "", "Update preset description")
	cmd.Flags().StringVar(&presetEditQuery, "query", "", "Update the SQL query")
	cmd.Flags().StringVar(&presetEditConnection, "connection", "", "Update the default connection profile")
	cmd.Flags().IntVar(&presetEditMaxRows, "max-rows", 0, "Update the default max rows limit")
	cmd.Flags().StringArrayVar(&presetEditParamDefs, "param-def", nil, "Update parameter definitions")
	cmd.Flags().StringVar(&presetEditTags, "tags", "", "Update tags (comma-separated)")

	return cmd
}

// createRemoveCommand creates a fresh remove command for testing
func createRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "remove <name>",
		Args: cobra.ExactArgs(1),
		RunE: presetRemoveRun,
	}

	presetRemoveForce = false
	cmd.Flags().BoolVar(&presetRemoveForce, "force", false, "Skip confirmation prompt")

	return cmd
}

// createUseCommand creates a fresh use command for testing
func createUseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "use [name]",
		Args: cobra.MaximumNArgs(1),
		RunE: presetUseRun,
	}

	return cmd
}

// =============================================================================
// PRESET EDIT Tests
// =============================================================================

// TestPresetEditSuccess tests VAL-CLI-023: preset edit --description updates field
func TestPresetEditSuccess(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset to edit
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "test-preset",
		Query:       "SELECT 1",
		Description: "original description",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	store.Save(presetFile)

	// Execute edit command
	cmd := createEditCommand()
	cmd.SetArgs([]string{"test-preset", "--description", "updated description"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify result contains updated preset
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	// Verify fields_updated contains description
	fieldsUpdated, _ := result["fields_updated"].([]interface{})
	found := false
	for _, f := range fieldsUpdated {
		if f == "description" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'description' in fields_updated, got %v", fieldsUpdated)
	}

	// Verify updatedAt was updated
	if result["updatedAt"] == now.Format(time.RFC3339) {
		t.Error("expected updatedAt to be updated, but it wasn't")
	}
}

// TestPresetEditMultipleFields tests VAL-CLI-024: preset edit with multiple flags updates all
func TestPresetEditMultipleFields(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset to edit
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "multi-preset",
		Query:       "SELECT 1",
		Description: "original",
		Connection:  "old-conn",
		MaxRows:     10,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	store.Save(presetFile)

	// Execute edit command with multiple flags
	cmd := createEditCommand()
	cmd.SetArgs([]string{
		"multi-preset",
		"--description", "new desc",
		"--query", "SELECT 2",
		"--connection", "new-conn",
		"--max-rows", "100",
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify fields_updated contains all updated fields
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	fieldsUpdated, _ := result["fields_updated"].([]interface{})
	expectedFields := map[string]bool{"description": false, "query": false, "connection": false, "maxRows": false}
	for _, f := range fieldsUpdated {
		if s, ok := f.(string); ok {
			expectedFields[s] = true
		}
	}
	for field, found := range expectedFields {
		if !found {
			t.Errorf("expected '%s' in fields_updated", field)
		}
	}

	// Verify store was updated
	presetFile2, _ := store.Load()
	updatedPreset := presetFile2.GetPreset("multi-preset")
	if updatedPreset.Description != "new desc" {
		t.Errorf("expected description='new desc', got '%s'", updatedPreset.Description)
	}
	if updatedPreset.Query != "SELECT 2" {
		t.Errorf("expected query='SELECT 2', got '%s'", updatedPreset.Query)
	}
	if updatedPreset.Connection != "new-conn" {
		t.Errorf("expected connection='new-conn', got '%s'", updatedPreset.Connection)
	}
	if updatedPreset.MaxRows != 100 {
		t.Errorf("expected maxRows=100, got %d", updatedPreset.MaxRows)
	}
}

// TestPresetEditNoFields tests VAL-CLI-025: preset edit without flags returns error NO_FIELDS_TO_UPDATE
func TestPresetEditNoFields(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "no-fields-preset",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute edit command without any flags
	cmd := createEditCommand()
	cmd.SetArgs([]string{"no-fields-preset"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify error
	if env.OK {
		t.Error("expected OK=false for no fields to update")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "NO_FIELDS_TO_UPDATE" {
		t.Errorf("expected error code 'NO_FIELDS_TO_UPDATE', got '%s'", env.Error.Code)
	}
}

// TestPresetEditNotFound tests VAL-CLI-026: preset edit nonexistent returns PRESET_NOT_FOUND
func TestPresetEditNotFound(t *testing.T) {
	setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Execute edit command on non-existent preset
	cmd := createEditCommand()
	cmd.SetArgs([]string{"nonexistent", "--description", "test"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify error
	if env.OK {
		t.Error("expected OK=false for preset not found")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "PRESET_NOT_FOUND" {
		t.Errorf("expected error code 'PRESET_NOT_FOUND', got '%s'", env.Error.Code)
	}
	// Verify hint points to list command
	if env.Error.Hint == "" || !strings.Contains(env.Error.Hint, "list") {
		t.Errorf("expected hint to suggest using 'list' command, got '%s'", env.Error.Hint)
	}
}

// TestPresetEditInvalidParamDef tests VAL-CLI-027: preset edit invalid param-def returns error
func TestPresetEditInvalidParamDef(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "param-preset",
		Query:     "SELECT :id",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute edit command with invalid param-def
	cmd := createEditCommand()
	cmd.SetArgs([]string{"param-preset", "--param-def", "invalid-format"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify error
	if env.OK {
		t.Error("expected OK=false for invalid param-def")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "INVALID_PARAM_DEF" {
		t.Errorf("expected error code 'INVALID_PARAM_DEF', got '%s'", env.Error.Code)
	}
}

// TestPresetEditPreservesOtherPresets tests VAL-STORAGE-009: Preserve other presets on update
func TestPresetEditPreservesOtherPresets(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create two presets
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "preset-a",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "preset-b",
		Query:       "SELECT 2",
		Description: "original b",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	store.Save(presetFile)

	// Edit preset-a
	cmd := createEditCommand()
	cmd.SetArgs([]string{"preset-a", "--description", "updated a"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reload and verify preset-b unchanged
	presetFile2, _ := store.Load()
	presetB := presetFile2.GetPreset("preset-b")
	if presetB == nil {
		t.Fatal("preset-b should still exist")
	}
	if presetB.Description != "original b" {
		t.Errorf("preset-b description should be unchanged, got '%s'", presetB.Description)
	}
}

// =============================================================================
// PRESET REMOVE Tests
// =============================================================================

// TestPresetRemoveSuccess tests VAL-CLI-028: preset remove deletes preset
func TestPresetRemoveSuccess(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "remove-test",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute remove command with --force to skip confirmation
	cmd := createRemoveCommand()
	cmd.SetArgs([]string{"remove-test", "--force"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify result contains removed preset name
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	if result["removed"] != "remove-test" {
		t.Errorf("expected removed='remove-test', got %v", result["removed"])
	}

	// Verify preset no longer exists
	presetFile2, _ := store.Load()
	if presetFile2.GetPreset("remove-test") != nil {
		t.Error("preset should have been removed")
	}
}

// TestPresetRemoveActivePreset tests VAL-CLI-029: preset remove on active clears active state
// Also tests VAL-STORAGE-011: Active preset clear on deletion
func TestPresetRemoveActivePreset(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create and set active preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "active-preset",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	presetFile.SetActivePreset("active-preset")
	store.Save(presetFile)

	// Execute remove command
	cmd := createRemoveCommand()
	cmd.SetArgs([]string{"active-preset", "--force"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify was_active is true
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	if result["was_active"] != true {
		t.Errorf("expected was_active=true, got %v", result["was_active"])
	}

	// Verify active preset is now cleared
	presetFile2, _ := store.Load()
	if presetFile2.ActivePreset != "" {
		t.Errorf("active preset should be cleared, got '%s'", presetFile2.ActivePreset)
	}
}

// TestPresetRemoveNotFound tests VAL-CLI-030: preset remove nonexistent returns PRESET_NOT_FOUND
func TestPresetRemoveNotFound(t *testing.T) {
	setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Execute remove command on non-existent preset
	cmd := createRemoveCommand()
	cmd.SetArgs([]string{"nonexistent", "--force"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify error
	if env.OK {
		t.Error("expected OK=false for preset not found")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "PRESET_NOT_FOUND" {
		t.Errorf("expected error code 'PRESET_NOT_FOUND', got '%s'", env.Error.Code)
	}
}

// TestPresetRemoveForceFlag tests VAL-CLI-031: preset remove --force skips confirmation
func TestPresetRemoveForceFlag(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "force-test",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute remove command with --force (no stdin needed)
	cmd := createRemoveCommand()
	cmd.SetArgs([]string{"force-test", "--force"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// This should succeed without prompting
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify success
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if !env.OK {
		t.Errorf("expected OK=true with --force, got false: %v", env.Error)
	}
}

// =============================================================================
// PRESET USE Tests
// =============================================================================

// TestPresetUseSuccess tests VAL-CLI-032: preset use sets active preset
func TestPresetUseSuccess(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "use-test",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute use command
	cmd := createUseCommand()
	cmd.SetArgs([]string{"use-test"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify result contains active_preset
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	if result["active_preset"] != "use-test" {
		t.Errorf("expected active_preset='use-test', got %v", result["active_preset"])
	}

	// Verify it persisted
	presetFile2, _ := store.Load()
	if presetFile2.ActivePreset != "use-test" {
		t.Errorf("active preset should be persisted as 'use-test', got '%s'", presetFile2.ActivePreset)
	}
}

// TestPresetUseNotFound tests VAL-CLI-033: preset use nonexistent returns PRESET_NOT_FOUND
func TestPresetUseNotFound(t *testing.T) {
	setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Execute use command on non-existent preset
	cmd := createUseCommand()
	cmd.SetArgs([]string{"nonexistent"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify error
	if env.OK {
		t.Error("expected OK=false for preset not found")
	}
	if env.Error == nil {
		t.Fatal("expected error to be set")
	}
	if env.Error.Code != "PRESET_NOT_FOUND" {
		t.Errorf("expected error code 'PRESET_NOT_FOUND', got '%s'", env.Error.Code)
	}
}

// TestPresetUseShowCurrent tests VAL-CLI-034: preset use without name shows current active
func TestPresetUseShowCurrent(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create and set active preset
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "current-active",
		Query:       "SELECT 1",
		Description: "active preset",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	presetFile.SetActivePreset("current-active")
	store.Save(presetFile)

	// Execute use command without name
	cmd := createUseCommand()
	cmd.SetArgs([]string{})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify result shows current active preset
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	if result["active_preset"] != "current-active" {
		t.Errorf("expected active_preset='current-active', got %v", result["active_preset"])
	}
}

// TestPresetUseShowCurrentNoneActive tests VAL-CLI-034: preset use shows null when no active
func TestPresetUseShowCurrentNoneActive(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	// Create a preset but don't set it as active
	presetFile, _ := store.Load()
	now := getPresetTestTime()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:      "inactive",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.Save(presetFile)

	// Execute use command without name
	cmd := createUseCommand()
	cmd.SetArgs([]string{})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Verify success
	if !env.OK {
		t.Errorf("expected OK=true, got false: %v", env.Error)
	}

	// Verify result shows null active preset
	result, ok := env.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %v", env.Result)
	}

	if result["active_preset"] != nil {
		t.Errorf("expected active_preset=null, got %v", result["active_preset"])
	}
}

// =============================================================================
// Cross-Area Flow Tests
// =============================================================================

// TestPresetManagementFlow tests VAL-CROSS-004: Preset management flow
func TestPresetManagementFlow(t *testing.T) {
	_, store := setupEditRemoveUseTest(t)
	defer cleanupEditRemoveUseTest(t)

	now := getPresetTestTime()

	// 1. Add a preset (simulating add command result)
	presetFile, _ := store.Load()
	presetFile.SetPreset(&preset.QueryPreset{
		Name:        "flow-test",
		Query:       "SELECT :id",
		Description: "original",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	store.Save(presetFile)

	// 2. Edit the preset
	editCmd := createEditCommand()
	editCmd.SetArgs([]string{"flow-test", "--description", "updated"})
	var editBuf bytes.Buffer
	editCmd.SetOut(&editBuf)
	editCmd.SetErr(&editBuf)
	if err := editCmd.Execute(); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	// 3. Set as active
	useCmd := createUseCommand()
	useCmd.SetArgs([]string{"flow-test"})
	var useBuf bytes.Buffer
	useCmd.SetOut(&useBuf)
	useCmd.SetErr(&useBuf)
	if err := useCmd.Execute(); err != nil {
		t.Fatalf("use failed: %v", err)
	}

	// Verify active
	presetFile2, _ := store.Load()
	if presetFile2.ActivePreset != "flow-test" {
		t.Errorf("expected active preset 'flow-test', got '%s'", presetFile2.ActivePreset)
	}

	// 4. Remove the preset
	removeCmd := createRemoveCommand()
	removeCmd.SetArgs([]string{"flow-test", "--force"})
	var removeBuf bytes.Buffer
	removeCmd.SetOut(&removeBuf)
	removeCmd.SetErr(&removeBuf)
	if err := removeCmd.Execute(); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	// Verify cleanup: preset gone, active cleared
	presetFile3, _ := store.Load()
	if presetFile3.GetPreset("flow-test") != nil {
		t.Error("preset should be removed")
	}
	if presetFile3.ActivePreset != "" {
		t.Errorf("active preset should be cleared, got '%s'", presetFile3.ActivePreset)
	}
}

// getPresetTestTime returns a fixed time for tests
func getPresetTestTime() time.Time {
	return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
}
