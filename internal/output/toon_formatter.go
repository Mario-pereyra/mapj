package output

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// TOONFormatter produces token-efficient TOON format output.
// TOON (Tabular Object Notation) is designed for ~40% token savings vs JSON.
type TOONFormatter struct {
	Verbose bool // When true, includes schemaVersion and timestamp fields
}

// Format serializes an Envelope to TOON format.
func (f TOONFormatter) Format(env *Envelope) string {
	// Add verbose fields if requested
	if f.Verbose {
		env = env.withHumanFields()
	}

	var sb strings.Builder

	// Write envelope fields
	sb.WriteString(fmt.Sprintf("ok: %t\n", env.OK))
	sb.WriteString(fmt.Sprintf("command: %s\n", f.formatString(env.Command)))

	// Write verbose fields if present
	if env.SchemaVersion != "" {
		sb.WriteString(fmt.Sprintf("schemaVersion: %s\n", env.SchemaVersion))
	}
	if env.Timestamp != "" {
		sb.WriteString(fmt.Sprintf("timestamp: %s\n", env.Timestamp))
	}

	// Write error or result
	if env.Error != nil {
		sb.WriteString("error:\n")
		f.encodeError(&sb, env.Error, 1)
	} else if env.Result != nil {
		// Convert Result to normalized form via JSON round-trip
		normalized := f.normalizeValue(env.Result)
		f.encodeResult(&sb, normalized)
	} else {
		sb.WriteString("result: null")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// normalizeValue converts any Go value to a normalized form (map[string]any, []any, or primitive)
// by marshaling to JSON and unmarshaling back. This ensures structs are converted to maps.
func (f *TOONFormatter) normalizeValue(v any) any {
	if v == nil {
		return nil
	}

	// Fast path for already-normalized types
	switch val := v.(type) {
	case map[string]any:
		// Recursively normalize values in the map
		result := make(map[string]any)
		for k, v := range val {
			result[k] = f.normalizeValue(v)
		}
		return result
	case []any:
		// Recursively normalize elements in the array
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = f.normalizeValue(v)
		}
		return result
	case string, bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64,
		nil:
		return v
	}

	// Slow path: marshal to JSON and unmarshal back
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		// Fallback: return as-is if JSON marshaling fails
		return v
	}

	var normalized any
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return v
	}

	return normalized
}

// encodeError serializes an ErrDetail to TOON format.
func (f *TOONFormatter) encodeError(sb *strings.Builder, err *ErrDetail, indent int) {
	prefix := f.indent(indent)
	sb.WriteString(fmt.Sprintf("%scode: %s\n", prefix, err.Code))
	sb.WriteString(fmt.Sprintf("%smessage: %s\n", prefix, f.formatString(err.Message)))
	if err.Hint != "" {
		sb.WriteString(fmt.Sprintf("%shint: %s\n", prefix, f.formatString(err.Hint)))
	}
	// VAL-CLI-035: Always include retryable field
	sb.WriteString(fmt.Sprintf("%sretryable: %t\n", prefix, err.Retryable))
	if err.RetryAfterMs > 0 {
		sb.WriteString(fmt.Sprintf("%sretryAfterMs: %d\n", prefix, err.RetryAfterMs))
	}
}

// encodeResult serializes the result field based on its type.
func (f *TOONFormatter) encodeResult(sb *strings.Builder, v any) {
	if v == nil {
		sb.WriteString("result: null")
		return
	}

	switch val := v.(type) {
	case []any:
		// Root-level arrays use special format: result[N]: ...
		f.encodeRootArray(sb, val)
	case map[string]any:
		sb.WriteString("result:\n")
		f.encodeObject(sb, val, 0)
	default:
		// Primitives
		sb.WriteString(fmt.Sprintf("result: %s", f.primitiveToString(val)))
	}
}

// encodeRootArray handles arrays at the root result level with "result" prefix.
func (f *TOONFormatter) encodeRootArray(sb *strings.Builder, arr []any) {
	if len(arr) == 0 {
		sb.WriteString("result[0]:\n")
		return
	}

	// Check if all elements are primitives (can use inline format)
	allPrimitives := true
	for _, v := range arr {
		if !f.isPrimitive(v) {
			allPrimitives = false
			break
		}
	}

	if allPrimitives {
		// Inline format: result[N]: v1,v2,v3
		parts := make([]string, len(arr))
		for i, v := range arr {
			parts[i] = f.primitiveToString(v)
		}
		sb.WriteString(fmt.Sprintf("result[%d]: %s\n", len(arr), strings.Join(parts, ",")))
		return
	}

	// Check if all elements are uniform objects (can use tabular format)
	if f.isUniformObjects(arr) {
		f.encodeRootTabularArray(sb, arr)
		return
	}

	// List format with - markers
	sb.WriteString(fmt.Sprintf("result[%d]:\n", len(arr)))
	for _, v := range arr {
		sb.WriteString("- ")
		f.encodeListItem(sb, v, 1)
		sb.WriteString("\n")
	}
}

// encodeRootTabularArray serializes uniform object arrays in tabular CSV-like format (root level).
func (f *TOONFormatter) encodeRootTabularArray(sb *strings.Builder, arr []any) {
	if len(arr) == 0 {
		return
	}

	// Get field names from first object
	firstObj := arr[0].(map[string]any)
	fields := make([]string, 0, len(firstObj))
	for k := range firstObj {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	// Write header: result[N]{field1,field2}:
	sb.WriteString(fmt.Sprintf("result[%d]{%s}:\n", len(arr), strings.Join(fields, ",")))

	// Write rows as CSV
	for _, item := range arr {
		obj := item.(map[string]any)
		values := make([]string, len(fields))
		for i, field := range fields {
			val := obj[field]
			values[i] = f.primitiveToString(val)
		}
		sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(values, ",")))
	}
}

// encodeObject serializes a map to TOON format with indentation.
func (f *TOONFormatter) encodeObject(sb *strings.Builder, m map[string]any, indent int) {
	if len(m) == 0 {
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	prefix := f.indent(indent + 1)
	for _, key := range keys {
		val := m[key]
		f.encodeObjectField(sb, key, val, indent+1, prefix)
	}
}

// encodeObjectField serializes a single object field.
func (f *TOONFormatter) encodeObjectField(sb *strings.Builder, key string, v any, indent int, prefix string) {
	switch val := v.(type) {
	case []any:
		// Arrays in objects use key[N]: format
		f.encodeObjectArray(sb, key, val, indent, prefix)
	case map[string]any:
		sb.WriteString(fmt.Sprintf("%s%s:\n", prefix, key))
		f.encodeObject(sb, val, indent)
	default:
		// Primitives
		sb.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, key, f.primitiveToString(val)))
	}
}

// encodeObjectArray serializes an array within an object.
func (f *TOONFormatter) encodeObjectArray(sb *strings.Builder, key string, arr []any, indent int, prefix string) {
	if len(arr) == 0 {
		sb.WriteString(fmt.Sprintf("%s%s[0]:\n", prefix, key))
		return
	}

	// Check if all elements are primitives (can use inline format)
	allPrimitives := true
	for _, v := range arr {
		if !f.isPrimitive(v) {
			allPrimitives = false
			break
		}
	}

	if allPrimitives {
		// Inline format: key[N]: v1,v2,v3
		parts := make([]string, len(arr))
		for i, v := range arr {
			parts[i] = f.primitiveToString(v)
		}
		sb.WriteString(fmt.Sprintf("%s%s[%d]: %s\n", prefix, key, len(arr), strings.Join(parts, ",")))
		return
	}

	// Check if all elements are uniform objects (can use tabular format)
	if f.isUniformObjects(arr) {
		f.encodeObjectTabularArray(sb, key, arr, indent, prefix)
		return
	}

	// List format with - markers
	sb.WriteString(fmt.Sprintf("%s%s[%d]:\n", prefix, key, len(arr)))
	itemPrefix := f.indent(indent + 1)
	for _, v := range arr {
		sb.WriteString(fmt.Sprintf("%s- ", itemPrefix))
		f.encodeListItem(sb, v, indent+1)
		sb.WriteString("\n")
	}
}

// encodeObjectTabularArray serializes uniform object arrays in tabular format (within objects).
func (f *TOONFormatter) encodeObjectTabularArray(sb *strings.Builder, key string, arr []any, indent int, prefix string) {
	if len(arr) == 0 {
		return
	}

	// Get field names from first object
	firstObj := arr[0].(map[string]any)
	fields := make([]string, 0, len(firstObj))
	for k := range firstObj {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	// Write header: key[N]{field1,field2}:
	sb.WriteString(fmt.Sprintf("%s%s[%d]{%s}:\n", prefix, key, len(arr), strings.Join(fields, ",")))

	// Write rows as CSV with proper indentation
	rowPrefix := f.indent(indent + 1)
	for _, item := range arr {
		obj := item.(map[string]any)
		values := make([]string, len(fields))
		for i, field := range fields {
			val := obj[field]
			values[i] = f.primitiveToString(val)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", rowPrefix, strings.Join(values, ",")))
	}
}

// encodeListItem serializes an item in a list format array.
func (f *TOONFormatter) encodeListItem(sb *strings.Builder, v any, indent int) {
	if v == nil {
		sb.WriteString("null")
		return
	}

	switch val := v.(type) {
	case bool:
		sb.WriteString(fmt.Sprintf("%t", val))
	case float64:
		sb.WriteString(fmt.Sprintf("%g", val))
	case int:
		sb.WriteString(fmt.Sprintf("%d", val))
	case int64:
		sb.WriteString(fmt.Sprintf("%d", val))
	case string:
		sb.WriteString(f.formatString(val))
	case map[string]any:
		// For objects in list, write inline then properties on new lines
		sb.WriteString("\n")
		prefix := f.indent(indent + 1)
		// Sort keys for consistent output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fieldVal := val[key]
			if nested, ok := fieldVal.(map[string]any); ok {
				sb.WriteString(fmt.Sprintf("%s%s:\n", prefix, key))
				f.encodeObject(sb, nested, indent+1)
			} else if arr, ok := fieldVal.([]any); ok {
				f.encodeObjectArray(sb, key, arr, indent+1, prefix)
			} else {
				sb.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, key, f.primitiveToString(fieldVal)))
			}
		}
	default:
		// Handle other numeric types via reflection
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			sb.WriteString(fmt.Sprintf("%d", rv.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			sb.WriteString(fmt.Sprintf("%d", rv.Uint()))
		case reflect.Float32, reflect.Float64:
			sb.WriteString(fmt.Sprintf("%g", rv.Float()))
		default:
			sb.WriteString(f.formatString(fmt.Sprintf("%v", v)))
		}
	}
}

// isPrimitive checks if a value is a primitive type (null, bool, number, string).
func (f *TOONFormatter) isPrimitive(v any) bool {
	if v == nil {
		return true
	}
	switch v.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return true
	}
	// Check reflection for other numeric types
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

// primitiveToString converts a primitive value to its TOON string representation.
func (f *TOONFormatter) primitiveToString(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case bool:
		return fmt.Sprintf("%t", val)
	case string:
		return f.formatString(val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case float32:
		return fmt.Sprintf("%g", val)
	default:
		// Handle other numeric types via reflection
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("%d", rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return fmt.Sprintf("%d", rv.Uint())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprintf("%g", rv.Float())
		default:
			return f.formatString(fmt.Sprintf("%v", v))
		}
	}
}

// formatString formats a string with quoting if necessary.
func (f *TOONFormatter) formatString(s string) string {
	if f.needsQuoting(s) {
		return fmt.Sprintf(`"%s"`, f.escapeString(s))
	}
	return s
}

// needsQuoting determines if a string needs to be quoted.
// Quote strings containing: :, ,, ", \, newlines, tabs, spaces, or starting with -
func (f *TOONFormatter) needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "-") {
		return true
	}
	// Check for whitespace (spaces, tabs, newlines)
	if strings.ContainsAny(s, " \t\n\r") {
		return true
	}
	if strings.ContainsAny(s, ":,\"\\") {
		return true
	}
	return false
}

// escapeString escapes special characters in a string.
// Escape sequences: \n, \t, \", \\
func (f *TOONFormatter) escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// isUniformObjects checks if all elements in an array are objects with the same keys.
func (f *TOONFormatter) isUniformObjects(arr []any) bool {
	if len(arr) == 0 {
		return false
	}

	// Check if first element is an object
	first, ok := arr[0].(map[string]any)
	if !ok {
		return false
	}

	// Get keys from first object
	firstKeys := make(map[string]bool)
	for k := range first {
		firstKeys[k] = true
	}

	// Check all other objects have the same keys
	for i := 1; i < len(arr); i++ {
		obj, ok := arr[i].(map[string]any)
		if !ok {
			return false
		}

		// Check same number of keys
		if len(obj) != len(firstKeys) {
			return false
		}

		// Check all keys match
		for k := range obj {
			if !firstKeys[k] {
				return false
			}
		}
	}

	return true
}

// indent returns the indentation string (2 spaces per level).
func (f *TOONFormatter) indent(n int) string {
	return strings.Repeat("  ", n)
}
