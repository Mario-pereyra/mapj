package output

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelope_Marshal(t *testing.T) {
	env := NewEnvelope("test command", map[string]string{"key": "value"})

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, true, parsed["ok"])
	assert.Equal(t, "test command", parsed["command"])
	assert.Equal(t, "1.0", parsed["schemaVersion"])
	assert.NotEmpty(t, parsed["timestamp"])
}

func TestEnvelope_Error(t *testing.T) {
	env := NewErrorEnvelope("test command", "TEST_ERROR", "something went wrong", false)

	assert.Equal(t, false, env.OK)
	assert.NotNil(t, env.Error)
	assert.Equal(t, "TEST_ERROR", env.Error.Code)
	assert.Equal(t, "something went wrong", env.Error.Message)
	assert.Equal(t, false, env.Error.Retryable)
}

func TestEnvelope_ErrorRetryable(t *testing.T) {
	env := NewErrorEnvelope("test command", "RATE_LIMIT", "too many requests", true)

	assert.Equal(t, true, env.Error.Retryable)
}

func TestJSONFormatter(t *testing.T) {
	formatter := JSONFormatter{}
	env := NewEnvelope("test", "result")

	output := formatter.Format(env)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "result", parsed["result"])
}

func TestTableFormatter(t *testing.T) {
	formatter := TableFormatter{}

	env := NewEnvelope("test", map[string]string{"key": "value"})
	output := formatter.Format(env)
	assert.Contains(t, output, "OK")
	assert.Contains(t, output, "key:value")

	errEnv := NewErrorEnvelope("test", "ERR", "error message", false)
	output = formatter.Format(errEnv)
	assert.Contains(t, output, "ERR")
	assert.Contains(t, output, "error message")
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format string
		isType interface{}
	}{
		{"json", JSONFormatter{}},
		{"table", TableFormatter{}},
		{"unknown", JSONFormatter{}},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := NewFormatter(tt.format)
			assert.IsType(t, tt.isType, formatter)
		})
	}
}
