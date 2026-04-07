package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Primitive Encoding Tests ───────────────────────────────────────────────────

func TestTOONFormatter_Primitives(t *testing.T) {
	formatter := TOONFormatter{}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"null", nil, "null"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"integer", 42, "42"},
		{"negative int", -123, "-123"},
		{"float", 3.14, "3.14"},
		{"zero", 0, "0"},
		{"empty string", "", `""`},
		{"simple string", "hello", "hello"},
		{"string with spaces", "hello world", `"hello world"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvelope("test", tt.input)
			output := formatter.Format(env)
			// Extract just the result value from the envelope
			lines := strings.Split(output, "\n")
			var resultLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "result:") {
					resultLine = strings.TrimPrefix(line, "result:")
					resultLine = strings.TrimSpace(resultLine)
					break
				}
			}
			assert.Equal(t, tt.expected, resultLine)
		})
	}
}

func TestTOONFormatter_StringQuoting(t *testing.T) {
	formatter := TOONFormatter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", `""`},
		{"simple", "hello", "hello"},
		{"with colon", "key:value", `"key:value"`},
		{"with comma", "a,b,c", `"a,b,c"`},
		{"with quote", `say "hello"`, `"say \"hello\""`},
		{"with backslash", `path\to\file`, `"path\\to\\file"`},
		{"with newline", "line1\nline2", "\"line1\\nline2\""},
		{"with tab", "col1\tcol2", "\"col1\\tcol2\""},
		{"starts with dash", "-value", `"-value"`},
		{"with spaces", "hello world", `"hello world"`},
		{"mixed special", `say: "hello\world"`, `"say: \"hello\\world\""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvelope("test", tt.input)
			output := formatter.Format(env)
			lines := strings.Split(output, "\n")
			var resultLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "result:") {
					resultLine = strings.TrimPrefix(line, "result:")
					resultLine = strings.TrimSpace(resultLine)
					break
				}
			}
			assert.Equal(t, tt.expected, resultLine)
		})
	}
}

// ─── Object Encoding Tests ──────────────────────────────────────────────────────

func TestTOONFormatter_SimpleObject(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"name":  "Alice",
		"age":   30,
		"admin": true,
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "ok: true")
	assert.Contains(t, output, "command: test")
	assert.Contains(t, output, "result:")
	assert.Contains(t, output, "  name: Alice")
	assert.Contains(t, output, "  age: 30")
	assert.Contains(t, output, "  admin: true")
}

func TestTOONFormatter_NestedObject(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"address": map[string]any{
				"city": "NYC",
				"zip":  "10001",
			},
		},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "result:")
	assert.Contains(t, output, "  user:")
	assert.Contains(t, output, "    name: Alice")
	assert.Contains(t, output, "    address:")
	assert.Contains(t, output, "      city: NYC")
	assert.Contains(t, output, "      zip: 10001")
}

// ─── Array Encoding Tests ───────────────────────────────────────────────────────

func TestTOONFormatter_PrimitiveArray(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{"apple", "banana", "cherry"}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "result[3]: apple,banana,cherry")
}

func TestTOONFormatter_PrimitiveArrayWithQuoting(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{"item:1", "item,2", "normal"}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	// Items with special chars should be quoted in inline format
	assert.Contains(t, output, `result[3]: "item:1","item,2",normal`)
}

func TestTOONFormatter_UniformObjectArray_Tabular(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{
		map[string]any{"id": 1, "name": "Alice", "active": true},
		map[string]any{"id": 2, "name": "Bob", "active": false},
		map[string]any{"id": 3, "name": "Carol", "active": true},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	// Should use tabular format with header (fields sorted alphabetically)
	assert.Contains(t, output, "result[3]{active,id,name}:")
	// Values follow field order: active,id,name
	assert.Contains(t, output, "true,1,Alice")
	assert.Contains(t, output, "false,2,Bob")
	assert.Contains(t, output, "true,3,Carol")
}

func TestTOONFormatter_NonUniformObjectArray_List(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{
		map[string]any{"id": 1, "name": "Alice"},
		map[string]any{"id": 2, "name": "Bob", "extra": "field"},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	// Should use list format with - markers (fields sorted alphabetically within each object)
	assert.Contains(t, output, "result[2]:")
	// Objects in list format have properties on separate lines
	assert.Contains(t, output, "id: 1")
	assert.Contains(t, output, "name: Alice")
	assert.Contains(t, output, "id: 2")
	assert.Contains(t, output, "extra: field")
}

func TestTOONFormatter_MixedArray(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{"string", 42, true}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	// All primitives use inline format
	assert.Contains(t, output, "result[3]: string,42,true")
}

func TestTOONFormatter_EmptyArray(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "result[0]:")
}

// ─── Error Envelope Tests ───────────────────────────────────────────────────────

func TestTOONFormatter_ErrorEnvelope(t *testing.T) {
	formatter := TOONFormatter{}
	env := NewErrorEnvelopeWithHint("test", "AUTH_ERROR", "invalid token", "run mapj auth login", true)
	output := formatter.Format(env)

	assert.Contains(t, output, "ok: false")
	assert.Contains(t, output, "command: test")
	assert.Contains(t, output, "error:")
	assert.Contains(t, output, "  code: AUTH_ERROR")
	assert.Contains(t, output, "  message: \"invalid token\"")
	assert.Contains(t, output, "  hint: \"run mapj auth login\"")
	assert.Contains(t, output, "  retryable: true")
}

func TestTOONFormatter_ErrorWithRetryAfter(t *testing.T) {
	formatter := TOONFormatter{}
	env := NewErrorEnvelope("test", "RATE_LIMIT", "too many requests", true)
	env.Error.RetryAfterMs = 5000
	output := formatter.Format(env)

	assert.Contains(t, output, "  retryable: true")
	assert.Contains(t, output, "  retryAfterMs: 5000")
}

// ─── Complex Result Tests ───────────────────────────────────────────────────────

func TestTOONFormatter_TDNSearchResult(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"results": []any{
			map[string]any{
				"id":         "235312129",
				"type":       "page",
				"title":      "AdvPL",
				"url":        "https://tdn.totvs.com/display/PROT/AdvPL",
				"childCount": 1,
			},
		},
		"count":   1,
		"total":   1,
		"hasNext": true,
	}
	env := NewEnvelope("mapj tdn search", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "ok: true")
	assert.Contains(t, output, "command: \"mapj tdn search\"")
	assert.Contains(t, output, "result:")
	// Fields are sorted alphabetically: childCount, id, title, type, url
	assert.Contains(t, output, "results[1]{childCount,id,title,type,url}:")
	// Values follow field order
	assert.Contains(t, output, "1,235312129,AdvPL,page,\"https://tdn.totvs.com/display/PROT/AdvPL\"")
	assert.Contains(t, output, "count: 1")
	assert.Contains(t, output, "total: 1")
	assert.Contains(t, output, "hasNext: true")
}

func TestTOONFormatter_ProtheusQueryResult(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"columns": []any{"A1_COD", "A1_NOME", "A1_CGC"},
		"rows": []any{
			map[string]any{"A1_COD": "000001", "A1_NOME": "CLIENT A", "A1_CGC": "12345678000195"},
			map[string]any{"A1_COD": "000002", "A1_NOME": "CLIENT B", "A1_CGC": "98765432000196"},
		},
		"rowCount": 2,
	}
	env := NewEnvelope("mapj protheus query", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "ok: true")
	assert.Contains(t, output, "result:")
	assert.Contains(t, output, "columns[3]: A1_COD,A1_NOME,A1_CGC")
	// Fields are sorted alphabetically: A1_CGC, A1_COD, A1_NOME
	assert.Contains(t, output, "rows[2]{A1_CGC,A1_COD,A1_NOME}:")
	// Values follow field order: A1_CGC, A1_COD, A1_NOME
	// CLIENT A and CLIENT B are quoted because they contain spaces
	assert.Contains(t, output, "12345678000195,000001,\"CLIENT A\"")
	assert.Contains(t, output, "98765432000196,000002,\"CLIENT B\"")
	assert.Contains(t, output, "rowCount: 2")
}

// ─── Helper Function Tests ──────────────────────────────────────────────────────

func TestTOONFormatter_needsQuoting(t *testing.T) {
	formatter := TOONFormatter{}

	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"", true},
		{"hello world", true},
		{"key:value", true},
		{"a,b,c", true},
		{`say "hi"`, true},
		{`path\file`, true},
		{"line1\nline2", true},
		{"-value", true},
		{"value-", false},
		{"UPPER", false},
		{"snake_case", false},
		{"camelCase", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.needsQuoting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTOONFormatter_escapeString(t *testing.T) {
	formatter := TOONFormatter{}

	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`say "hello"`, `say \"hello\"`},
		{`path\to\file`, `path\\to\\file`},
		{"line1\nline2", "line1\\nline2"},
		{"col1\tcol2", "col1\\tcol2"},
		{`back\slash`, `back\\slash`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.escapeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTOONFormatter_isUniformObjects(t *testing.T) {
	formatter := TOONFormatter{}

	tests := []struct {
		name     string
		input    []any
		expected bool
	}{
		{
			"uniform objects",
			[]any{
				map[string]any{"a": 1, "b": 2},
				map[string]any{"a": 3, "b": 4},
			},
			true,
		},
		{
			"different keys",
			[]any{
				map[string]any{"a": 1, "b": 2},
				map[string]any{"a": 3, "c": 4},
			},
			false,
		},
		{
			"primitives",
			[]any{"a", "b", "c"},
			false,
		},
		{
			"mixed types",
			[]any{"string", 42},
			false,
		},
		{
			"empty array",
			[]any{},
			false,
		},
		{
			"single object",
			[]any{map[string]any{"a": 1}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.isUniformObjects(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ─── Edge Cases ───────────────────────────────────────────────────────────────────

func TestTOONFormatter_NilResult(t *testing.T) {
	formatter := TOONFormatter{}
	env := NewEnvelope("test", nil)
	output := formatter.Format(env)

	assert.Contains(t, output, "result: null")
}

func TestTOONFormatter_DeeplyNested(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"value": "deep",
				},
			},
		},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "        value: deep") // 8 spaces = 4 levels * 2
}

func TestTOONFormatter_ArrayInObject(t *testing.T) {
	formatter := TOONFormatter{}
	result := map[string]any{
		"tags": []any{"go", "cli", "json"},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	assert.Contains(t, output, "tags[3]: go,cli,json")
}

func TestTOONFormatter_ObjectInArray(t *testing.T) {
	formatter := TOONFormatter{}
	result := []any{
		map[string]any{"name": "item1"},
		map[string]any{"name": "item2"},
	}
	env := NewEnvelope("test", result)
	output := formatter.Format(env)

	// Should be tabular since uniform
	assert.Contains(t, output, "[2]{name}:")
	assert.Contains(t, output, "item1")
	assert.Contains(t, output, "item2")
}
