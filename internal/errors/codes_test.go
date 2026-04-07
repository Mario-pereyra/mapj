package errors

import (
	"errors"
	"testing"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitSuccess is 0", ExitSuccess, 0},
		{"ExitError is 1", ExitError, 1},
		{"ExitUsage is 2", ExitUsage, 2},
		{"ExitAuth is 3", ExitAuth, 3},
		{"ExitRetry is 4", ExitRetry, 4},
		{"ExitConflict is 5", ExitConflict, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("got %d, want %d", tt.code, tt.expected)
			}
		})
	}
}

func TestMapErrorToCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, ExitSuccess},
		{"AuthError", &AuthError{Msg: "auth"}, ExitAuth},
		{"UsageError", &UsageError{Msg: "usage"}, ExitUsage},
		{"RetryableError", &RetryableError{Msg: "retry"}, ExitRetry},
		{"ConflictError", &ConflictError{Msg: "conflict"}, ExitConflict},
		{"GeneralError", &GeneralError{Msg: "general"}, ExitError},
		{"untyped error", errors.New("something else"), ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := MapErrorToCode(tt.err)
			if code != tt.expected {
				t.Errorf("MapErrorToCode(%v) = %d, want %d", tt.err, code, tt.expected)
			}
		})
	}
}
