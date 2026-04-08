package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Envelope Tests ───────────────────────────────────────────────────────────

func TestEnvelope_Marshal_LLMMode(t *testing.T) {
	// In LLM mode (default), schemaVersion and timestamp are NOT included
	env := NewEnvelope("test command", map[string]string{"key": "value"})

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, true, parsed["ok"])
	assert.Equal(t, "test command", parsed["command"])

	// LLM mode: schemaVersion and timestamp must NOT be present (noise reduction)
	assert.Nil(t, parsed["schemaVersion"], "LLM mode must omit schemaVersion")
	assert.Nil(t, parsed["timestamp"], "LLM mode must omit timestamp")
}

func TestEnvelope_Error(t *testing.T) {
	env := NewErrorEnvelope("test command", "TEST_ERROR", "something went wrong", false)

	assert.Equal(t, false, env.OK)
	assert.NotNil(t, env.Error)
	assert.Equal(t, "TEST_ERROR", env.Error.Code)
	assert.Equal(t, "something went wrong", env.Error.Message)
	assert.Equal(t, false, env.Error.Retryable)
	assert.Empty(t, env.Error.Hint) // no hint by default
}

func TestEnvelope_ErrorRetryable(t *testing.T) {
	env := NewErrorEnvelope("test command", "RATE_LIMIT", "too many requests", true)
	assert.Equal(t, true, env.Error.Retryable)
}

func TestEnvelope_ErrorWithHint(t *testing.T) {
	env := NewErrorEnvelopeWithHint("test command", "AUTH_ERROR", "no token", "run mapj auth login first", false)
	assert.Equal(t, "run mapj auth login first", env.Error.Hint)
}

// ─── Formatter Tests ──────────────────────────────────────────────────────────

func TestLLMFormatter_Compact(t *testing.T) {
	formatter := LLMFormatter{}
	env := NewEnvelope("test", "result")

	output := formatter.Format(env)

	// Must be valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Must be compact (no newlines, no indentation)
	assert.NotContains(t, output, "\n", "LLM formatter must produce compact JSON (no newlines)")
	assert.Equal(t, "result", parsed["result"])
	assert.Nil(t, parsed["schemaVersion"], "LLM formatter must omit schemaVersion")
	assert.Nil(t, parsed["timestamp"], "LLM formatter must omit timestamp")
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format   string
		expected Formatter
	}{
		{"llm", LLMFormatter{}},
		{"", AutoFormatter{}},        // empty = auto (default)
		{"unknown", AutoFormatter{}}, // unknown = auto (safe default)
		{"json", LLMFormatter{}},
		{"toon", TOONFormatter{}},
		{"JSON", LLMFormatter{}}, // case-insensitive
		{"LLM", LLMFormatter{}},
		{"TOON", TOONFormatter{}}, // case-insensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := NewFormatter(tt.format)
			assert.IsType(t, tt.expected, formatter)
		})
	}
}

// ─── Verbose Mode Tests (VAL-CLI-039) ────────────────────────────────────────

func TestLLMFormatter_Verbose(t *testing.T) {
	formatter := LLMFormatter{Verbose: true}
	env := NewEnvelope("test command", map[string]string{"key": "value"})

	output := formatter.Format(env)

	// Must be valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verbose mode: schemaVersion and timestamp MUST be present
	assert.Equal(t, "1.0", parsed["schemaVersion"], "Verbose mode must include schemaVersion")
	assert.NotNil(t, parsed["timestamp"], "Verbose mode must include timestamp")
	assert.NotEmpty(t, parsed["timestamp"], "Timestamp must not be empty")
}

func TestLLMFormatter_VerboseWithError(t *testing.T) {
	formatter := LLMFormatter{Verbose: true}
	env := NewErrorEnvelopeWithHint("test", "ERROR_CODE", "error message", "try this hint", true)

	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verify error structure
	assert.Equal(t, false, parsed["ok"])
	errObj := parsed["error"].(map[string]interface{})
	assert.Equal(t, "ERROR_CODE", errObj["code"])
	assert.Equal(t, "error message", errObj["message"])
	assert.Equal(t, "try this hint", errObj["hint"])
	assert.Equal(t, true, errObj["retryable"])

	// Verbose mode fields
	assert.Equal(t, "1.0", parsed["schemaVersion"])
	assert.NotNil(t, parsed["timestamp"])
}

func TestTOONFormatter_Verbose(t *testing.T) {
	formatter := TOONFormatter{Verbose: true}
	env := NewEnvelope("test command", "result")

	output := formatter.Format(env)

	// Verbose mode must include schemaVersion and timestamp in TOON format
	assert.Contains(t, output, "schemaVersion: 1.0")
	assert.Contains(t, output, "timestamp:")
}

// ─── Error Structure Tests (VAL-CLI-035) ──────────────────────────────────────

func TestErrorStructure_AllFields(t *testing.T) {
	// Test complete error structure with all fields
	env := NewErrorEnvelopeWithHint("mapj protheus preset run", "PRESET_NOT_FOUND", "preset 'nonexistent' not found", "Use 'mapj protheus preset list' to see available presets", false)

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verify structure: {ok: false, error: {code, message, hint, retryable}}
	assert.Equal(t, false, parsed["ok"])
	assert.NotNil(t, parsed["error"])
	assert.Nil(t, parsed["result"])

	errObj := parsed["error"].(map[string]interface{})
	assert.Equal(t, "PRESET_NOT_FOUND", errObj["code"], "Error code must be UPPER_SNAKE_CASE")
	assert.Equal(t, "preset 'nonexistent' not found", errObj["message"])
	assert.Equal(t, "Use 'mapj protheus preset list' to see available presets", errObj["hint"])
	assert.Equal(t, false, errObj["retryable"])
}

func TestErrorStructure_Retryable(t *testing.T) {
	env := NewErrorEnvelope("mapj protheus query", "CONNECTION_FAILED", "connection timeout", true)

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	errObj := parsed["error"].(map[string]interface{})
	assert.Equal(t, true, errObj["retryable"], "Retryable errors must have retryable: true")
}

func TestErrorStructure_NonRetryable(t *testing.T) {
	env := NewErrorEnvelope("mapj protheus preset add", "PRESET_EXISTS", "preset 'test' already exists", false)

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	errObj := parsed["error"].(map[string]interface{})
	assert.Equal(t, false, errObj["retryable"], "Non-retryable errors must have retryable: false")
}

// ─── Success Structure Tests (VAL-CLI-036) ─────────────────────────────────────

func TestSuccessStructure_Basic(t *testing.T) {
	env := NewEnvelope("mapj protheus preset list", map[string]interface{}{
		"presets": []string{"preset1", "preset2"},
		"count":   2,
	})

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verify structure: {ok: true, ...}
	assert.Equal(t, true, parsed["ok"])
	assert.Equal(t, "mapj protheus preset list", parsed["command"])
	assert.NotNil(t, parsed["result"])
	assert.Nil(t, parsed["error"])
}

func TestSuccessStructure_WithResult(t *testing.T) {
	result := map[string]interface{}{
		"name":        "test-preset",
		"query":       "SELECT :name FROM users",
		"parameters":  []map[string]string{{"name": "name", "type": "string", "required": "true"}},
		"createdAt":   "2024-01-15T10:30:00Z",
		"updatedAt":   "2024-01-15T10:30:00Z",
	}
	env := NewEnvelope("mapj protheus preset show", result)

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, true, parsed["ok"])
	resultObj := parsed["result"].(map[string]interface{})
	assert.Equal(t, "test-preset", resultObj["name"])
	assert.Equal(t, "SELECT :name FROM users", resultObj["query"])
}

// ─── JSON Output Flag Tests (VAL-CLI-038) ─────────────────────────────────────

func TestJSONOutput_PureJSON(t *testing.T) {
	// --json must produce pure JSON without colors or decorations
	env := NewEnvelope("mapj protheus preset list", map[string]interface{}{
		"presets": []map[string]string{{"name": "test"}},
	})

	formatter := LLMFormatter{Verbose: false}
	output := formatter.Format(env)

	// Must be valid JSON
	assert.True(t, json.Valid([]byte(output)), "Output must be valid JSON")

	// Must be compact (no decorations)
	assert.NotContains(t, output, "\n", "JSON output must be compact (no newlines)")

	// Must not contain ANSI color codes
	assert.NotContains(t, output, "\x1b[", "JSON output must not contain ANSI color codes")
	assert.NotContains(t, output, "\033[", "JSON output must not contain ANSI escape sequences")
}

func TestJSONOutput_SuitableForJQ(t *testing.T) {
	// --json output must be suitable for piping to jq
	env := NewEnvelope("mapj protheus preset list", map[string]interface{}{
		"presets": []string{"a", "b"},
		"count":   2,
	})

	formatter := LLMFormatter{}
	output := formatter.Format(env)

	// Must be parseable as JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "Output must be parseable by jq")

	// Must contain expected fields
	assert.Equal(t, true, parsed["ok"])

	// Count is inside result object, not at top level
	result := parsed["result"].(map[string]interface{})
	assert.Equal(t, float64(2), result["count"])
}

// ─── NewFormatterWithVerbose Tests ────────────────────────────────────────────

func TestNewFormatterWithVerbose(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		verbose  bool
		expected Formatter
	}{
		{"llm_verbose", "llm", true, LLMFormatter{Verbose: true}},
		{"llm_non_verbose", "llm", false, LLMFormatter{Verbose: false}},
		{"json_verbose", "json", true, LLMFormatter{Verbose: true}},
		{"toon_verbose", "toon", true, TOONFormatter{Verbose: true}},
		{"auto_verbose", "", true, AutoFormatter{Verbose: true}},
		{"auto_non_verbose", "", false, AutoFormatter{Verbose: false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatterWithVerbose(tt.format, tt.verbose)
			assert.Equal(t, tt.expected, formatter)
		})
	}
}

// ─── TOON Error Structure Tests ───────────────────────────────────────────────

func TestTOONErrorStructure(t *testing.T) {
	env := NewErrorEnvelopeWithHint("test", "MISSING_PARAMETER", "missing required parameter 'name'", "--param name=value", false)

	formatter := TOONFormatter{}
	output := formatter.Format(env)

	// Verify TOON error format
	assert.Contains(t, output, "ok: false")
	assert.Contains(t, output, "command: test")
	assert.Contains(t, output, "error:")
	assert.Contains(t, output, "code: MISSING_PARAMETER")
	// Strings with special characters (like ') are quoted in TOON format
	assert.Contains(t, output, `message: "missing required parameter 'name'"`)
	assert.Contains(t, output, `hint: "--param name=value"`)
	// VAL-CLI-035: retryable field must always be present
	assert.Contains(t, output, "retryable: false")
}

// ─── Error Codes UPPER_SNAKE_CASE Tests ───────────────────────────────────────

func TestErrorCodes_UpperSnakeCase(t *testing.T) {
	// Common error codes must follow UPPER_SNAKE_CASE convention
	codes := []string{
		"PRESET_NOT_FOUND",
		"MISSING_PARAMETER",
		"MISSING_REQUIRED_FIELD",
		"PRESET_EXISTS",
		"INVALID_PARAM_DEF",
		"TYPE_MISMATCH",
		"SQL_INJECTION_DETECTED",
		"CONNECTION_FAILED",
		"NO_CONNECTION",
		"USAGE_ERROR",
		"AUTH_ERROR",
		"PROFILE_NOT_FOUND",
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			env := NewErrorEnvelope("test", code, "test message", false)
			formatter := LLMFormatter{}
			output := formatter.Format(env)

			var parsed map[string]interface{}
			err := json.Unmarshal([]byte(output), &parsed)
			require.NoError(t, err)

			errObj := parsed["error"].(map[string]interface{})
			assert.Equal(t, code, errObj["code"])

			// Verify code format: uppercase letters, underscores, no lowercase
			parsedCode := errObj["code"].(string)
			assert.Equal(t, strings.ToUpper(parsedCode), parsedCode, "Error code must be uppercase")
			assert.NotContains(t, parsedCode, " ", "Error code must not contain spaces")
			assert.NotContains(t, parsedCode, "-", "Error code must not contain hyphens")
		})
	}
}
