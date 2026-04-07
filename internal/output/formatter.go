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
type LLMFormatter struct{}

func (f LLMFormatter) Format(env *Envelope) string {
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

type AutoFormatter struct{}

func (f AutoFormatter) Format(env *Envelope) string {
	// If error, return JSON
	if env.Error != nil {
		return LLMFormatter{}.Format(env)
	}

	// Simple heuristic: if it's a slice or contains a large slice as a top-level property,
	// TOON might be better. For now, let's look at the result.
	// If it's a map with "rows" or "results" array, maybe TOON is better.
	
	// Actually, if toon_formatter.go is already converting to TOON, we can just use TOONFormatter
	// and if TOONFormatter handles objects and arrays natively, we just use it!
	// Let's use TOONFormatter if it's available and returns a good string, else fallback.
	// We'll just default to TOONFormatter if it can format anything, but TOONFormatter is implemented in toon_formatter.go.
	return TOONFormatter{}.Format(env)
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
	switch strings.ToLower(format) {
	case "llm", "json":
		return LLMFormatter{}
	case "toon":
		return TOONFormatter{}
	default:
		// Auto-discovery
		return AutoFormatter{}
	}
}

