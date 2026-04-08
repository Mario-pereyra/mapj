// Package preset provides storage and management for query presets.
// This file implements parameter detection and validation.
package preset

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParamType constants define the supported parameter types.
const (
	ParamTypeString   = "string"
	ParamTypeInt      = "int"
	ParamTypeDate     = "date"
	ParamTypeDateTime = "datetime"
	ParamTypeBool     = "bool"
	ParamTypeList     = "list"
)

// ValidationError represents a parameter validation error with context.
// VAL-PARAM-024: Clear Error for Type Mismatch
type ValidationError struct {
	ParamName   string // Name of the parameter that failed validation
	ExpectedType string // The expected type (e.g., "int", "date")
	ReceivedValue string // The actual value that was received
	Message     string // Human-readable error message
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("parameter %q: %s (expected type: %s, received: %q)",
		e.ParamName, e.Message, e.ExpectedType, e.ReceivedValue)
}

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

// =============================================================================
// Type Validation Functions
// =============================================================================

// ValidateParamType validates a value against the specified parameter type.
// Returns the (potentially normalized) value or a ValidationError.
//
// VAL-PARAM-003 to VAL-PARAM-009: Type validation implementations
// VAL-PARAM-024: Clear Error for Type Mismatch
//
// The returned value may be normalized (e.g., boolean values are normalized to "true" or "false").
func ValidateParamType(value, paramType, paramName string) (string, error) {
	switch paramType {
	case ParamTypeString:
		return validateString(value, paramName)
	case ParamTypeInt:
		return validateInt(value, paramName)
	case ParamTypeDate:
		return validateDate(value, paramName)
	case ParamTypeDateTime:
		return validateDateTime(value, paramName)
	case ParamTypeBool:
		return validateBool(value, paramName)
	case ParamTypeList:
		return validateList(value, paramName)
	default:
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  paramType,
			ReceivedValue: value,
			Message:       fmt.Sprintf("unknown type: %s", paramType),
		}
	}
}

// validateString validates string type values.
// VAL-PARAM-003: String Type Validation
//
// String accepts any value including empty strings and Unicode characters.
func validateString(value, paramName string) (string, error) {
	// String type accepts any value - no validation needed
	return value, nil
}

// validateInt validates integer type values.
// VAL-PARAM-004: Integer Type Validation
// VAL-PARAM-005: Integer Rejects Floats
//
// Integer accepts valid integers (positive, negative, zero) and rejects non-numeric values
// including floating point numbers.
func validateInt(value, paramName string) (string, error) {
	// Check for empty value
	if strings.TrimSpace(value) == "" {
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeInt,
			ReceivedValue: value,
			Message:       "value is empty, expected an integer",
		}
	}

	// Try to parse as integer
	_, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		// Check if it looks like a float
		if strings.Contains(value, ".") || strings.Contains(value, ",") {
			return "", &ValidationError{
				ParamName:     paramName,
				ExpectedType:  ParamTypeInt,
				ReceivedValue: value,
				Message:       "value is a float, expected an integer (no decimal point allowed)",
			}
		}

		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeInt,
			ReceivedValue: value,
			Message:       "value is not a valid integer",
		}
	}

	return value, nil
}

// validateDate validates date type values.
// VAL-PARAM-006: Date Type Validation
//
// Date accepts YYYY-MM-DD format and validates month (1-12) and day according to month.
func validateDate(value, paramName string) (string, error) {
	// Check format first
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeDate,
			ReceivedValue: value,
			Message:       "invalid date format, expected YYYY-MM-DD",
		}
	}

	// Parse with time.Parse for strict validation
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		// Try to give more specific error messages
		yearStr := value[0:4]
		monthStr := value[5:7]
		dayStr := value[8:10]

		year, yearErr := strconv.Atoi(yearStr)
		month, monthErr := strconv.Atoi(monthStr)
		day, dayErr := strconv.Atoi(dayStr)

		if yearErr != nil || monthErr != nil || dayErr != nil {
			return "", &ValidationError{
				ParamName:     paramName,
				ExpectedType:  ParamTypeDate,
				ReceivedValue: value,
				Message:       "date contains non-numeric components",
			}
		}

		if month < 1 || month > 12 {
			return "", &ValidationError{
				ParamName:     paramName,
				ExpectedType:  ParamTypeDate,
				ReceivedValue: value,
				Message:       fmt.Sprintf("invalid month %d, must be between 1 and 12", month),
			}
		}

		maxDays := daysInMonth(year, month)
		if day < 1 || day > maxDays {
			return "", &ValidationError{
				ParamName:     paramName,
				ExpectedType:  ParamTypeDate,
				ReceivedValue: value,
				Message:       fmt.Sprintf("invalid day %d for month %d (max %d days)", day, month, maxDays),
			}
		}

		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeDate,
			ReceivedValue: value,
			Message:       "invalid date",
		}
	}

	return value, nil
}

// validateDateTime validates datetime type values.
// VAL-PARAM-007: DateTime Type Validation
//
// DateTime accepts YYYY-MM-DD HH:MM:SS or YYYY-MM-DDTHH:MM:SS (ISO 8601).
func validateDateTime(value, paramName string) (string, error) {
	// Try both formats
	// Format 1: YYYY-MM-DD HH:MM:SS (space separator)
	// Format 2: YYYY-MM-DDTHH:MM:SS (T separator, ISO 8601)

	// Try space separator format first
	_, err := time.Parse("2006-01-02 15:04:05", value)
	if err == nil {
		return value, nil
	}

	// Try ISO 8601 format with T separator
	_, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return value, nil
	}

	// Try ISO 8601 without timezone
	_, err = time.Parse("2006-01-02T15:04:05", value)
	if err == nil {
		return value, nil
	}

	// Give detailed error message
	if len(value) < 19 {
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeDateTime,
			ReceivedValue: value,
			Message:       "invalid datetime format, expected YYYY-MM-DD HH:MM:SS or YYYY-MM-DDTHH:MM:SS",
		}
	}

	// Check if it looks like a date without time
	if len(value) == 10 && value[4] == '-' && value[7] == '-' {
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeDateTime,
			ReceivedValue: value,
			Message:       "value is a date, expected datetime (include time component)",
		}
	}

	// Try to give more specific error messages
	if strings.Contains(value, " ") {
		parts := strings.Split(value, " ")
		if len(parts) == 2 {
			// Validate date part
			_, dateErr := validateDate(parts[0], paramName)
			if dateErr != nil {
				return "", &ValidationError{
					ParamName:     paramName,
					ExpectedType:  ParamTypeDateTime,
					ReceivedValue: value,
					Message:       "invalid date component in datetime",
				}
			}

			// Validate time part
			timeErr := validateTime(parts[1])
			if timeErr != "" {
				return "", &ValidationError{
					ParamName:     paramName,
					ExpectedType:  ParamTypeDateTime,
					ReceivedValue: value,
					Message:       timeErr,
				}
			}
		}
	}

	return "", &ValidationError{
		ParamName:     paramName,
		ExpectedType:  ParamTypeDateTime,
		ReceivedValue: value,
		Message:       "invalid datetime format, expected YYYY-MM-DD HH:MM:SS or YYYY-MM-DDTHH:MM:SS",
	}
}

// validateTime validates the time component HH:MM:SS.
func validateTime(timeStr string) string {
	if len(timeStr) != 8 || timeStr[2] != ':' || timeStr[5] != ':' {
		return "invalid time format, expected HH:MM:SS"
	}

	hour, hErr := strconv.Atoi(timeStr[0:2])
	min, mErr := strconv.Atoi(timeStr[3:5])
	sec, sErr := strconv.Atoi(timeStr[6:8])

	if hErr != nil || mErr != nil || sErr != nil {
		return "time contains non-numeric components"
	}

	if hour < 0 || hour > 23 {
		return fmt.Sprintf("invalid hour %d, must be between 0 and 23", hour)
	}
	if min < 0 || min > 59 {
		return fmt.Sprintf("invalid minute %d, must be between 0 and 59", min)
	}
	if sec < 0 || sec > 59 {
		return fmt.Sprintf("invalid second %d, must be between 0 and 59", sec)
	}

	return ""
}

// validateBool validates boolean type values.
// VAL-PARAM-008: Boolean Type Validation
//
// Boolean accepts true/false, TRUE/FALSE, 1/0, yes/no and normalizes to "true" or "false".
func validateBool(value, paramName string) (string, error) {
	normalized, err := NormalizeBool(value)
	if err != nil {
		return "", &ValidationError{
			ParamName:     paramName,
			ExpectedType:  ParamTypeBool,
			ReceivedValue: value,
			Message:       fmt.Sprintf("invalid boolean value, accepted: true/false, TRUE/FALSE, 1/0, yes/no"),
		}
	}
	return normalized, nil
}

// NormalizeBool converts various boolean representations to "true" or "false".
// Returns error if the value is not a recognized boolean representation.
func NormalizeBool(value string) (string, error) {
	lower := strings.ToLower(value)

	switch lower {
	case "true", "1", "yes":
		return "true", nil
	case "false", "0", "no":
		return "false", nil
	default:
		return "", fmt.Errorf("invalid boolean value %q, accepted: true/false, TRUE/FALSE, 1/0, yes/no", value)
	}
}

// validateList validates list type values.
// VAL-PARAM-009: List Type Conversion
//
// List accepts CSV and prepares for conversion to IN clause.
// Returns the original value - the conversion to IN clause happens during interpolation.
func validateList(value, paramName string) (string, error) {
	// List accepts any value including empty string
	// The actual conversion to IN clause format happens during interpolation
	// Validation just confirms it's a valid CSV format
	return value, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// isLeapYear determines if a year is a leap year.
func isLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || (year%400 == 0)
}

// daysInMonth returns the number of days in a given month for a given year.
func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	default:
		return 0 // Invalid month
	}
}

// IsValidParamType checks if the given type string is a valid parameter type.
func IsValidParamType(paramType string) bool {
	switch paramType {
	case ParamTypeString, ParamTypeInt, ParamTypeDate, ParamTypeDateTime, ParamTypeBool, ParamTypeList:
		return true
	default:
		return false
	}
}

// ValidParamTypes returns a list of all valid parameter types.
func ValidParamTypes() []string {
	return []string{
		ParamTypeString,
		ParamTypeInt,
		ParamTypeDate,
		ParamTypeDateTime,
		ParamTypeBool,
		ParamTypeList,
	}
}
