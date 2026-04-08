// Package preset provides storage and management for query presets.
// This file implements escaping, SQL injection detection, and query interpolation.
package preset

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Injection Pattern Detection
// =============================================================================

// InjectionPattern represents a detected SQL injection pattern.
type InjectionPattern struct {
	Name    string // Pattern identifier (e.g., "DROP", "UNION_SELECT")
	Pattern string // Regex pattern that matched
	Value   string // The substring that matched
}

// sqlInjectionPatterns defines patterns used to detect SQL injection attempts.
// These patterns are designed to catch common injection vectors while minimizing
// false positives for legitimate data.
var sqlInjectionPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	// Pattern 1: Semicolon followed by dangerous SQL keywords
	// Catches: ; DROP, ; DELETE, ; INSERT, ; UPDATE, ; TRUNCATE, ; EXEC
	// Returns the specific keyword detected (DROP, DELETE, etc.)
	{
		name:    "DROP",
		pattern: regexp.MustCompile(`(?i);\s*DROP\b`),
	},
	{
		name:    "DELETE",
		pattern: regexp.MustCompile(`(?i);\s*DELETE\b`),
	},
	{
		name:    "INSERT",
		pattern: regexp.MustCompile(`(?i);\s*INSERT\b`),
	},
	{
		name:    "UPDATE",
		pattern: regexp.MustCompile(`(?i);\s*UPDATE\b`),
	},
	{
		name:    "TRUNCATE",
		pattern: regexp.MustCompile(`(?i);\s*TRUNCATE\b`),
	},
	{
		name:    "EXEC",
		pattern: regexp.MustCompile(`(?i);\s*EXEC\b`),
	},
	{
		name:    "ALTER",
		pattern: regexp.MustCompile(`(?i);\s*ALTER\b`),
	},
	{
		name:    "CREATE",
		pattern: regexp.MustCompile(`(?i);\s*CREATE\b`),
	},

	// Pattern 2: OR with always-true conditions
	// Catches: OR 1=1, OR '1'='1', OR 'a'='a', OR 2=2, etc.
	// Must have a quote before OR to avoid false positives in natural language
	{
		name:    "OR_1=1",
		pattern: regexp.MustCompile(`(?i)'\s*OR\s+['"]?\d+['"]?\s*=\s*['"]?\d+['"]?\b`),
	},
	{
		name:    "OR_STRING=STRING",
		pattern: regexp.MustCompile(`(?i)'\s*OR\s+['"][^'"]+['"]\s*=\s*['"][^'"]+['"]`),
	},
	{
		name:    "OR_COMPARISON",
		pattern: regexp.MustCompile(`(?i)'\s*OR\s+\d+\s*[<>=]+\s*\d+\b`),
	},

	// Pattern 3: AND with always-true conditions (less common but still dangerous)
	{
		name:    "AND_1=1",
		pattern: regexp.MustCompile(`(?i)'\s*AND\s+['"]?\d+['"]?\s*=\s*['"]?\d+['"]?\b`),
	},

	// Pattern 4: UNION SELECT (data exfiltration)
	// Catches: ' UNION SELECT, ' UNION ALL SELECT, 1 UNION SELECT, UNION SELECT at start
	// VAL-PARAM-014: Must detect UNION SELECT pattern (case-insensitive) in parameter values.
	// Pattern: UNION must be preceded by quote, digit, or start of string.
	// This avoids false positives for natural language like "labor union" where
	// "union" is preceded by a word character, not a SQL context marker.
	{
		name:    "UNION_SELECT",
		pattern: regexp.MustCompile(`(?i)(?:^|['")]|\d)\s*UNION\s+(ALL\s+)?SELECT\b`),
	},

	// Pattern 5: SQL comment injection
	// Catches: value'-- or value; -- or trailing --
	// Detects -- that could truncate the rest of a query
	// Pattern: quote followed by optional space then --, or semicolon followed by --, or -- at end
	{
		name:    "COMMENT_INJECTION",
		pattern: regexp.MustCompile(`'\s*--|;\s*--|--\s*$`),
	},

	// Pattern 6: Semicolon followed by SELECT (multiple statement injection)
	{
		name:    "SEMICOLON_SELECT",
		pattern: regexp.MustCompile(`(?i);\s*SELECT\b`),
	},
}

// DetectSQLInjection checks a value for SQL injection patterns.
// Returns true if any injection pattern is detected, along with the list of detected patterns.
//
// VAL-PARAM-012: Detects ; DROP patterns
// VAL-PARAM-013: Detects OR 1=1 patterns
// VAL-PARAM-014: Detects UNION SELECT patterns
// VAL-PARAM-015: Detects SQL comment patterns --
// VAL-PARAM-016: Detects multiple injection patterns in single value
func DetectSQLInjection(value string) (bool, []string) {
	var detectedPatterns []string

	for _, ip := range sqlInjectionPatterns {
		if ip.pattern.MatchString(value) {
			detectedPatterns = append(detectedPatterns, ip.name)
		}
	}

	// Deduplicate pattern names
	if len(detectedPatterns) > 0 {
		seen := make(map[string]bool)
		unique := make([]string, 0, len(detectedPatterns))
		for _, p := range detectedPatterns {
			if !seen[p] {
				seen[p] = true
				unique = append(unique, p)
			}
		}
		return true, unique
	}

	return false, nil
}

// =============================================================================
// Escaping Functions
// =============================================================================

// EscapeStringValue escapes a string value for safe use in SQL Server queries.
// Duplicates single quotes: ' → ''
//
// VAL-PARAM-010: String values must have single quotes escaped by duplication
// VAL-PARAM-011: Original data must be preserved after escaping
func EscapeStringValue(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// EscapeListValue converts a CSV string to a SQL IN clause formatted string.
// Input: "a,b,c" → Output: "'a', 'b', 'c'"
//
// Each item is escaped for single quotes before being quoted.
//
// VAL-PARAM-009: List type must convert CSV to SQL IN clause format
func EscapeListValue(value string) string {
	if value == "" {
		return ""
	}

	items := strings.Split(value, ",")
	escaped := make([]string, len(items))

	for i, item := range items {
		// Escape single quotes in each item
		escapedItem := EscapeStringValue(item)
		// Wrap in single quotes
		escaped[i] = fmt.Sprintf("'%s'", escapedItem)
	}

	return strings.Join(escaped, ", ")
}

// =============================================================================
// Query Interpolation
// =============================================================================

// InterpolationError represents an error during query interpolation.
type InterpolationError struct {
	Type        string // "missing_param", "type_mismatch", "sql_injection", "validation"
	ParamName   string // Parameter name (if applicable)
	Message     string // Human-readable error message
	Detected    []string // Detected injection patterns (if applicable)
}

// Error implements the error interface.
func (e *InterpolationError) Error() string {
	switch e.Type {
	case "missing_param":
		return fmt.Sprintf("missing required parameter: %s", e.ParamName)
	case "type_mismatch":
		return fmt.Sprintf("parameter %q: type mismatch - %s", e.ParamName, e.Message)
	case "sql_injection":
		return fmt.Sprintf("SQL injection detected in parameter %q: patterns %v", e.ParamName, e.Detected)
	default:
		return e.Message
	}
}

// InterpolateResult contains the result of query interpolation.
type InterpolateResult struct {
	Query         string            // The interpolated query
	ParamsUsed    map[string]string // Parameters that were used (with defaults applied)
	ParamsMissing []string          // Required parameters that were missing (error case)
}

// InterpolateQuery replaces :placeholder parameters in a query with escaped values.
//
// VAL-PARAM-017: Parameters with defaults use default when not provided
// VAL-PARAM-018: Explicit values override defaults
// VAL-PARAM-019: Defaults work for all types
// VAL-PARAM-020: Optional parameters don't cause errors when not provided
// VAL-PARAM-021: Required parameters cause error if missing; optional don't
// VAL-PARAM-022: Optional parameters with defaults use the default
// VAL-CROSS-003: System rejects injection attempts with clear errors
func InterpolateQuery(query string, params map[string]string, paramDefs []ParamDef) (string, error) {
	// Build a map of param definitions for quick lookup
	defMap := make(map[string]ParamDef)
	for _, def := range paramDefs {
		defMap[def.Name] = def
	}

	// Detect all parameters in the query
	detectedParams := DetectParameters(query)

	// Process each detected parameter
	// We need to replace from end to start to preserve indices
	// But since we're using string replacement, we can do it in any order

	result := query
	usedParams := make(map[string]string)

	for _, paramName := range detectedParams {
		value, hasValue := params[paramName]
		def, hasDef := defMap[paramName]

		// Determine the value to use
		if !hasValue {
			// No value provided
			if hasDef {
				if def.Required {
					// Required parameter missing
					return "", &InterpolationError{
						Type:      "missing_param",
						ParamName: paramName,
						Message:   fmt.Sprintf("required parameter %q was not provided", paramName),
					}
				}
				// Not required - check for default
				if def.Default != "" {
					value = def.Default
					hasValue = true
				}
			} else {
				// No definition for this parameter - treat as optional with no default
				// The placeholder will remain in the query (this is acceptable)
				continue
			}
		}

		// Skip if still no value
		if !hasValue {
			continue
		}

		// Determine type
		paramType := ParamTypeString // default type
		if hasDef {
			paramType = def.Type
		}

		// Validate type
		validatedValue, err := ValidateParamType(value, paramType, paramName)
		if err != nil {
			return "", &InterpolationError{
				Type:      "type_mismatch",
				ParamName: paramName,
				Message:   err.Error(),
			}
		}

		// Check for SQL injection
		detected, patterns := DetectSQLInjection(validatedValue)
		if detected {
			return "", &InterpolationError{
				Type:      "sql_injection",
				ParamName: paramName,
				Detected:  patterns,
				Message:   fmt.Sprintf("potential SQL injection detected: %v", patterns),
			}
		}

		// Format the value based on type
		formattedValue := formatValueForSQL(validatedValue, paramType)

		// Replace the placeholder
		// Use a regex to ensure we only replace :paramName (not part of another identifier)
		placeholderPattern := regexp.MustCompile(`(^|[^:]):` + regexp.QuoteMeta(paramName) + `\b`)
		result = placeholderPattern.ReplaceAllStringFunc(result, func(match string) string {
			// Preserve the character before the colon if any
			if len(match) > 0 && match[0] != ':' {
				return string(match[0]) + formattedValue
			}
			return formattedValue
		})

		usedParams[paramName] = value
	}

	return result, nil
}

// formatValueForSQL formats a validated value for SQL query interpolation.
func formatValueForSQL(value, paramType string) string {
	switch paramType {
	case ParamTypeInt:
		// Integers are not quoted
		return value

	case ParamTypeBool:
		// Booleans become 1 or 0 for SQL Server
		if value == "true" {
			return "1"
		}
		return "0"

	case ParamTypeList:
		// Lists become: 'item1', 'item2', 'item3'
		return EscapeListValue(value)

	case ParamTypeString, ParamTypeDate, ParamTypeDateTime:
		// These types are quoted strings
		return "'" + EscapeStringValue(value) + "'"

	default:
		// Unknown types are treated as strings
		return "'" + EscapeStringValue(value) + "'"
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// IsInterpolationError checks if an error is an InterpolationError.
func IsInterpolationError(err error) bool {
	_, ok := err.(*InterpolationError)
	return ok
}

// GetInterpolationError returns the InterpolationError if the error is one.
func GetInterpolationError(err error) *InterpolationError {
	if ierr, ok := err.(*InterpolationError); ok {
		return ierr
	}
	return nil
}

// ValidatePresetParams validates all parameters for a preset before execution.
// This is a convenience function that combines detection, validation, and injection checking.
func ValidatePresetParams(query string, params map[string]string, paramDefs []ParamDef) error {
	// Build definition map
	defMap := make(map[string]ParamDef)
	for _, def := range paramDefs {
		defMap[def.Name] = def
	}

	// Detect parameters in query
	detectedParams := DetectParameters(query)

	// Check each parameter
	for _, paramName := range detectedParams {
		value, hasValue := params[paramName]
		def, hasDef := defMap[paramName]

		if !hasValue {
			if hasDef && def.Required {
				return &InterpolationError{
					Type:      "missing_param",
					ParamName: paramName,
				}
			}
			if hasDef && def.Default != "" {
				value = def.Default
				hasValue = true
			}
		}

		if !hasValue {
			continue
		}

		// Determine type
		paramType := ParamTypeString
		if hasDef {
			paramType = def.Type
		}

		// Validate type
		_, err := ValidateParamType(value, paramType, paramName)
		if err != nil {
			return &InterpolationError{
				Type:      "type_mismatch",
				ParamName: paramName,
				Message:   err.Error(),
			}
		}

		// Check for SQL injection
		detected, patterns := DetectSQLInjection(value)
		if detected {
			return &InterpolationError{
				Type:      "sql_injection",
				ParamName: paramName,
				Detected:  patterns,
			}
		}
	}

	return nil
}
