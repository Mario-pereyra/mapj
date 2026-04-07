package output

import (
	"encoding/json"
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
