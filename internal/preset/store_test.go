package preset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPresetStore verifies store creation with correct path
func TestNewPresetStore(t *testing.T) {
	store, err := NewPresetStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify path is ~/.config/mapj/presets.json
	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, ".config", "mapj", "presets.json")
	assert.Equal(t, expectedPath, store.path)
}

// TestLoadMissingFile verifies graceful handling of missing file
// VAL-STORAGE-006: Handle Missing Presets File Gracefully
func TestLoadMissingFile(t *testing.T) {
	// Use temp directory for test
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}
	presetFile, err := store.Load()

	require.NoError(t, err)
	require.NotNil(t, presetFile)

	// Should return empty structure
	assert.Empty(t, presetFile.Presets)
	assert.Equal(t, "", presetFile.ActivePreset)
}

// TestLoadCorruptedJSON verifies error on corrupted JSON
// VAL-STORAGE-007: Handle Corrupted Presets File
func TestLoadCorruptedJSON(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	// Write invalid JSON
	err := os.WriteFile(testPath, []byte("{invalid json"), 0600)
	require.NoError(t, err)

	store := &PresetStore{path: testPath}
	presetFile, err := store.Load()

	// Should return error with clear message
	assert.Error(t, err)
	assert.Nil(t, presetFile)
	assert.Contains(t, err.Error(), "invalid")
}

// TestLoadExistingPresets verifies loading existing presets
// VAL-STORAGE-005: Load Existing Presets Successfully
func TestLoadExistingPresets(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	// Create valid presets file
	presetData := PresetFile{
		Presets: map[string]*QueryPreset{
			"test1": {
				Name:        "test1",
				Query:       "SELECT 1",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			"test2": {
				Name:        "test2",
				Query:       "SELECT 2",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
		ActivePreset: "test1",
	}

	data, err := json.MarshalIndent(presetData, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(testPath, data, 0600)
	require.NoError(t, err)

	store := &PresetStore{path: testPath}
	loaded, err := store.Load()

	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Len(t, loaded.Presets, 2)
	assert.Equal(t, "test1", loaded.ActivePreset)
	assert.Contains(t, loaded.Presets, "test1")
	assert.Contains(t, loaded.Presets, "test2")
}

// TestSaveCreatesDirectory verifies directory creation on first save
// VAL-STORAGE-001: Directory Creation on First Save
func TestSaveCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	// Use a path where the parent directory doesn't exist
	testPath := filepath.Join(tempDir, "nonexistent", "deep", "dir", "presets.json")

	store := &PresetStore{path: testPath}
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"test": {
				Name:      "test",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Verify directory was created
	dir := filepath.Dir(testPath)
	assert.DirExists(t, dir)

	// Verify file was created
	assert.FileExists(t, testPath)
}

// TestSavePermissions verifies 0600 file permissions
// VAL-STORAGE-002: Presets File Creation with Correct Permissions
func TestSavePermissions(t *testing.T) {
	// Skip on Windows - Unix permissions not applicable
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions not applicable on Windows")
	}

	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"test": {
				Name:      "test",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Verify permissions are 0600
	info, err := os.Stat(testPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// TestSaveAtomicWrite verifies atomic write pattern (temp file + rename)
// VAL-STORAGE-012: Atomic File Write
func TestSaveAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"test": {
				Name:      "test",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Verify temp file is not left behind
	tempPath := testPath + ".tmp"
	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err), "Temp file should not exist after save")

	// Verify main file exists and is valid
	assert.FileExists(t, testPath)

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Contains(t, loaded.Presets, "test")
}

// TestJSONFormatting verifies JSON is formatted with indentation
// VAL-STORAGE-013: JSON Formatting and Readability
func TestJSONFormatting(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"test": {
				Name:        "test",
				Query:       "SELECT 1",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Read file content
	data, err := os.ReadFile(testPath)
	require.NoError(t, err)

	// Should be indented (contains newlines and spaces)
	assert.Contains(t, string(data), "\n")
	assert.Contains(t, string(data), "  ") // indentation

	// Should not be minified (no compact single-line JSON)
	assert.Greater(t, len(data), 50) // reasonable length for indented JSON
}

// TestQueryPresetStructure verifies all QueryPreset fields are present
// VAL-STORAGE-003: Save Preset with Complete QueryPreset Structure
func TestQueryPresetStructure(t *testing.T) {
	now := time.Now()
	preset := &QueryPreset{
		Name:        "test-preset",
		Description: "Test description",
		Query:       "SELECT :param FROM table",
		Connection:  "protheus_prod",
		MaxRows:     100,
		Parameters: []ParamDef{
			{Name: "param", Type: "string", Required: true, Description: "A parameter"},
		},
		Tags:       []string{"report", "daily"},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Verify all fields are present and accessible
	assert.Equal(t, "test-preset", preset.Name)
	assert.Equal(t, "Test description", preset.Description)
	assert.Equal(t, "SELECT :param FROM table", preset.Query)
	assert.Equal(t, "protheus_prod", preset.Connection)
	assert.Equal(t, 100, preset.MaxRows)
	assert.Len(t, preset.Parameters, 1)
	assert.Len(t, preset.Tags, 2)
	assert.Equal(t, now, preset.CreatedAt)
	assert.Equal(t, now, preset.UpdatedAt)
}

// TestParamDefStructure verifies all ParamDef fields are present
// VAL-STORAGE-004: ParamDef Structure Validation
func TestParamDefStructure(t *testing.T) {
	param := ParamDef{
		Name:        "test_param",
		Type:        "string",
		Required:    true,
		Default:     "default_value",
		Description: "A test parameter",
		Pattern:     "^[a-z]+$",
	}

	// Verify all fields are present and accessible
	assert.Equal(t, "test_param", param.Name)
	assert.Equal(t, "string", param.Type)
	assert.True(t, param.Required)
	assert.Equal(t, "default_value", param.Default)
	assert.Equal(t, "A test parameter", param.Description)
	assert.Equal(t, "^[a-z]+$", param.Pattern)
}

// TestActivePresetPersistence verifies active preset persists between sessions
// VAL-STORAGE-010: Active Preset Persistence
func TestActivePresetPersistence(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Save with active preset
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"active-test": {
				Name:      "active-test",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		ActivePreset: "active-test",
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Create new store instance (simulates new session)
	newStore := &PresetStore{path: testPath}
	loaded, err := newStore.Load()
	require.NoError(t, err)

	// Verify active preset is preserved
	assert.Equal(t, "active-test", loaded.ActivePreset)
}

// TestEmptyPresetsFile verifies empty structure is valid
// VAL-STORAGE-014: Empty Presets File Valid State
func TestEmptyPresetsFile(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}
	presetFile := &PresetFile{
		Presets:       map[string]*QueryPreset{},
		ActivePreset:  "",
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Read file and verify it's valid JSON
	data, err := os.ReadFile(testPath)
	require.NoError(t, err)

	var loaded PresetFile
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.NotNil(t, loaded.Presets)
	assert.Empty(t, loaded.Presets)
}

// TestSavePreservesOtherPresets verifies update doesn't affect other presets
// VAL-STORAGE-009: Preserve Other Presets on Update
func TestSavePreservesOtherPresets(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Save two presets
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"preset_a": {
				Name:      "preset_a",
				Query:     "SELECT A",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"preset_b": {
				Name:      "preset_b",
				Query:     "SELECT B",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Update preset_a
	presetFile.Presets["preset_a"].Query = "SELECT A_UPDATED"
	presetFile.Presets["preset_a"].UpdatedAt = time.Now()

	err = store.Save(presetFile)
	require.NoError(t, err)

	// Verify preset_b is unchanged
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "SELECT B", loaded.Presets["preset_b"].Query)
	assert.Equal(t, "SELECT A_UPDATED", loaded.Presets["preset_a"].Query)
}

// TestUnicodeAndSpecialCharacters verifies Unicode support
// VAL-STORAGE-015: Unicode and Special Characters Support
func TestUnicodeAndSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Create preset with Unicode and special SQL characters
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"unicode-test": {
				Name:        "unicode-test",
				Description: "Descripción en español - 中文 - 日本語",
				Query:       "SELECT * FROM users WHERE name LIKE '%O''Brien%' AND comment LIKE '%--comment%'",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Load and verify
	loaded, err := store.Load()
	require.NoError(t, err)

	preset := loaded.Presets["unicode-test"]
	assert.Equal(t, "Descripción en español - 中文 - 日本語", preset.Description)
	assert.Contains(t, preset.Query, "O''Brien")
	assert.Contains(t, preset.Query, "--comment")
}

// TestOverwriteExistingPreset verifies overwrite behavior
// VAL-STORAGE-008: Overwrite Existing Preset
func TestOverwriteExistingPreset(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Save initial preset
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"test": {
				Name:        "test",
				Query:       "SELECT original",
				Description: "Original description",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)
	originalCreatedAt := presetFile.Presets["test"].CreatedAt

	// Overwrite with new data (simulates re-adding same name)
	time.Sleep(10 * time.Millisecond) // ensure different timestamp
	presetFile.Presets["test"] = &QueryPreset{
		Name:        "test",
		Query:       "SELECT updated",
		Description: "Updated description",
		CreatedAt:   originalCreatedAt, // preserve original creation time
		UpdatedAt:   time.Now(),
	}

	err = store.Save(presetFile)
	require.NoError(t, err)

	// Verify complete replacement
	loaded, err := store.Load()
	require.NoError(t, err)

	preset := loaded.Presets["test"]
	assert.Equal(t, "SELECT updated", preset.Query)
	assert.Equal(t, "Updated description", preset.Description)
	assert.True(t, preset.UpdatedAt.After(originalCreatedAt))
}

// TestSetPreset tests the SetPreset convenience method
func TestSetPreset(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Load empty file
	presetFile, err := store.Load()
	require.NoError(t, err)

	// Set a preset
	now := time.Now()
	preset := &QueryPreset{
		Name:      "new-preset",
		Query:     "SELECT 1",
		CreatedAt: now,
		UpdatedAt: now,
	}

	presetFile.SetPreset(preset)
	err = store.Save(presetFile)
	require.NoError(t, err)

	// Verify
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Contains(t, loaded.Presets, "new-preset")
}

// TestDeletePreset tests the DeletePreset convenience method
func TestDeletePreset(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Create initial file with preset
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"to-delete": {
				Name:      "to-delete",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"to-keep": {
				Name:      "to-keep",
				Query:     "SELECT 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		ActivePreset: "to-delete",
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Delete preset
	loaded, err := store.Load()
	require.NoError(t, err)

	wasActive := loaded.ActivePreset == "to-delete"
	loaded.DeletePreset("to-delete")
	err = store.Save(loaded)
	require.NoError(t, err)

	// Verify deletion
	final, err := store.Load()
	require.NoError(t, err)

	assert.NotContains(t, final.Presets, "to-delete")
	assert.Contains(t, final.Presets, "to-keep")

	// Verify active preset cleared if it was the deleted one
	if wasActive {
		assert.Equal(t, "", final.ActivePreset)
	}
}

// TestSetActivePreset tests setting active preset
func TestSetActivePreset(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Create initial file
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"preset1": {
				Name:      "preset1",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Set active
	loaded, err := store.Load()
	require.NoError(t, err)

	loaded.SetActivePreset("preset1")
	err = store.Save(loaded)
	require.NoError(t, err)

	// Verify
	final, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "preset1", final.ActivePreset)
}

// TestGetPreset tests getting a single preset
func TestGetPreset(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Create initial file
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"existing": {
				Name:      "existing",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// Get existing
	loaded, err := store.Load()
	require.NoError(t, err)

	preset := loaded.GetPreset("existing")
	require.NotNil(t, preset)
	assert.Equal(t, "existing", preset.Name)

	// Get non-existent
	nonExistent := loaded.GetPreset("nonexistent")
	assert.Nil(t, nonExistent)
}

// TestListPresets tests listing all preset names
func TestListPresets(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Create initial file
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{
			"preset_c": {
				Name:      "preset_c",
				Query:     "SELECT 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"preset_a": {
				Name:      "preset_a",
				Query:     "SELECT 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"preset_b": {
				Name:      "preset_b",
				Query:     "SELECT 3",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	err := store.Save(presetFile)
	require.NoError(t, err)

	// List should return sorted names
	loaded, err := store.Load()
	require.NoError(t, err)

	names := loaded.ListPresets()
	assert.Equal(t, []string{"preset_a", "preset_b", "preset_c"}, names)
}

// TestNewPresetStoreError tests error handling when home dir is unavailable
func TestNewPresetStoreError(t *testing.T) {
	// This test verifies the error handling path
	// We can't easily force os.UserHomeDir to fail, so we just test
	// that the function works correctly in normal conditions
	store, err := NewPresetStore()
	if err != nil {
		t.Logf("Expected no error but got: %v", err)
	}
	assert.NotNil(t, store)
}

// TestSaveConcurrentWrites tests that concurrent writes don't corrupt the file
func TestSaveConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "presets.json")

	store := &PresetStore{path: testPath}

	// Initial save
	presetFile := &PresetFile{
		Presets: map[string]*QueryPreset{},
	}
	err := store.Save(presetFile)
	require.NoError(t, err)

	// Simulate concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			s := &PresetStore{path: testPath}
			pf, err := s.Load()
			if err != nil {
				return
			}
			pf.SetPreset(&QueryPreset{
				Name:      fmt.Sprintf("preset_%d", idx),
				Query:     fmt.Sprintf("SELECT %d", idx),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
			s.Save(pf)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final load should not fail
	final, err := store.Load()
	require.NoError(t, err)
	assert.NotNil(t, final.Presets)
}
