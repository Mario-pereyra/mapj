// Package preset provides storage and management for query presets.
// Presets allow users to save, reuse, and parameterize frequently used SQL queries.
package preset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// QueryPreset represents a saved query with optional parameters and metadata.
// VAL-STORAGE-003: Complete QueryPreset structure
type QueryPreset struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Query       string     `json:"query"`
	Connection  string     `json:"connection,omitempty"`
	MaxRows     int        `json:"maxRows,omitempty"`
	Parameters  []ParamDef `json:"parameters,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// ParamDef defines a parameter's type, constraints, and defaults.
// VAL-STORAGE-004: Complete ParamDef structure
type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`                 // string, int, date, datetime, bool, list
	Required    bool   `json:"required"`             // true if parameter must be provided
	Default     string `json:"default,omitempty"`    // default value if not provided
	Description string `json:"description,omitempty"` // human-readable description
	Pattern     string `json:"pattern,omitempty"`    // regex pattern for validation (optional)
}

// PresetFile represents the JSON file structure containing all presets.
type PresetFile struct {
	Presets       map[string]*QueryPreset `json:"presets"`
	ActivePreset  string                  `json:"activePreset,omitempty"`
}

// PresetStore manages the persistence of presets to a JSON file.
type PresetStore struct {
	path string
}

// SetPath sets a custom path for the preset store (used for testing).
func (s *PresetStore) SetPath(path string) {
	s.path = path
}

// GetPath returns the current path of the preset store.
func (s *PresetStore) GetPath() string {
	return s.path
}

// NewPresetStore creates a new preset store with the default path.
// The path is ~/.config/mapj/presets.json
func NewPresetStore() (*PresetStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "mapj")
	path := filepath.Join(configDir, "presets.json")

	return &PresetStore{path: path}, nil
}

// Load reads the presets file and returns the PresetFile structure.
// VAL-STORAGE-005: Load existing presets successfully
// VAL-STORAGE-006: Handle missing file gracefully
// VAL-STORAGE-007: Handle corrupted JSON with error
func (s *PresetStore) Load() (*PresetFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty structure for missing file
			return &PresetFile{
				Presets:      make(map[string]*QueryPreset),
				ActivePreset: "",
			}, nil
		}
		return nil, fmt.Errorf("failed to read presets file: %w", err)
	}

	var presetFile PresetFile
	if err := json.Unmarshal(data, &presetFile); err != nil {
		return nil, fmt.Errorf("invalid presets file format: %w", err)
	}

	// Ensure Presets map is initialized
	if presetFile.Presets == nil {
		presetFile.Presets = make(map[string]*QueryPreset)
	}

	return &presetFile, nil
}

// Save writes the PresetFile to disk with atomic write pattern.
// VAL-STORAGE-001: Creates directory if it doesn't exist
// VAL-STORAGE-002: Sets file permissions to 0600
// VAL-STORAGE-012: Uses atomic write (temp file + rename)
// VAL-STORAGE-013: Formats JSON with indentation
func (s *PresetStore) Save(presetFile *PresetFile) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(presetFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal presets: %w", err)
	}

	// Add trailing newline for clean file format
	data = append(data, '\n')

	// Atomic write: write to temp file, then rename
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp presets file: %w", err)
	}

	// Rename temp file to final path (atomic on most systems)
	if err := os.Rename(tempPath, s.path); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename presets file: %w", err)
	}

	return nil
}

// GetPreset returns a preset by name, or nil if not found.
func (pf *PresetFile) GetPreset(name string) *QueryPreset {
	if pf.Presets == nil {
		return nil
	}
	return pf.Presets[name]
}

// SetPreset adds or updates a preset in the file.
func (pf *PresetFile) SetPreset(preset *QueryPreset) {
	if pf.Presets == nil {
		pf.Presets = make(map[string]*QueryPreset)
	}
	pf.Presets[preset.Name] = preset
}

// DeletePreset removes a preset by name.
// If the deleted preset is the active preset, clears ActivePreset.
func (pf *PresetFile) DeletePreset(name string) {
	if pf.Presets == nil {
		return
	}

	// Clear active preset if it's being deleted
	if pf.ActivePreset == name {
		pf.ActivePreset = ""
	}

	delete(pf.Presets, name)
}

// SetActivePreset sets the active preset name.
func (pf *PresetFile) SetActivePreset(name string) {
	pf.ActivePreset = name
}

// GetActivePreset returns the active preset, or nil if none is set.
func (pf *PresetFile) GetActivePreset() *QueryPreset {
	if pf.ActivePreset == "" || pf.Presets == nil {
		return nil
	}
	return pf.Presets[pf.ActivePreset]
}

// ListPresets returns sorted list of preset names.
func (pf *PresetFile) ListPresets() []string {
	if pf.Presets == nil {
		return []string{}
	}

	names := make([]string, 0, len(pf.Presets))
	for name := range pf.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasPresets returns true if there are any presets.
func (pf *PresetFile) HasPresets() bool {
	return len(pf.Presets) > 0
}

// PresetCount returns the number of presets.
func (pf *PresetFile) PresetCount() int {
	return len(pf.Presets)
}
