package logging

import (
	"os"

	"github.com/google/uuid"
)

var (
	// traceID holds the current trace ID for this invocation.
	traceID string
)

// GenerateTraceID generates a new trace ID (UUID v4) if not already set.
// If MAPJ_TRACE_ID environment variable is set, it will be used instead.
func GenerateTraceID() string {
	if traceID == "" {
		traceID = os.Getenv("MAPJ_TRACE_ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
	}
	return traceID
}

// GetTraceID returns the current trace ID.
func GetTraceID() string {
	return traceID
}

// ResetTraceID resets the trace ID (useful for testing).
func ResetTraceID() {
	traceID = ""
}
