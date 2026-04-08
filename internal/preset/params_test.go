package preset

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectParameters_Basic tests basic parameter detection
// VAL-PARAM-001: Placeholder Detection
func TestDetectParameters_Basic(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "single parameter",
			query:    "SELECT * FROM users WHERE id = :id",
			expected: []string{"id"},
		},
		{
			name:     "multiple parameters",
			query:    "SELECT * FROM users WHERE id = :id AND name = :name",
			expected: []string{"id", "name"},
		},
		{
			name:     "no parameters",
			query:    "SELECT * FROM users",
			expected: []string{},
		},
		{
			name:     "parameter at start",
			query:    ":id IS NOT NULL",
			expected: []string{"id"},
		},
		{
			name:     "parameter at end",
			query:    "SELECT * FROM users WHERE id = :id",
			expected: []string{"id"},
		},
		{
			name:     "parameter in middle of value",
			query:    "SELECT * FROM users WHERE id = :userId AND status = :status",
			expected: []string{"userId", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.expected, params)
		})
	}
}

// TestDetectParameters_Deduplication tests duplicate parameter handling
func TestDetectParameters_Deduplication(t *testing.T) {
	query := "SELECT * FROM users WHERE id = :id OR alternate_id = :id"
	params := DetectParameters(query)

	// Should return single "id" despite appearing twice
	assert.Equal(t, []string{"id"}, params)
}

// TestDetectParameters_InStrings tests detection inside SQL strings
func TestDetectParameters_InStrings(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "parameter inside single-quoted string should be detected",
			query:    "SELECT * FROM users WHERE name = ':name'",
			expected: []string{"name"},
		},
		{
			name:     "parameter inside double-quoted string should be detected",
			query:    `SELECT * FROM users WHERE name = ":name"`,
			expected: []string{"name"},
		},
		{
			name:     "parameter next to string literal",
			query:    "SELECT * FROM users WHERE name = 'prefix' + :suffix",
			expected: []string{"suffix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.expected, params)
		})
	}
}

// TestDetectParameters_EmptyAndWhitespace tests empty/whitespace parameter handling
func TestDetectParameters_EmptyAndWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "empty parameter should be ignored",
			query:    "SELECT * FROM users WHERE id = :",
			expected: []string{},
		},
		{
			name:     "whitespace-only parameter should be ignored",
			query:    "SELECT * FROM users WHERE id = :   ",
			expected: []string{},
		},
		{
			name:     "colon without parameter name at end",
			query:    "SELECT time::timestamp FROM users",
			expected: []string{},
		},
		{
			name:     "valid param after empty placeholder",
			query:    "SELECT :  :validParam",
			expected: []string{"validParam"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.expected, params)
		})
	}
}

// TestDetectParameters_ComplexQueries tests detection in complex SQL
func TestDetectParameters_ComplexQueries(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name: "CTE with multiple parameters",
			query: `WITH filtered AS (
				SELECT * FROM users WHERE department = :dept
			)
			SELECT * FROM filtered WHERE status = :status`,
			expected: []string{"dept", "status"},
		},
		{
			name: "JOIN with parameters",
			query: `SELECT u.name, o.total
				FROM users u
				JOIN orders o ON u.id = o.user_id
				WHERE u.created_at > :startDate AND o.total > :minTotal`,
			expected: []string{"startDate", "minTotal"},
		},
		{
			name: "IN clause with list parameter",
			query: "SELECT * FROM users WHERE id IN (:idList)",
			expected: []string{"idList"},
		},
		{
			name: "subquery with parameters",
			query: `SELECT * FROM users WHERE department_id IN (
				SELECT id FROM departments WHERE region = :region
			) AND active = :active`,
			expected: []string{"region", "active"},
		},
		{
			name: "CASE expression with parameters",
			query: `SELECT
				CASE
					WHEN status = :activeStatus THEN 'Active'
					WHEN status = :inactiveStatus THEN 'Inactive'
				END as status_label
				FROM users`,
			expected: []string{"activeStatus", "inactiveStatus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.expected, params)
		})
	}
}

// TestDetectParameters_OrderPreservation tests that parameters are returned in order of first appearance
func TestDetectParameters_OrderPreservation(t *testing.T) {
	query := "SELECT :z, :a, :m, :a, :z FROM table"
	params := DetectParameters(query)

	// Should preserve order of first appearance, deduplicated
	assert.Equal(t, []string{"z", "a", "m"}, params)
}

// TestValidateParamName_Valid tests valid parameter name validation
// VAL-PARAM-002: Valid Parameter Names Only
func TestValidateParamName_Valid(t *testing.T) {
	tests := []struct {
		name   string
		param  string
		valid  bool
	}{
		{
			name:  "simple name",
			param: "id",
			valid: true,
		},
		{
			name:  "name with underscore",
			param: "user_id",
			valid: true,
		},
		{
			name:  "camelCase",
			param: "userId",
			valid: true,
		},
		{
			name:  "PascalCase",
			param: "UserId",
			valid: true,
		},
		{
			name:  "all caps",
			param: "USER_ID",
			valid: true,
		},
		{
			name:  "starts with underscore",
			param: "_private",
			valid: true,
		},
		{
			name:  "numeric suffix",
			param: "param1",
			valid: true,
		},
		{
			name:  "single letter",
			param: "x",
			valid: true,
		},
		{
			name:  "longer name",
			param: "veryLongParameterName",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParamName(tt.param)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestValidateParamName_Invalid tests invalid parameter name detection
// VAL-PARAM-002: Valid Parameter Names Only
func TestValidateParamName_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		param       string
		errContains string
	}{
		{
			name:        "hyphen in name",
			param:       "user-id",
			errContains: "invalid character",
		},
		{
			name:        "dot in name",
			param:       "user.id",
			errContains: "invalid character",
		},
		{
			name:        "dollar sign",
			param:       "user$id",
			errContains: "invalid character",
		},
		{
			name:        "space in name",
			param:       "user id",
			errContains: "invalid character",
		},
		{
			name:        "at sign",
			param:       "@user",
			errContains: "invalid character",
		},
		{
			name:        "hash sign",
			param:       "user#id",
			errContains: "invalid character",
		},
		{
			name:        "percent sign",
			param:       "user%",
			errContains: "invalid character",
		},
		{
			name:        "asterisk",
			param:       "user*",
			errContains: "invalid character",
		},
		{
			name:        "parentheses",
			param:       "user(id)",
			errContains: "invalid character",
		},
		{
			name:        "brackets",
			param:       "user[id]",
			errContains: "invalid character",
		},
		{
			name:        "braces",
			param:       "user{id}",
			errContains: "invalid character",
		},
		{
			name:        "starts with number",
			param:       "1param",
			errContains: "must start with letter or underscore",
		},
		{
			name:        "empty string",
			param:       "",
			errContains: "cannot be empty",
		},
		{
			name:        "only whitespace",
			param:       "   ",
			errContains: "cannot be empty",
		},
		{
			name:        "exclamation mark",
			param:       "user!",
			errContains: "invalid character",
		},
		{
			name:        "question mark",
			param:       "user?",
			errContains: "invalid character",
		},
		{
			name:        "semicolon",
			param:       "user;id",
			errContains: "invalid character",
		},
		{
			name:        "colon",
			param:       "user:id",
			errContains: "invalid character",
		},
		{
			name:        "comma",
			param:       "user,id",
			errContains: "invalid character",
		},
		{
			name:        "equals",
			param:       "user=id",
			errContains: "invalid character",
		},
		{
			name:        "plus sign",
			param:       "user+id",
			errContains: "invalid character",
		},
		{
			name:        "slash",
			param:       "user/id",
			errContains: "invalid character",
		},
		{
			name:        "backslash",
			param:       `user\id`,
			errContains: "invalid character",
		},
		{
			name:        "pipe",
			param:       "user|id",
			errContains: "invalid character",
		},
		{
			name:        "ampersand",
			param:       "user&id",
			errContains: "invalid character",
		},
		{
			name:        "tilde",
			param:       "user~id",
			errContains: "invalid character",
		},
		{
			name:        "backtick",
			param:       "`user`",
			errContains: "invalid character",
		},
		{
			name:        "quotes",
			param:       `"user"`,
			errContains: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParamName(tt.param)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// TestValidateParamNames_Batch tests validating multiple parameter names at once
func TestValidateParamNames_Batch(t *testing.T) {
	tests := []struct {
		name     string
		params   []string
		hasError bool
		errCount int
	}{
		{
			name:     "all valid",
			params:   []string{"id", "name", "user_id"},
			hasError: false,
		},
		{
			name:     "one invalid",
			params:   []string{"id", "user-id", "name"},
			hasError: true,
			errCount: 1,
		},
		{
			name:     "multiple invalid",
			params:   []string{"id", "user-id", "user.name", "valid"},
			hasError: true,
			errCount: 2,
		},
		{
			name:     "empty list",
			params:   []string{},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateParamNames(tt.params)
			if tt.hasError {
				assert.Len(t, errs, tt.errCount)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

// TestDetectAndValidateParameters tests combined detection and validation
func TestDetectAndValidateParameters(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		params    []string
		errCount  int
	}{
		{
			name:     "all valid parameters",
			query:    "SELECT * FROM users WHERE id = :id AND name = :name",
			params:   []string{"id", "name"},
			errCount: 0,
		},
		{
			name:     "one invalid parameter",
			query:    "SELECT * FROM users WHERE id = :user-id",
			params:   []string{"user-id"},
			errCount: 1,
		},
		{
			name:     "mixed valid and invalid",
			query:    "SELECT * FROM users WHERE id = :user-id AND name = :name AND dept = :dept.code",
			params:   []string{"user-id", "name", "dept.code"},
			errCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.params, params)

			errs := ValidateParamNames(params)
			assert.Len(t, errs, tt.errCount)
		})
	}
}

// =============================================================================
// Type Validation Tests
// =============================================================================

// TestValidateString tests string type validation
// VAL-PARAM-003: String Type Validation
func TestValidateString(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Accept any value including empty and Unicode
		{"empty string", "", false},
		{"simple string", "hello", false},
		{"string with spaces", "hello world", false},
		{"string with numbers", "abc123", false},
		{"string with special chars", "hello!@#$%", false},
		{"unicode characters", "héllo wörld 日本語", false},
		{"emoji", "😀🎉", false},
		{"newlines and tabs", "line1\nline2\ttab", false},
		{"SQL-like content", "SELECT * FROM users", false},
		{"single quote", "O'Brien", false},
		{"double quote", `"quoted"`, false},
		{"backslash", `path\to\file`, false},
		{"very long string", strings.Repeat("a", 1000), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateParamType(tt.value, ParamTypeString, "testParam")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateInt tests integer type validation
// VAL-PARAM-004: Integer Type Validation
// VAL-PARAM-005: Integer Rejects Floats
func TestValidateInt(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string // substring expected in error message
	}{
		// Valid integers
		{"zero", "0", false, ""},
		{"positive", "123", false, ""},
		{"negative", "-456", false, ""},
		{"large positive", "999999999999", false, ""},
		{"large negative", "-999999999999", false, ""},
		{"single digit", "5", false, ""},
		{"positive with plus", "+42", false, ""},

		// Invalid - floats
		{"float", "3.14", true, "float"},
		{"negative float", "-2.5", true, "float"},
		{"scientific notation", "1e10", true, "integer"},
		{"comma decimal", "1,5", true, "integer"},

		// Invalid - non-numeric
		{"text", "abc", true, "integer"},
		{"mixed", "123abc", true, "integer"},
		{"empty", "", true, "integer"},
		{"spaces", "  ", true, "integer"},
		{"special chars", "12@3", true, "integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateParamType(tt.value, ParamTypeInt, "intParam")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				// Verify error message includes param name, expected type, received value
				assert.Contains(t, err.Error(), "intParam")
				assert.Contains(t, err.Error(), "int")
				assert.Contains(t, err.Error(), tt.value)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}
		})
	}
}

// TestValidateDate tests date type validation
// VAL-PARAM-006: Date Type Validation
func TestValidateDate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string
	}{
		// Valid dates
		{"standard date", "2024-01-15", false, ""},
		{"min month", "2024-01-01", false, ""},
		{"max month", "2024-12-31", false, ""},
		{"leap year feb 29", "2024-02-29", false, ""},
		{"non-leap year feb 28", "2023-02-28", false, ""},
		{"31 day month", "2024-01-31", false, ""},
		{"30 day month", "2024-04-30", false, ""},
		{"epoch", "1970-01-01", false, ""},
		{"far future", "2099-12-31", false, ""},

		// Invalid format
		{"wrong separator", "2024/01/15", true, "format"},
		{"no separator", "20240115", true, "format"},
		{"european format", "15-01-2024", true, "format"},
		{"month name", "2024-Jan-15", true, "format"},
		{"two digit year", "24-01-15", true, "format"},

		// Invalid month
		{"month zero", "2024-00-15", true, "month"},
		{"month 13", "2024-13-15", true, "month"},
		{"month 99", "2024-99-15", true, "month"},

		// Invalid day
		{"day zero", "2024-01-00", true, "day"},
		{"day 32", "2024-01-32", true, "day"},
		{"day 31 in 30-day month", "2024-04-31", true, "day"},
		{"feb 30", "2024-02-30", true, "day"},
		{"feb 29 non-leap", "2023-02-29", true, "day"},

		// Non-date values
		{"empty", "", true, "format"},
		{"text", "not-a-date", true, "format"},
		{"partial", "2024-01", true, "format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateParamType(tt.value, ParamTypeDate, "dateParam")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				// Verify error includes context
				assert.Contains(t, err.Error(), "dateParam")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}
		})
	}
}

// TestValidateDateTime tests datetime type validation
// VAL-PARAM-007: DateTime Type Validation
func TestValidateDateTime(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string
	}{
		// Valid datetime with space separator
		{"standard datetime", "2024-01-15 14:30:00", false, ""},
		{"midnight", "2024-01-15 00:00:00", false, ""},
		{"end of day", "2024-01-15 23:59:59", false, ""},
		{"noon", "2024-01-15 12:00:00", false, ""},

		// Valid datetime with T separator (ISO 8601)
		{"ISO 8601", "2024-01-15T14:30:00", false, ""},
		{"ISO midnight", "2024-01-15T00:00:00", false, ""},
		{"ISO end of day", "2024-01-15T23:59:59", false, ""},

		// Invalid format
		{"no time", "2024-01-15", true, "format"},
		{"wrong time separator", "2024-01-15T14:30:00", false, ""}, // This is valid (ISO)
		{"time only", "14:30:00", true, "format"},
		{"slash date", "2024/01/15 14:30:00", true, "invalid date"},
		{"no seconds", "2024-01-15 14:30", true, "format"},

		// Invalid time values
		{"hour 25", "2024-01-15 25:00:00", true, "hour"},
		{"minute 60", "2024-01-15 14:60:00", true, "minute"},
		{"second 60", "2024-01-15 14:30:60", true, "second"},

		// Non-datetime values
		{"empty", "", true, "format"},
		{"text", "not-a-datetime", true, "format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateParamType(tt.value, ParamTypeDateTime, "dtParam")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}
		})
	}
}

// TestValidateBool tests boolean type validation
// VAL-PARAM-008: Boolean Type Validation
func TestValidateBool(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantErr  bool
		expected string // normalized value
	}{
		// true variations
		{"lowercase true", "true", false, "true"},
		{"uppercase TRUE", "TRUE", false, "true"},
		{"mixed case True", "True", false, "true"},
		{"mixed case TrUe", "TrUe", false, "true"},
		{"numeric 1", "1", false, "true"},
		{"lowercase yes", "yes", false, "true"},
		{"uppercase YES", "YES", false, "true"},
		{"mixed case Yes", "Yes", false, "true"},

		// false variations
		{"lowercase false", "false", false, "false"},
		{"uppercase FALSE", "FALSE", false, "false"},
		{"mixed case False", "False", false, "false"},
		{"mixed case FaLsE", "FaLsE", false, "false"},
		{"numeric 0", "0", false, "false"},
		{"lowercase no", "no", false, "false"},
		{"uppercase NO", "NO", false, "false"},
		{"mixed case No", "No", false, "false"},

		// Invalid values
		{"empty", "", true, ""},
		{"maybe", "maybe", true, ""},
		{"2", "2", true, ""},
		{"-1", "-1", true, ""},
		{"y", "y", true, ""},
		{"n", "n", true, ""},
		{"t", "t", true, ""},
		{"f", "f", true, ""},
		{"spaces", " true ", true, ""}, // no trimming
		{"TRUE with spaces", "TRUE ", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateParamType(tt.value, ParamTypeBool, "boolParam")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "bool")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestValidateList tests list type validation
// VAL-PARAM-009: List Type Conversion
func TestValidateList(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid lists
		{"single item", "item1", false},
		{"two items", "item1,item2", false},
		{"multiple items", "a,b,c,d,e", false},
		{"numeric items", "1,2,3,4,5", false},
		{"mixed items", "a1,b2,c3", false},
		{"items with spaces", "item 1,item 2", false},
		{"empty item allowed", "item1,,item3", false}, // empty item is preserved
		{"single item no comma", "single", false},
		{"quoted items", `'a',"b",c`, false},
		{"unicode items", "日本,中国,韩国", false},

		// Edge cases - lists can be empty string (will result in empty list)
		{"empty string", "", false},
		{"trailing comma", "a,b,c,", false},
		{"leading comma", ",a,b,c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateParamType(tt.value, ParamTypeList, "listParam")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateParamType_InvalidType tests error handling for unknown types
func TestValidateParamType_InvalidType(t *testing.T) {
	_, err := ValidateParamType("value", "unknown_type", "param")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

// =============================================================================
// Type Conversion Tests
// =============================================================================

// TestNormalizeBool tests boolean normalization
func TestNormalizeBool(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"true", "true", false},
		{"TRUE", "true", false},
		{"1", "true", false},
		{"yes", "true", false},
		{"YES", "true", false},
		{"false", "false", false},
		{"FALSE", "false", false},
		{"0", "false", false},
		{"no", "false", false},
		{"NO", "false", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeBool(tt.input)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestIsLeapYear tests leap year detection
func TestIsLeapYear(t *testing.T) {
	tests := []struct {
		year  int
		leap  bool
	}{
		{2000, true},  // divisible by 400
		{1900, false}, // divisible by 100 but not 400
		{2024, true},  // divisible by 4
		{2023, false}, // not divisible by 4
		{2100, false}, // divisible by 100 but not 400
		{2400, true},  // divisible by 400
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.year), func(t *testing.T) {
			assert.Equal(t, tt.leap, isLeapYear(tt.year))
		})
	}
}

// TestDaysInMonth tests days in month calculation
func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		year   int
		month  int
		days   int
	}{
		{2024, 1, 31},  // January
		{2024, 2, 29},  // February leap year
		{2023, 2, 28},  // February non-leap year
		{2024, 3, 31},  // March
		{2024, 4, 30},  // April
		{2024, 5, 31},  // May
		{2024, 6, 30},  // June
		{2024, 7, 31},  // July
		{2024, 8, 31},  // August
		{2024, 9, 30},  // September
		{2024, 10, 31}, // October
		{2024, 11, 30}, // November
		{2024, 12, 31}, // December
		{2000, 2, 29},  // February leap year (divisible by 400)
		{1900, 2, 28},  // February non-leap (divisible by 100 but not 400)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d-%02d", tt.year, tt.month), func(t *testing.T) {
			assert.Equal(t, tt.days, daysInMonth(tt.year, tt.month))
		})
	}
}

// =============================================================================
// Error Message Quality Tests
// =============================================================================

// TestValidationErrorMessage_Quality tests VAL-PARAM-023, VAL-PARAM-024
// Error messages must include param name, expected type, and received value
func TestValidationErrorMessage_Quality(t *testing.T) {
	t.Run("int type mismatch", func(t *testing.T) {
		_, err := ValidateParamType("not-an-int", ParamTypeInt, "myParam")
		require.Error(t, err)
		errMsg := err.Error()

		// Must include param name
		assert.Contains(t, errMsg, "myParam")
		// Must include expected type
		assert.Contains(t, errMsg, "int")
		// Must include received value
		assert.Contains(t, errMsg, "not-an-int")
	})

	t.Run("date format error", func(t *testing.T) {
		_, err := ValidateParamType("invalid-date", ParamTypeDate, "birthDate")
		require.Error(t, err)
		errMsg := err.Error()

		assert.Contains(t, errMsg, "birthDate")
		assert.Contains(t, errMsg, "date")
		assert.Contains(t, errMsg, "invalid-date")
	})

	t.Run("bool invalid value", func(t *testing.T) {
		_, err := ValidateParamType("maybe", ParamTypeBool, "isActive")
		require.Error(t, err)
		errMsg := err.Error()

		assert.Contains(t, errMsg, "isActive")
		assert.Contains(t, errMsg, "bool")
		assert.Contains(t, errMsg, "maybe")
	})
}

// =============================================================================
// Automatic Type Conversion Tests
// =============================================================================

// TestAutoTypeConversion tests VAL-PARAM-025
// Compatible types should be auto-converted
func TestAutoTypeConversion(t *testing.T) {
	t.Run("numeric string to int", func(t *testing.T) {
		// String "123" should be accepted as int
		result, err := ValidateParamType("123", ParamTypeInt, "numParam")
		require.NoError(t, err)
		assert.Equal(t, "123", result)
	})

	t.Run("int string remains string", func(t *testing.T) {
		// String "123" as string type should remain string
		result, err := ValidateParamType("123", ParamTypeString, "strParam")
		require.NoError(t, err)
		assert.Equal(t, "123", result)
	})

	t.Run("ISO date to datetime", func(t *testing.T) {
		// Date format should NOT be accepted as datetime (needs time)
		_, err := ValidateParamType("2024-01-15", ParamTypeDateTime, "dtParam")
		require.Error(t, err)
	})
}

// TestDetectParameters_EdgeCases tests various edge cases
func TestDetectParameters_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "double colon (PostgreSQL cast)",
			query:    "SELECT created_at::date FROM users WHERE id = :id",
			expected: []string{"id"},
		},
		{
			name:     "parameter followed by colon",
			query:    "SELECT :id::int",
			expected: []string{"id"},
		},
		{
			name:     "multiple colons not a parameter",
			query:    "SELECT time::timestamp::varchar FROM users",
			expected: []string{},
		},
		{
			name:     "parameter in comment should still be detected",
			query:    "SELECT * FROM users -- WHERE id = :commentedParam\n WHERE id = :id",
			expected: []string{"commentedParam", "id"},
		},
		{
			name:     "like pattern with colon",
			query:    "SELECT * FROM users WHERE name LIKE '%:pattern%'",
			expected: []string{"pattern"},
		},
		{
			name:     "concatenation with parameter",
			query:    "SELECT 'Prefix:' + :suffix AS result",
			expected: []string{"suffix"},
		},
		{
			name:     "very long parameter name",
			query:    "SELECT * FROM users WHERE id = :veryLongParameterNameThatShouldStillBeDetected",
			expected: []string{"veryLongParameterNameThatShouldStillBeDetected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DetectParameters(tt.query)
			assert.Equal(t, tt.expected, params)
		})
	}
}
