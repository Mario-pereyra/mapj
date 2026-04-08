package output

import (
	"encoding/json"
	"os"
	"strings"
)

// Formatter serializes an Envelope to a string.
type Formatter interface {
	Format(*Envelope) string
}

// ─── LLM Formatter ───────────────────────────────────────────────────────────

// LLMFormatter produces compact, token-efficient JSON for LLM consumption.
type LLMFormatter struct {
	Verbose bool // When true, includes schemaVersion and timestamp fields
}

func (f LLMFormatter) Format(env *Envelope) string {
	// Add verbose fields if requested
	if f.Verbose {
		env = env.withHumanFields()
	}

	b, err := json.Marshal(env)
	if err != nil {
		return `{"ok":false,"command":"","error":{"code":"SERIALIZATION_ERROR","message":"failed to serialize output"}}`
	}
	return string(b)
}

// ─── TOON Formatter ──────────────────────────────────────────────────────────

// TOONFormatter is a placeholder for the actual TOON serialization.
// If the actual TOON serialization is implemented in toon_formatter.go, we just use it.
// If not, we might need to rely on the existing TOONFormatter struct.

// ─── Auto Formatter ──────────────────────────────────────────────────────────

type AutoFormatter struct {
	Verbose bool // When true, includes schemaVersion and timestamp fields
}

func (f AutoFormatter) Format(env *Envelope) string {
	// If error, return JSON
	if env.Error != nil {
		return LLMFormatter{Verbose: f.Verbose}.Format(env)
	}

	// Simple heuristic: if it's a slice or contains a large slice as a top-level property,
	// TOON might be better. For now, let's look at the result.
	// If it's a map with "rows" or "results" array, maybe TOON is better.
	
	// Actually, if toon_formatter.go is already converting to TOON, we can just use TOONFormatter
	// and if TOONFormatter handles objects and arrays natively, we just use it!
	// Let's use TOONFormatter if it's available and returns a good string, else fallback.
	// We'll just default to TOONFormatter if it can format anything, but TOONFormatter is implemented in toon_formatter.go.
	return TOONFormatter{Verbose: f.Verbose}.Format(env)
}

// ─── File Writer ─────────────────────────────────────────────────────────────

// WriteToFile writes the formatted output to a file.
func WriteToFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ─── Factory ─────────────────────────────────────────────────────────────────

// NewFormatter returns the correct Formatter for the given format string.
// Auto-detects if format is empty.
func NewFormatter(format string) Formatter {
	return NewFormatterWithVerbose(format, false)
}

// NewFormatterWithVerbose returns a Formatter with verbose mode support.
// When verbose is true, the envelope includes additional fields like schemaVersion and timestamp.
func NewFormatterWithVerbose(format string, verbose bool) Formatter {
	switch strings.ToLower(format) {
	case "llm", "json":
		return LLMFormatter{Verbose: verbose}
	case "toon":
		return TOONFormatter{Verbose: verbose}
	default:
		// Auto-discovery
		return AutoFormatter{Verbose: verbose}
	}
}

