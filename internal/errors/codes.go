package errors

import "strings"

const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitRetry    = 4
	ExitConflict = 5
)

const (
	ErrCodeAuth     = "AUTH_ERROR"
	ErrCodeNotAuth  = "NOT_AUTHENTICATED"
	ErrCodeUsage    = "USAGE_ERROR"
	ErrCodeInvalid  = "INVALID"
	ErrCodeRetry    = "RETRY"
	ErrCodeConflict = "CONFLICT"
)

func MapErrorToCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	errStr := err.Error()

	if strings.Contains(errStr, ErrCodeAuth) || strings.Contains(errStr, ErrCodeNotAuth) {
		return ExitAuth
	}
	if strings.Contains(errStr, ErrCodeUsage) || strings.Contains(errStr, ErrCodeInvalid) {
		return ExitUsage
	}
	if strings.Contains(errStr, ErrCodeRetry) {
		return ExitRetry
	}
	if strings.Contains(errStr, ErrCodeConflict) {
		return ExitConflict
	}

	return ExitError
}
