package preset

import (
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
