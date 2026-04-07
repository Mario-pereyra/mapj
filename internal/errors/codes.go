package errors

const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitRetry    = 4
	ExitConflict = 5
)

type ExitCoder interface {
	ExitCode() int
}

type AuthError struct {
	Msg string
}

func (e *AuthError) Error() string { return e.Msg }
func (e *AuthError) ExitCode() int { return ExitAuth }

type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }
func (e *UsageError) ExitCode() int { return ExitUsage }

type RetryableError struct {
	Msg string
}

func (e *RetryableError) Error() string { return e.Msg }
func (e *RetryableError) ExitCode() int { return ExitRetry }

type ConflictError struct {
	Msg string
}

func (e *ConflictError) Error() string { return e.Msg }
func (e *ConflictError) ExitCode() int { return ExitConflict }

type GeneralError struct {
	Msg string
}

func (e *GeneralError) Error() string { return e.Msg }
func (e *GeneralError) ExitCode() int { return ExitError }

func MapErrorToCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if exitCoder, ok := err.(ExitCoder); ok {
		return exitCoder.ExitCode()
	}
	return ExitError
}
