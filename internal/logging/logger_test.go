package logging

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateTraceID(t *testing.T) {
	// Reset trace ID before each test
	ResetTraceID()
	
	traceID := GenerateTraceID()
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 36) // UUID v4 format
	
	// Calling again should return the same trace ID
	traceID2 := GenerateTraceID()
	assert.Equal(t, traceID, traceID2)
}

func TestGenerateTraceIDWithEnvOverride(t *testing.T) {
	ResetTraceID()
	os.Setenv("MAPJ_TRACE_ID", "custom-trace-id-123")
	defer os.Unsetenv("MAPJ_TRACE_ID")
	
	traceID := GenerateTraceID()
	assert.Equal(t, "custom-trace-id-123", traceID)
}

func TestResetTraceID(t *testing.T) {
	ResetTraceID()
	traceID1 := GenerateTraceID()
	ResetTraceID()
	traceID2 := GenerateTraceID()
	
	assert.NotEqual(t, traceID1, traceID2)
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"error", "error"},
		{"DEBUG", "debug"},
		{"INFO", "info"},
		{"invalid", "info"}, // default
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			expected := parseLevel(tt.expected)
			assert.Equal(t, expected, level)
		})
	}
}

func TestInit(t *testing.T) {
	ResetTraceID()
	GenerateTraceID()
	
	cfg := Config{
		Level:   "debug",
		TraceID: "test-trace-123",
	}
	
	Init(cfg)
	
	assert.Equal(t, "debug", LogLevel)
}

func TestSetLevel(t *testing.T) {
	SetLevel("warn")
	assert.Equal(t, "warn", LogLevel)
	
	SetLevel("error")
	assert.Equal(t, "error", LogLevel)
}

func TestGetLevel(t *testing.T) {
	SetLevel("debug")
	assert.Equal(t, "debug", GetLevel())
}

func TestTraceIDFormat(t *testing.T) {
	ResetTraceID()
	traceID := GenerateTraceID()
	
	// UUID v4 format: 8-4-4-4-12 = 36 characters
	assert.Len(t, traceID, 36)
	
	// Should contain 4 dashes
	dashCount := strings.Count(traceID, "-")
	assert.Equal(t, 4, dashCount)
}
