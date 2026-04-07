package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Formatter serializes an Envelope to a string.
type Formatter interface {
	Format(*Envelope) string
}

// ─── LLM Formatter (default) ─────────────────────────────────────────────────

// LLMFormatter produces compact, token-efficient JSON for LLM consumption.
// - No indentation (JSON compact)
// - Omits schemaVersion and timestamp (noise for agents)
// - All action-relevant fields included
type LLMFormatter struct{}

func (f LLMFormatter) Format(env *Envelope) string {
	b, err := json.Marshal(env)
	if err != nil {
		return `{"ok":false,"command":"","error":{"code":"SERIALIZATION_ERROR","message":"failed to serialize output"}}`
	}
	return string(b)
}

// ─── Human Formatter ─────────────────────────────────────────────────────────

// HumanFormatter produces indented, verbose JSON for human reading.
// - 2-space indentation
// - Includes schemaVersion and timestamp
type HumanFormatter struct{}

func (f HumanFormatter) Format(env *Envelope) string {
	// Add verbose fields for human mode
	env.withHumanFields()
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return `{"error": "serialization failed"}`
	}
	return string(b)
}

// ─── CSV Formatter (Protheus only) ───────────────────────────────────────────

// CSVFormatter produces raw CSV (no envelope wrapper).
// Only effective when Result is a *CSVPayload.
// For all other results, falls back to LLMFormatter.
type CSVFormatter struct{}

// CSVPayload is the result type that CSVFormatter knows how to serialize.
type CSVPayload struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

func (f CSVFormatter) Format(env *Envelope) string {
	if env.Error != nil {
		return fmt.Sprintf("ERROR [%s]: %s", env.Error.Code, env.Error.Message)
	}
	if payload, ok := env.Result.(*CSVPayload); ok {
		return renderCSV(payload)
	}
	// Fallback to LLM compact
	return LLMFormatter{}.Format(env)
}

// renderCSV produces RFC 4180-compliant CSV.
func renderCSV(p *CSVPayload) string {
	var sb strings.Builder
	sb.WriteString(csvRow(p.Headers))
	for _, row := range p.Rows {
		sb.WriteString(csvRow(row))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// csvRow formats a single row with proper RFC 4180 escaping.
// Only quotes fields that contain commas, double-quotes, newlines, or carriage returns.
func csvRow(fields []string) string {
	escaped := make([]string, len(fields))
	for i, f := range fields {
		needsQuoting := strings.Contains(f, ",") ||
			strings.Contains(f, `"`) ||
			strings.Contains(f, "\n") ||
			strings.Contains(f, "\r")
		if needsQuoting {
			// Escape inner quotes by doubling them, then wrap in quotes
			f = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
		}
		escaped[i] = f
	}
	return strings.Join(escaped, ",") + "\n"
}

// ─── File Writer ─────────────────────────────────────────────────────────────

// WriteToFile writes the formatted output to a file.
// Used by --output-file flag (Protheus query).
// Returns the absolute path written.
func WriteToFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ─── Factory ─────────────────────────────────────────────────────────────────

// NewFormatter returns the correct Formatter for the given format string.
// "llm" (default)  → LLMFormatter   — compact JSON, no metadata noise
// "json" / "human" → HumanFormatter — indented JSON with timestamp
// "csv"            → CSVFormatter   — RFC 4180 CSV
// "table"          → HumanFormatter — alias for json (table was never implemented)
// "toon"           → TOONFormatter  — token-efficient tabular format
func NewFormatter(format string) Formatter {
	switch strings.ToLower(format) {
	case "json", "human", "table":
		return HumanFormatter{}
	case "csv":
		return CSVFormatter{}
	case "toon":
		return TOONFormatter{}
	default: // "llm" or empty
		return LLMFormatter{}
	}
}
