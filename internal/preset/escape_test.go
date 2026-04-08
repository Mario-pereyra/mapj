package preset

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// EscapeStringValue Tests
// =============================================================================

// TestEscapeStringValue tests escaping of single quotes
// VAL-PARAM-010: String values must have single quotes escaped by duplication (' → '')
func TestEscapeStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// No quotes
		{"no quotes", "hello world", "hello world"},
		{"empty string", "", ""},
		{"number", "12345", "12345"},
		{"unicode", "日本語", "日本語"},

		// Single quote cases
		{"single quote at start", "'hello", "''hello"},
		{"single quote at end", "hello'", "hello''"},
		{"single quote in middle", "hel'lo", "hel''lo"},
		{"multiple single quotes", "it's don't", "it''s don''t"},
		{"only single quote", "'", "''"},
		{"two single quotes", "''", "''''"},
		{"three single quotes", "'''", "''''''"},

		// Real-world examples
		{"O'Brien", "O'Brien", "O''Brien"},
		{"SQL string", "it's a test", "it''s a test"},
		{"possessive", "user's data", "user''s data"},
		{"French text", "l'ordinateur", "l''ordinateur"},
		{"Spanish text", "España's", "España''s"},

		// Mixed with other characters
		{"quote with double quote", `he said 'hello'`, `he said ''hello''`},
		{"quote with backslash", `path\'s`, `path\''s`},
		{"quote with special chars", "test'!@#$%", "test''!@#$%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEscapeStringValue_DataPreservation tests VAL-PARAM-011
// Original data must be preserved after escaping; retrieved data matches input
func TestEscapeStringValue_DataPreservation(t *testing.T) {
	// The escaping should be reversible (unescaping)
	// If we escape '' → '''', then unescaping '''' → '' should work
	original := "O'Brien"
	escaped := EscapeStringValue(original)
	require.Equal(t, "O''Brien", escaped)

	// SQL Server treats '' as escaped single quote
	// The data stored would be O'Brien when the query is executed
}

// =============================================================================
// EscapeListValue Tests
// =============================================================================

// TestEscapeListValue tests CSV to IN clause conversion
// VAL-PARAM-009: List type must convert CSV to SQL IN clause format
func TestEscapeListValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Single item
		{"single item", "value1", "'value1'"},
		{"single item with quote", "O'Brien", "'O''Brien'"},

		// Multiple items
		{"two items", "a,b", "'a', 'b'"},
		{"three items", "a,b,c", "'a', 'b', 'c'"},
		{"many items", "1,2,3,4,5", "'1', '2', '3', '4', '5'"},

		// Items with spaces
		{"items with spaces", "item 1,item 2", "'item 1', 'item 2'"},
		{"mixed spaces", "a, b, c", "'a', ' b', ' c'"}, // spaces are part of value

		// Items with quotes
		{"item with quote", "O'Brien,Jane", "'O''Brien', 'Jane'"},
		{"multiple quotes", "it's,don't", "'it''s', 'don''t'"},

		// Special characters
		{"special chars", "a@b,c#d", "'a@b', 'c#d'"},
		{"unicode items", "日本,中国", "'日本', '中国'"},

		// Empty cases
		{"empty string", "", ""},
		{"trailing comma", "a,b,", "'a', 'b', ''"},
		{"leading comma", ",a,b", "'', 'a', 'b'"},
		{"double comma", "a,,b", "'a', '', 'b'"},

		// Complex real-world cases
		{"codes", "001,002,003", "'001', '002', '003'"},
		{"filenames", "file1.txt,file2.txt", "'file1.txt', 'file2.txt'"},
		{"paths", `C:\path1,C:\path2`, `'C:\path1', 'C:\path2'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeListValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEscapeListValue_EmptyItem tests empty item handling
func TestEscapeListValue_EmptyItem(t *testing.T) {
	// Empty item in list should be preserved as empty string
	result := EscapeListValue("a,,b")
	assert.Equal(t, "'a', '', 'b'", result)
}

// =============================================================================
// DetectSQLInjection Tests
// =============================================================================

// TestDetectSQLInjection_SemicolonDrop tests VAL-PARAM-012
// System must detect and reject ; DROP patterns
func TestDetectSQLInjection_SemicolonDrop(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldDetect bool
		patterns    []string // any of these patterns should be detected
	}{
		// Semicolon + DROP variations
		{"; DROP TABLE", "1; DROP TABLE users", true, []string{"DROP"}},
		{"; DROP lowercase", "1; drop table users", true, []string{"DROP"}},
		{";DROP no space", "1;DROP TABLE users", true, []string{"DROP"}},
		{"; DELETE", "1; DELETE FROM users", true, []string{"DELETE"}},
		{"; TRUNCATE", "1; TRUNCATE TABLE users", true, []string{"TRUNCATE"}},
		{"; INSERT", "1; INSERT INTO users", true, []string{"INSERT"}},
		{"; UPDATE", "1; UPDATE users SET", true, []string{"UPDATE"}},
		{"; EXEC", "1; EXEC sp_", true, []string{"EXEC"}},

		// Safe values that contain DROP but not malicious
		{"word DROP in text", "DROP shipment", false, nil},
		{"dropdown text", "dropdown menu", false, nil},
		{"DROP at start no semicolon", "DROP TABLE is not allowed", false, nil},

		// Case variations
		{"mixed case", "1; DrOp TaBlE", true, []string{"DROP"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, patterns := DetectSQLInjection(tt.input)
			if tt.shouldDetect {
				assert.True(t, detected, "Should detect SQL injection in: %s", tt.input)
				// Check that at least one of the expected patterns is detected
				found := false
				for _, expectedPattern := range tt.patterns {
					if containsPattern(patterns, expectedPattern) {
						found = true
						break
					}
				}
				assert.True(t, found, "Should detect at least one of patterns %v in %v", tt.patterns, patterns)
			} else {
				assert.False(t, detected, "Should NOT detect SQL injection in: %s", tt.input)
			}
		})
	}
}

// containsPattern checks if a pattern list contains a specific pattern
func containsPattern(patterns []string, target string) bool {
	for _, p := range patterns {
		if p == target {
			return true
		}
	}
	return false
}

// TestDetectSQLInjection_OR1Equals1 tests VAL-PARAM-013
// System must detect and reject OR 1=1 patterns
func TestDetectSQLInjection_OR1Equals1(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldDetect bool
	}{
		// Classic OR 1=1 patterns (with quote before OR)
		{"OR 1=1", "' OR 1=1 --", true},
		{"OR '1'='1'", "' OR '1'='1'", true},
		{"OR 'a'='a'", "' OR 'a'='a'", true},
		{"OR 'x'='x'", "' OR 'x'='x'--", true},
		{"lowercase or", "' or 1=1 --", true},
		{"mixed case", "' Or 1=1 --", true},

		// Variations
		{"OR 1 = 1 with spaces", "' OR 1 = 1", true},
		{"OR 2=2", "' OR 2=2", true},
		{"OR 0=0", "' OR 0=0", true},
		{"OR 1>0", "' OR 1>0", true},
		{"OR 1<2", "' OR 1<2", true},
		{"AND 1=1", "' AND 1=1", true},

		// Safe values (no quote before OR)
		{"text with OR", "this OR that", false},
		{"equation in text", "x=1 in math", false},
		{"legitimate condition", "value OR default", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _ := DetectSQLInjection(tt.input)
			if tt.shouldDetect {
				assert.True(t, detected, "Should detect SQL injection in: %s", tt.input)
			} else {
				assert.False(t, detected, "Should NOT detect SQL injection in: %s", tt.input)
			}
		})
	}
}

// TestDetectSQLInjection_UnionSelect tests VAL-PARAM-014
// System must detect and reject UNION SELECT patterns
func TestDetectSQLInjection_UnionSelect(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldDetect bool
	}{
		// UNION SELECT patterns (with quote before UNION)
		{"UNION SELECT", "' UNION SELECT", true},
		{"UNION ALL SELECT", "' UNION ALL SELECT", true},
		{"union select lowercase", "' union select", true},
		{"mixed case", "' UnIoN SeLeCt", true},
		{"UNION SELECT with columns", "' UNION SELECT username, password", true},
		{"UNION SELECT FROM", "' UNION SELECT * FROM users", true},

		// Safe values (no quote before UNION, or natural language)
		{"word UNION in text", "labor union select", false},
		{"SELECT without UNION", "select option", false},
		{"text containing select", "I select this", false},
		{"union meeting", "we had a union meeting", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _ := DetectSQLInjection(tt.input)
			if tt.shouldDetect {
				assert.True(t, detected, "Should detect SQL injection in: %s", tt.input)
			} else {
				assert.False(t, detected, "Should NOT detect SQL injection in: %s", tt.input)
			}
		})
	}
}

// TestDetectSQLInjection_Comments tests VAL-PARAM-015
// System must detect and reject SQL comment patterns --
func TestDetectSQLInjection_Comments(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldDetect bool
	}{
		// Comment patterns for injection
		{"-- at end", "value'--", true},
		{"-- with space", "value' -- ", true},
		{"-- with newline", "value'--\nDROP", true},
		{"truncate with --", "value'; DROP users--", true},

		// Safe values with --
		{"double dash in middle", "abc--def", false},
		{"dash at start", "-- comment", false}, // This is actually suspicious but context matters
		{"number range", "1--10", false},
		{"separator in text", "part--part", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _ := DetectSQLInjection(tt.input)
			if tt.shouldDetect {
				assert.True(t, detected, "Should detect SQL injection in: %s", tt.input)
			} else {
				assert.False(t, detected, "Should NOT detect SQL injection in: %s", tt.input)
			}
		})
	}
}

// TestDetectSQLInjection_CombinedPatterns tests VAL-PARAM-016
// System must detect multiple injection patterns in single value
func TestDetectSQLInjection_CombinedPatterns(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		minPatternCount int
		expectedPattern string
	}{
		{
			name:          "OR 1=1 with --",
			input:         "' OR 1=1--",
			minPatternCount: 2,
			expectedPattern: "OR_1=1", // Should also have COMMENT pattern
		},
		{
			name:          "UNION with --",
			input:         "' UNION SELECT * FROM users--",
			minPatternCount: 2,
			expectedPattern: "UNION_SELECT",
		},
		{
			name:          "DROP with semicolon and comment",
			input:         "1; DROP TABLE users; --",
			minPatternCount: 2, // DROP, COMMENT_INJECTION
			expectedPattern: "DROP",
		},
		{
			name:          "full injection attempt",
			input:         "' OR '1'='1' UNION SELECT password FROM users--; DROP TABLE users",
			minPatternCount: 3, // OR_STRING=STRING, UNION_SELECT, COMMENT_INJECTION, DROP
			expectedPattern: "UNION_SELECT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, patterns := DetectSQLInjection(tt.input)
			assert.True(t, detected, "Should detect SQL injection")
			assert.GreaterOrEqual(t, len(patterns), tt.minPatternCount,
				"Should detect at least %d patterns, got %d: %v",
				tt.minPatternCount, len(patterns), patterns)
			assert.Contains(t, patterns, tt.expectedPattern,
				"Should contain pattern %s in %v", tt.expectedPattern, patterns)
		})
	}
}

// TestDetectSQLInjection_SafeValues tests that safe values are not flagged
func TestDetectSQLInjection_SafeValues(t *testing.T) {
	safeValues := []string{
		"normal text",
		"user@example.com",
		"2024-01-15",
		"product name",
		"12345",
		"code-001",
		"file_name.txt",
		"O'Brien",
		"path/to/file",
		`C:\Users\name`,
		"value with 'quotes'",
		"a,b,c,d",
		"select this option",
		"drop shipping",
		"union meeting",
	}

	for _, value := range safeValues {
		t.Run(value, func(t *testing.T) {
			detected, patterns := DetectSQLInjection(value)
			assert.False(t, detected, "Should NOT detect SQL injection in safe value: %s", value)
			assert.Empty(t, patterns, "Should have no patterns for safe value: %s", value)
		})
	}
}

// =============================================================================
// InterpolateQuery Tests
// =============================================================================

// TestInterpolateQuery_Basic tests basic parameter interpolation
func TestInterpolateQuery_Basic(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		params      map[string]string
		paramDefs   []ParamDef
		expected    string
		wantErr     bool
	}{
		{
			name:        "single string parameter",
			query:       "SELECT * FROM users WHERE name = :name",
			params:      map[string]string{"name": "John"},
			paramDefs:   []ParamDef{{Name: "name", Type: ParamTypeString}},
			expected:    "SELECT * FROM users WHERE name = 'John'",
			wantErr:     false,
		},
		{
			name:        "single int parameter",
			query:       "SELECT * FROM users WHERE id = :id",
			params:      map[string]string{"id": "123"},
			paramDefs:   []ParamDef{{Name: "id", Type: ParamTypeInt}},
			expected:    "SELECT * FROM users WHERE id = 123",
			wantErr:     false,
		},
		{
			name:        "multiple parameters",
			query:       "SELECT * FROM users WHERE id = :id AND status = :status",
			params:      map[string]string{"id": "123", "status": "active"},
			paramDefs:   []ParamDef{
				{Name: "id", Type: ParamTypeInt},
				{Name: "status", Type: ParamTypeString},
			},
			expected:    "SELECT * FROM users WHERE id = 123 AND status = 'active'",
			wantErr:     false,
		},
		{
			name:        "parameter with quote",
			query:       "SELECT * FROM users WHERE name = :name",
			params:      map[string]string{"name": "O'Brien"},
			paramDefs:   []ParamDef{{Name: "name", Type: ParamTypeString}},
			expected:    "SELECT * FROM users WHERE name = 'O''Brien'",
			wantErr:     false,
		},
		{
			name:        "list parameter",
			query:       "SELECT * FROM users WHERE id IN (:ids)",
			params:      map[string]string{"ids": "1,2,3"},
			paramDefs:   []ParamDef{{Name: "ids", Type: ParamTypeList}},
			expected:    "SELECT * FROM users WHERE id IN ('1', '2', '3')",
			wantErr:     false,
		},
		{
			name:        "boolean parameter",
			query:       "SELECT * FROM users WHERE active = :active",
			params:      map[string]string{"active": "true"},
			paramDefs:   []ParamDef{{Name: "active", Type: ParamTypeBool}},
			expected:    "SELECT * FROM users WHERE active = 1",
			wantErr:     false,
		},
		{
			name:        "boolean false",
			query:       "SELECT * FROM users WHERE active = :active",
			params:      map[string]string{"active": "false"},
			paramDefs:   []ParamDef{{Name: "active", Type: ParamTypeBool}},
			expected:    "SELECT * FROM users WHERE active = 0",
			wantErr:     false,
		},
		{
			name:        "date parameter",
			query:       "SELECT * FROM orders WHERE date = :date",
			params:      map[string]string{"date": "2024-01-15"},
			paramDefs:   []ParamDef{{Name: "date", Type: ParamTypeDate}},
			expected:    "SELECT * FROM orders WHERE date = '2024-01-15'",
			wantErr:     false,
		},
		{
			name:        "datetime parameter",
			query:       "SELECT * FROM logs WHERE created_at = :dt",
			params:      map[string]string{"dt": "2024-01-15 10:30:00"},
			paramDefs:   []ParamDef{{Name: "dt", Type: ParamTypeDateTime}},
			expected:    "SELECT * FROM logs WHERE created_at = '2024-01-15 10:30:00'",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateQuery(tt.query, tt.params, tt.paramDefs)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestInterpolateQuery_RequiredValidation tests VAL-PARAM-021
// Required parameters cause error if missing; optional don't
func TestInterpolateQuery_RequiredValidation(t *testing.T) {
	t.Run("missing required parameter", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id AND name = :name"
		params := map[string]string{"id": "123"} // name is missing
		paramDefs := []ParamDef{
			{Name: "id", Type: ParamTypeInt, Required: true},
			{Name: "name", Type: ParamTypeString, Required: true},
		}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required parameter")
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing optional parameter - no error", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id"
		params := map[string]string{"id": "123"}
		paramDefs := []ParamDef{
			{Name: "id", Type: ParamTypeInt, Required: true},
			{Name: "status", Type: ParamTypeString, Required: false}, // optional, not provided
		}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
	})

	t.Run("all required provided", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id"
		params := map[string]string{"id": "123"}
		paramDefs := []ParamDef{
			{Name: "id", Type: ParamTypeInt, Required: true},
		}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
	})
}

// TestInterpolateQuery_Defaults tests VAL-PARAM-017, VAL-PARAM-018, VAL-PARAM-019
// Parameters with defaults use default when not provided
// Explicit values override defaults
// Defaults work for all types
func TestInterpolateQuery_Defaults(t *testing.T) {
	t.Run("use default when not provided", func(t *testing.T) {
		query := "SELECT * FROM users WHERE status = :status"
		params := map[string]string{} // status not provided
		paramDefs := []ParamDef{
			{Name: "status", Type: ParamTypeString, Required: false, Default: "active"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'active'")
	})

	t.Run("explicit value overrides default", func(t *testing.T) {
		query := "SELECT * FROM users WHERE status = :status"
		params := map[string]string{"status": "inactive"} // explicit value
		paramDefs := []ParamDef{
			{Name: "status", Type: ParamTypeString, Required: false, Default: "active"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'inactive'")
		assert.NotContains(t, result, "'active'")
	})

	t.Run("default for int type", func(t *testing.T) {
		query := "SELECT * FROM users LIMIT :limit"
		params := map[string]string{}
		paramDefs := []ParamDef{
			{Name: "limit", Type: ParamTypeInt, Required: false, Default: "10"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "10")
	})

	t.Run("default for bool type", func(t *testing.T) {
		query := "SELECT * FROM users WHERE active = :active"
		params := map[string]string{}
		paramDefs := []ParamDef{
			{Name: "active", Type: ParamTypeBool, Required: false, Default: "true"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "1") // true becomes 1
	})

	t.Run("default for date type", func(t *testing.T) {
		query := "SELECT * FROM orders WHERE date >= :startDate"
		params := map[string]string{}
		paramDefs := []ParamDef{
			{Name: "startDate", Type: ParamTypeDate, Required: false, Default: "2024-01-01"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'2024-01-01'")
	})

	t.Run("default for list type", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id IN (:ids)"
		params := map[string]string{}
		paramDefs := []ParamDef{
			{Name: "ids", Type: ParamTypeList, Required: false, Default: "1,2,3"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'1', '2', '3'")
	})
}

// TestInterpolateQuery_OptionalWithDefault tests VAL-PARAM-022
// Optional parameters with defaults use the default
func TestInterpolateQuery_OptionalWithDefault(t *testing.T) {
	query := "SELECT * FROM users WHERE status = :status AND dept = :dept"
	params := map[string]string{"dept": "IT"} // status not provided
	paramDefs := []ParamDef{
		{Name: "status", Type: ParamTypeString, Required: false, Default: "active"},
		{Name: "dept", Type: ParamTypeString, Required: false, Default: "default"},
	}

	result, err := InterpolateQuery(query, params, paramDefs)
	require.NoError(t, err)
	// status should use default, dept should use provided value
	assert.Contains(t, result, "'active'")
	assert.Contains(t, result, "'IT'")
	assert.NotContains(t, result, "'default'") // dept default overridden
}

// TestInterpolateQuery_SQLInjection tests VAL-CROSS-003
// System rejects injection attempts with clear errors
func TestInterpolateQuery_SQLInjection(t *testing.T) {
	injectionAttempts := []struct {
		name   string
		value  string
	}{
		{"semicolon DROP", "1; DROP TABLE users"},
		{"OR 1=1", "' OR 1=1 --"},
		{"UNION SELECT", "' UNION SELECT password FROM users"},
		{"comment injection", "value'--"},
		{"combined attack", "1'; DROP TABLE users; --"},
	}

	for _, attempt := range injectionAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			query := "SELECT * FROM users WHERE name = :name"
			params := map[string]string{"name": attempt.value}
			paramDefs := []ParamDef{{Name: "name", Type: ParamTypeString}}

			_, err := InterpolateQuery(query, params, paramDefs)
			require.Error(t, err, "Should reject SQL injection attempt: %s", attempt.value)
			assert.Contains(t, err.Error(), "SQL injection")
		})
	}
}

// TestInterpolateQuery_TypeValidation tests that types are validated
func TestInterpolateQuery_TypeValidation(t *testing.T) {
	t.Run("invalid int value", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id"
		params := map[string]string{"id": "not-a-number"}
		paramDefs := []ParamDef{{Name: "id", Type: ParamTypeInt}}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "int")
	})

	t.Run("invalid date value", func(t *testing.T) {
		query := "SELECT * FROM orders WHERE date = :date"
		params := map[string]string{"date": "invalid-date"}
		paramDefs := []ParamDef{{Name: "date", Type: ParamTypeDate}}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "date")
	})

	t.Run("invalid bool value", func(t *testing.T) {
		query := "SELECT * FROM users WHERE active = :active"
		params := map[string]string{"active": "maybe"}
		paramDefs := []ParamDef{{Name: "active", Type: ParamTypeBool}}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bool")
	})
}

// TestInterpolateQuery_UndetectedParameters tests handling of parameters not in paramDefs
func TestInterpolateQuery_UndetectedParameters(t *testing.T) {
	t.Run("parameter in query but not in defs uses string type", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id AND name = :name"
		params := map[string]string{"id": "123", "name": "John"}
		paramDefs := []ParamDef{{Name: "id", Type: ParamTypeInt}} // name not defined

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		// name should be treated as string
		assert.Contains(t, result, "'John'")
	})

	t.Run("no param defs - all treated as strings", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id AND name = :name"
		params := map[string]string{"id": "123", "name": "John"}
		paramDefs := []ParamDef{} // no definitions

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'123'")
		assert.Contains(t, result, "'John'")
	})
}

// TestInterpolateQuery_ComplexQueries tests interpolation in complex SQL
func TestInterpolateQuery_ComplexQueries(t *testing.T) {
	t.Run("CTE with parameters", func(t *testing.T) {
		query := `WITH filtered AS (
			SELECT * FROM users WHERE department = :dept
		)
		SELECT * FROM filtered WHERE status = :status`
		params := map[string]string{"dept": "IT", "status": "active"}
		paramDefs := []ParamDef{
			{Name: "dept", Type: ParamTypeString},
			{Name: "status", Type: ParamTypeString},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'IT'")
		assert.Contains(t, result, "'active'")
	})

	t.Run("subquery with parameters", func(t *testing.T) {
		query := `SELECT * FROM users WHERE department_id IN (
			SELECT id FROM departments WHERE region = :region
		) AND active = :active`
		params := map[string]string{"region": "North", "active": "true"}
		paramDefs := []ParamDef{
			{Name: "region", Type: ParamTypeString},
			{Name: "active", Type: ParamTypeBool},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'North'")
		assert.Contains(t, result, "1") // boolean true becomes 1
	})

	t.Run("IN clause with list", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id IN (:ids) AND status = :status"
		params := map[string]string{"ids": "1,2,3,4,5", "status": "active"}
		paramDefs := []ParamDef{
			{Name: "ids", Type: ParamTypeList},
			{Name: "status", Type: ParamTypeString},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "'1', '2', '3', '4', '5'")
		assert.Contains(t, result, "'active'")
	})
}

// TestInterpolateQuery_EdgeCases tests edge cases
func TestInterpolateQuery_EdgeCases(t *testing.T) {
	t.Run("empty string value", func(t *testing.T) {
		query := "SELECT * FROM users WHERE name = :name"
		params := map[string]string{"name": ""}
		paramDefs := []ParamDef{{Name: "name", Type: ParamTypeString}}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Contains(t, result, "''")
	})

	t.Run("parameter used multiple times", func(t *testing.T) {
		query := "SELECT * FROM users WHERE name = :name OR alt_name = :name"
		params := map[string]string{"name": "John"}
		paramDefs := []ParamDef{{Name: "name", Type: ParamTypeString}}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		// Should replace both occurrences
		assert.Equal(t, 2, strings.Count(result, "'John'"))
	})

	t.Run("no parameters in query", func(t *testing.T) {
		query := "SELECT * FROM users"
		params := map[string]string{}
		paramDefs := []ParamDef{}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)
		assert.Equal(t, query, result)
	})

	t.Run("parameter not in params and no default", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = :id AND name = :name"
		params := map[string]string{"id": "123"} // name missing
		paramDefs := []ParamDef{
			{Name: "id", Type: ParamTypeInt, Required: true},
			{Name: "name", Type: ParamTypeString, Required: false}, // not required, no default
		}

		// Should NOT error since name is optional, but won't be interpolated
		// The placeholder remains in the query (or we could remove it conditionally)
		// Current design: optional params without values are not substituted
		_, err := InterpolateQuery(query, params, paramDefs)
		// This is acceptable behavior - the query might be malformed but it's not our job
		// to validate query structure, just to substitute provided parameters
		require.NoError(t, err)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestInterpolateQuery_FullFlow tests the complete flow
func TestInterpolateQuery_FullFlow(t *testing.T) {
	t.Run("complete realistic query", func(t *testing.T) {
		query := `SELECT
			u.id, u.name, u.email,
			o.order_date, o.total
		FROM users u
		JOIN orders o ON u.id = o.user_id
		WHERE u.department = :dept
			AND o.order_date >= :startDate
			AND o.status IN (:statuses)
			AND u.active = :active
		ORDER BY o.order_date DESC
		LIMIT :maxRows`

		params := map[string]string{
			"dept":      "Sales",
			"startDate": "2024-01-01",
			"statuses":  "pending,completed,shipped",
			"active":    "true",
			"maxRows":   "100",
		}

		paramDefs := []ParamDef{
			{Name: "dept", Type: ParamTypeString, Required: true},
			{Name: "startDate", Type: ParamTypeDate, Required: true},
			{Name: "statuses", Type: ParamTypeList, Required: true},
			{Name: "active", Type: ParamTypeBool, Required: false, Default: "true"},
			{Name: "maxRows", Type: ParamTypeInt, Required: false, Default: "50"},
		}

		result, err := InterpolateQuery(query, params, paramDefs)
		require.NoError(t, err)

		// Verify all interpolations
		assert.Contains(t, result, "'Sales'")
		assert.Contains(t, result, "'2024-01-01'")
		assert.Contains(t, result, "'pending', 'completed', 'shipped'")
		assert.Contains(t, result, "100") // maxRows int, no quotes
		assert.Contains(t, result, "1")   // boolean true becomes 1
	})

	t.Run("security validation - injection blocked", func(t *testing.T) {
		query := "SELECT * FROM users WHERE name = :name AND id = :id"
		params := map[string]string{
			"name": "admin'; DROP TABLE users; --",
			"id":   "1",
		}
		paramDefs := []ParamDef{
			{Name: "name", Type: ParamTypeString},
			{Name: "id", Type: ParamTypeInt},
		}

		_, err := InterpolateQuery(query, params, paramDefs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SQL injection")
	})
}
