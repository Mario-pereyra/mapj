// Package preset provides storage and management for query presets.
// This file implements parameter detection and validation.
package preset

import (
	"fmt"
	"regexp"
	"strings"
)

// paramRegex matches :paramName placeholders in SQL queries.
// Uses a pattern that ensures the colon is NOT preceded by another colon
// (to avoid matching PostgreSQL :: cast operator).
//
// Pattern explanation:
// - (?:^|[^:]) - non-capturing group: start of string OR any char that is NOT a colon
// - : - literal colon
// - ([a-zA-Z_][a-zA-Z0-9_\-.]*) - capture group starting with letter/underscore,
//   followed by letters, digits, underscores, hyphens, or dots
//
// This captures potentially invalid names (with hyphens, dots) so they can be validated separately.
var paramRegex = regexp.MustCompile(`(?:^|[^:]):([a-zA-Z_][a-zA-Z0-9_\-.]*)`)

// validParamNameRegex validates parameter names.
// Must start with letter or underscore, followed by any number of letters, digits, or underscores.
var validParamNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// invalidCharRegex finds any character that is not alphanumeric or underscore.
var invalidCharRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// DetectParameters extracts all parameter names from a SQL query.
// VAL-PARAM-001: Placeholder Detection
//
// It finds all :paramName placeholders and returns unique parameter names
// in order of their first appearance.
//
// Note: This function detects parameters with potentially invalid characters
// (like hyphens or dots) so they can be validated separately. Use
// ValidateParamName to check if a detected name is valid.
//
// Examples:
//
//	"SELECT * FROM users WHERE id = :id" → ["id"]
//	"SELECT * FROM users WHERE id = :id AND name = :name" → ["id", "name"]
//	"SELECT * FROM users WHERE id = :id OR alt_id = :id" → ["id"] (deduplicated)
//	"SELECT created_at::date FROM users WHERE id = :id" → ["id"] (:: is PostgreSQL cast, not param)
func DetectParameters(query string) []string {
	// Find all matches
	matches := paramRegex.FindAllStringSubmatch(query, -1)

	// Track seen parameters to deduplicate while preserving order
	seen := make(map[string]bool)
	// Initialize as empty slice, not nil, for consistent JSON serialization
	params := []string{}

	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			// Skip empty matches (shouldn't happen with our regex, but be safe)
			if paramName == "" {
				continue
			}
			// Only add if not seen before
			if !seen[paramName] {
				seen[paramName] = true
				params = append(params, paramName)
			}
		}
	}

	return params
}

// ValidateParamName checks if a parameter name is valid.
// VAL-PARAM-002: Valid Parameter Names Only
//
// Valid names:
//   - Start with a letter (a-z, A-Z) or underscore (_)
//   - Contain only letters, digits, and underscores
//
// Invalid names contain special characters like: -, ., $, #, @, etc.
//
// Returns nil if valid, or an error describing why the name is invalid.
func ValidateParamName(name string) error {
	// Check for empty or whitespace-only
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}

	// Check if starts with number
	if len(trimmed) > 0 && (trimmed[0] >= '0' && trimmed[0] <= '9') {
		return fmt.Errorf("parameter name %q must start with letter or underscore, not a digit", name)
	}

	// Check against valid pattern
	if !validParamNameRegex.MatchString(trimmed) {
		// Find the invalid character for better error message
		invalidChars := invalidCharRegex.FindAllString(trimmed, -1)
		if len(invalidChars) > 0 {
			return fmt.Errorf("parameter name %q contains invalid character(s): %q (only letters, digits, and underscores allowed)", name, strings.Join(invalidChars, ", "))
		}
		return fmt.Errorf("parameter name %q is invalid", name)
	}

	return nil
}

// ValidateParamNames validates multiple parameter names at once.
// Returns a slice of errors for all invalid names.
// If all names are valid, returns an empty slice.
func ValidateParamNames(names []string) []error {
	var errs []error
	for _, name := range names {
		if err := ValidateParamName(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// DetectAndValidateParameters combines detection and validation.
// Returns the detected parameters and any validation errors.
// This is a convenience function for the common use case.
func DetectAndValidateParameters(query string) ([]string, []error) {
	params := DetectParameters(query)
	errs := ValidateParamNames(params)
	return params, errs
}

// HasInvalidParamNames checks if any parameter name in the list is invalid.
// Returns true if any name is invalid, false otherwise.
func HasInvalidParamNames(names []string) bool {
	return len(ValidateParamNames(names)) > 0
}
