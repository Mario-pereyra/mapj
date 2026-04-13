package output

import (
	"time"

	"github.com/Mario-pereyra/mapj/internal/logging"
)

// Envelope is the standard response wrapper for all mapj commands.
// Fields are designed for LLM consumption: minimal noise, actionable structure.
type Envelope struct {
	OK      bool       `json:"ok"`
	Command string     `json:"command"`
	Result  any        `json:"result,omitempty"`
	Error   *ErrDetail `json:"error,omitempty"`

	// Human-mode only fields (omitted in llm mode)
	SchemaVersion string `json:"schemaVersion,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

// ErrDetail holds structured error information.
// hint is a natural-language suggestion for the LLM on how to recover.
type ErrDetail struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Hint         string `json:"hint,omitempty"`    // actionable recovery suggestion for LLM
	Retryable    bool   `json:"retryable"`         // always included per VAL-CLI-035
	RetryAfterMs int    `json:"retryAfterMs,omitempty"`
	TraceId      string `json:"traceId,omitempty"` // correlation ID for error tracing
	Phase        string `json:"phase,omitempty"`   // error phase: validate, auth, execute, cleanup
}

// envelopeMode controls which fields are included in the serialized output.
type envelopeMode int

const (
	// ModeLLM produces compact, minimal output for LLM consumption (default).
	// Omits: schemaVersion, timestamp. Uses JSON compact (no indent).
	ModeLLM envelopeMode = iota

	// ModeHuman produces indented, verbose output for human reading.
	// Includes: schemaVersion, timestamp. Uses JSON pretty (2-space indent).
	ModeHuman
)

// NewEnvelope creates a success envelope.
func NewEnvelope(cmd string, result any) *Envelope {
	return &Envelope{
		OK:      true,
		Command: cmd,
		Result:  result,
	}
}

// NewErrorEnvelope creates an error envelope.
// hint should be an actionable suggestion the LLM can act on (e.g. "run mapj auth login first").
// Automatically includes traceId from the logging context for error correlation.
func NewErrorEnvelope(cmd string, code, message string, retryable bool) *Envelope {
	return &Envelope{
		OK:      false,
		Command: cmd,
		Error:   &ErrDetail{Code: code, Message: message, Retryable: retryable, TraceId: logging.GetTraceID()},
	}
}

// NewErrorEnvelopeWithHint creates an error envelope with a recovery hint for the LLM.
// Automatically includes traceId from the logging context for error correlation.
func NewErrorEnvelopeWithHint(cmd, code, message, hint string, retryable bool) *Envelope {
	return &Envelope{
		OK:      false,
		Command: cmd,
		Error:   &ErrDetail{Code: code, Message: message, Hint: hint, Retryable: retryable, TraceId: logging.GetTraceID()},
	}
}

// NewErrorEnvelopeWithPhase creates an error envelope with phase information.
// Automatically includes traceId from the logging context for error correlation.
func NewErrorEnvelopeWithPhase(cmd, code, message, phase string, retryable bool) *Envelope {
	return &Envelope{
		OK:      false,
		Command: cmd,
		Error:   &ErrDetail{Code: code, Message: message, Retryable: retryable, TraceId: logging.GetTraceID(), Phase: phase},
	}
}

// NewErrorEnvelopeFull creates a fully populated error envelope with all fields.
func NewErrorEnvelopeFull(cmd, code, message, hint string, retryable bool, traceId, phase string) *Envelope {
	return &Envelope{
		OK:      false,
		Command: cmd,
		Error:   &ErrDetail{Code: code, Message: message, Hint: hint, Retryable: retryable, TraceId: traceId, Phase: phase},
	}
}

// withHumanFields adds verbose fields for human-readable output mode.
func (e *Envelope) withHumanFields() *Envelope {
	e.SchemaVersion = "1.0"
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return e
}