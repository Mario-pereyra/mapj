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

func TestEnvelope_Marshal_HumanMode(t *testing.T) {
	// In human mode, withHumanFields() adds schemaVersion and timestamp
	env := NewEnvelope("test command", map[string]string{"key": "value"})
	env.withHumanFields()

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0", parsed["schemaVersion"], "Human mode must include schemaVersion")
	assert.NotEmpty(t, parsed["timestamp"], "Human mode must include timestamp")
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

func TestHumanFormatter_Pretty(t *testing.T) {
	formatter := HumanFormatter{}
	env := NewEnvelope("test", "result")

	output := formatter.Format(env)

	// Must be valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Must be indented
	assert.Contains(t, output, "  ", "Human formatter must produce indented JSON")
	assert.Contains(t, output, "\n", "Human formatter must produce multi-line JSON")
	assert.Equal(t, "1.0", parsed["schemaVersion"], "Human formatter must include schemaVersion")
	assert.NotEmpty(t, parsed["timestamp"], "Human formatter must include timestamp")
}

func TestHumanFormatter_Error(t *testing.T) {
	formatter := HumanFormatter{}
	errEnv := NewErrorEnvelope("test", "ERR", "error message", false)
	output := formatter.Format(errEnv)
	assert.Contains(t, output, "ERR")
	assert.Contains(t, output, "error message")
}

func TestCSVFormatter_RFC4180(t *testing.T) {
	formatter := CSVFormatter{}
	payload := &CSVPayload{
		Headers: []string{"name", "value", "note"},
		Rows: [][]string{
			{"Alice", "100", "no comma"},
			{"Bob, Jr.", "200", `has "quotes"`},
			{"Carol", "300", "normal"},
		},
	}
	env := NewEnvelope("test", payload)
	output := formatter.Format(env)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 4, len(lines), "CSV must have header + 3 data rows")
	assert.Equal(t, "name,value,note", lines[0], "First line must be headers")
	assert.Equal(t, "Alice,100,no comma", lines[1], "Alice row must have no quoting")
	// Bob's name contains a comma — must be quoted
	assert.Contains(t, lines[2], `"Bob, Jr."`, "Field with comma must be quoted")
	// Bob's note contains quotes — must be double-quoted
	assert.Contains(t, lines[2], `"has ""quotes"""`, "Field with quotes must be double-quoted")
}

func TestCSVFormatter_Fallback(t *testing.T) {
	// Non-CSVPayload result falls back to LLM compact JSON
	formatter := CSVFormatter{}
	env := NewEnvelope("test", map[string]string{"key": "val"})
	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "CSV formatter fallback must produce valid JSON")
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format   string
		expected Formatter
	}{
		{"llm", LLMFormatter{}},
		{"", LLMFormatter{}},          // empty = llm (default)
		{"unknown", LLMFormatter{}},   // unknown = llm (safe default)
		{"json", HumanFormatter{}},
		{"human", HumanFormatter{}},
		{"table", HumanFormatter{}},   // table is alias for human
		{"csv", CSVFormatter{}},
		{"JSON", HumanFormatter{}},    // case-insensitive
		{"LLM", LLMFormatter{}},
		{"CSV", CSVFormatter{}},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := NewFormatter(tt.format)
			assert.IsType(t, tt.expected, formatter)
		})
	}
}
