package output

import "time"

type Envelope struct {
	OK            bool        `json:"ok"`
	Command       string      `json:"command"`
	Result        interface{} `json:"result,omitempty"`
	Error         *ErrDetail  `json:"error,omitempty"`
	SchemaVersion string      `json:"schemaVersion"`
	Timestamp     string      `json:"timestamp"`
}

type ErrDetail struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Retryable   bool   `json:"retryable,omitempty"`
	RetryAfterMs int    `json:"retryAfterMs,omitempty"`
}

func NewEnvelope(cmd string, result interface{}) *Envelope {
	return &Envelope{
		OK:            true,
		Command:       cmd,
		Result:        result,
		SchemaVersion: "1.0",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}

func NewErrorEnvelope(cmd string, code, message string, retryable bool) *Envelope {
	return &Envelope{
		OK:            false,
		Command:       cmd,
		Error:         &ErrDetail{Code: code, Message: message, Retryable: retryable},
		SchemaVersion: "1.0",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}