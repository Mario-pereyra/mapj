# Bug Analysis: `--param-def` Datetime Parsing Conflict

**Date:** 2026-04-08  
**Issue:** VAL-PARAM-019 - `--param-def` datetime parsing fails due to colon delimiter conflict

---

## 1. Problem Statement

When using `--param-def` with a `datetime` type that includes a default value with time component, the parsing fails because the colons in the timestamp conflict with the `:` delimiter used in the param-def format.

### Example

```bash
--param-def "timestamp:datetime:2024-01-01T10:00:00"
```

### Expected Behavior

- `name`: `timestamp`
- `type`: `datetime`
- `default`: `2024-01-01T10:00:00`
- `description`: (empty)

### Actual Behavior

- `name`: `timestamp`
- `type`: `datetime`
- `default`: `2024-01-01T10` (incorrect, truncated)
- `description`: `00` (incorrect, misinterpreted)

---

## 2. Root Cause Analysis

### Location of Bug

**File:** `D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\protheus_preset.go`

### Current Implementation

```go
// parseParamDef parses a parameter definition string.
// Format: name:type[:default][:description]
// VAL-CLI-005: Invalid param definition returns error
func parseParamDef(s string) (preset.ParamDef, error) {
	parts := strings.Split(s, ":")

	// Must have at least name and type
	if len(parts) < 2 {
		return preset.ParamDef{}, fmt.Errorf("missing type component")
	}

	name := strings.TrimSpace(parts[0])
	paramType := strings.TrimSpace(parts[1])

	// Validate name is not empty
	if name == "" {
		return preset.ParamDef{}, fmt.Errorf("parameter name cannot be empty")
	}

	// Validate type
	if !preset.IsValidParamType(paramType) {
		return preset.ParamDef{}, fmt.Errorf("invalid type '%s', valid types: %s", paramType, strings.Join(preset.ValidParamTypes(), ", "))
	}

	// Build the ParamDef
	def := preset.ParamDef{
		Name:     name,
		Type:     paramType,
		Required: true, // Default to required
	}

	// Parse optional components
	if len(parts) >= 3 {
		def.Default = strings.TrimSpace(parts[2])
		if def.Default != "" {
			def.Required = false // Has default, so not required
		}
	}

	if len(parts) >= 4 {
		def.Description = strings.TrimSpace(parts[3])
	}

	return def, nil
}
```

### The Problem

`strings.Split(s, ":")` splits the string on **every** colon character, including the ones in the datetime value:

```
Input:  "timestamp:datetime:2024-01-01T10:00:00"
Split:  ["timestamp", "datetime", "2024-01-01T10", "00", "00"]
         parts[0]    parts[1]   parts[2]         parts[3] parts[4]
```

The default value `2024-01-01T10:00:00` gets split into three parts, and the description field receives garbage data.

### Affected Formats

This bug affects any param-def where the default value contains colons:

| Type | Example Value | Problematic? |
|------|---------------|--------------|
| `string` | `"http://example.com"` | ✅ Yes |
| `datetime` | `"2024-01-01T10:00:00"` | ✅ Yes |
| `datetime` | `"2024-01-01 10:00:00"` | ❌ No (uses space) |
| `date` | `"2024-01-01"` | ❌ No |
| `int` | `"123"` | ❌ No |
| `bool` | `"true"` | ❌ No |

---

## 3. Solution Options

### Option A: Change Delimiter to `|`

**Approach:** Replace `:` with `|` as the delimiter.

```bash
--param-def "timestamp|datetime|2024-01-01T10:00:00|Description here"
```

**Pros:**
- Clean and unambiguous separation
- No conflict with datetime values
- Pipe is rarely used in SQL/default values
- Breaking change is minimal (feature is new)

**Cons:**
- Breaking change for existing users
- Pipe character might conflict with shell pipes (requires quotes)
- Less intuitive than `:` for name:type pairing

**Code Change:**
```go
parts := strings.Split(s, "|")
```

---

### Option B: Use Quoted Strings for Default/Description

**Approach:** Keep `:` as primary delimiter, but allow quoted strings for values that contain colons.

```bash
--param-def 'timestamp:datetime:"2024-01-01T10:00:00":"Description here"'
```

**Pros:**
- Backward compatible with simple cases
- Flexible for any special characters
- Follows common CLI conventions

**Cons:**
- More complex parsing logic required
- Users must remember to quote datetime values
- Shell quoting complexity (single vs double quotes)
- More verbose

**Code Change:**
```go
// Need to implement a smart splitter that respects quotes
func splitParamDef(s string) []string {
	// Custom parsing logic that handles quoted strings
}
```

---

### Option C: Smart Parsing with Limited Split

**Approach:** Split only on the first 2 colons (for name and type), then handle the rest specially.

```bash
--param-def "timestamp:datetime:2024-01-01T10:00:00"
```

**Logic:**
1. Find first `:` → extract name
2. Find second `:` → extract type  
3. Everything after second `:` is the default
4. If description is needed, use a different mechanism

**Pros:**
- Backward compatible
- Works for most cases
- Simple to implement

**Cons:**
- Cannot have description if default contains `:`
- Ambiguous: is `foo:string:bar:baz` → default=`bar`, desc=`baz` OR default=`bar:baz`?
- Inconsistent behavior depending on whether description is present

**Code Change:**
```go
func parseParamDef(s string) (preset.ParamDef, error) {
	// Split only first 2 times for name and type
	parts := strings.SplitN(s, ":", 3)
	
	if len(parts) < 2 {
		return preset.ParamDef{}, fmt.Errorf("missing type component")
	}
	
	name := strings.TrimSpace(parts[0])
	paramType := strings.TrimSpace(parts[1])
	
	// Remaining part is default:description OR just default
	// Problem: we can't distinguish these cases
}
```

---

### Option D: Hybrid - Change Delimiter + Support Both

**Approach:** Support both `|` as primary delimiter and `:` as legacy, with deprecation warning.

```bash
# New format (preferred)
--param-def "timestamp|datetime|2024-01-01T10:00:00|Description"

# Legacy format (deprecated, still works for simple cases)
--param-def "name:string:value:desc"
```

**Pros:**
- Smooth migration path
- Clear preferred format going forward
- Backward compatible during transition

**Cons:**
- Two formats to maintain
- Code complexity for detection
- Confusing for users during transition

---

### Option E: Separate Flags for Each Component

**Approach:** Instead of a single `--param-def` flag, use separate flags for each component.

```bash
--param name=timestamp --param-type datetime --param-default "2024-01-01T10:00:00" --param-desc "Description"
```

Or structured JSON:

```bash
--param-def '{"name":"timestamp","type":"datetime","default":"2024-01-01T10:00:00","description":"..."}'
```

**Pros:**
- No delimiter ambiguity
- Very flexible
- Easy to extend with new fields

**Cons:**
- Much more verbose
- Breaking change from current API
- JSON requires proper escaping in shell
- Harder to use in quick CLI commands

---

## 4. Recommendation

**Recommended: Option A - Change Delimiter to `|`**

### Rationale

1. **The feature is new:** Since this is a recently implemented feature, breaking changes are acceptable.

2. **Clean and unambiguous:** Pipe character is rarely used in SQL values or descriptions, making it ideal for separation.

3. **Simple implementation:** Single line change in parsing function.

4. **Clear documentation:** Easy to document and explain to users.

### Implementation Plan

1. Update `parseParamDef()` to use `|` as delimiter:
   ```go
   parts := strings.Split(s, "|")
   ```

2. Update help text and documentation:
   ```
   Format: name|type[|default][|description]
   ```

3. Update examples in CLI help:
   ```bash
   --param-def "id|int|0|User ID"
   --param-def "timestamp|datetime|2024-01-01T10:00:00"
   ```

4. Add validation tests for datetime and URL defaults.

### Alternative: Option B (Quoted Strings)

If backward compatibility is a priority, Option B with quoted strings is a good alternative. However, it adds parsing complexity and user friction.

---

## 5. Test Cases for Fix

| Input | Expected Name | Expected Type | Expected Default | Expected Description |
|-------|---------------|---------------|------------------|---------------------|
| `foo\|string` | `foo` | `string` | `` | `` |
| `foo\|string\|bar` | `foo` | `string` | `bar` | `` |
| `foo\|string\|bar\|desc` | `foo` | `string` | `bar` | `desc` |
| `ts\|datetime\|2024-01-01T10:00:00` | `ts` | `datetime` | `2024-01-01T10:00:00` | `` |
| `url\|string\|http://example.com` | `url` | `string` | `http://example.com` | `` |
| `ts\|datetime\|2024-01-01T10:00:00\|Created at` | `ts` | `datetime` | `2024-01-01T10:00:00` | `Created at` |

---

## 6. Files to Modify

1. **`internal/cli/protheus_preset.go`** - Update `parseParamDef()` function
2. **`internal/cli/protheus_preset.go`** - Update command help text
3. **`internal/preset/params_test.go`** - Add test cases for datetime/URL defaults (if needed)

---

## 7. Summary

| Aspect | Details |
|--------|---------|
| **Bug Location** | `parseParamDef()` in `protheus_preset.go` |
| **Root Cause** | `strings.Split(s, ":")` splits on all colons, including those in datetime values |
| **Impact** | Cannot use datetime or URL values in `--param-def` defaults |
| **Recommended Fix** | Change delimiter from `:` to `\|` |
| **Effort** | Low (single line change + documentation) |
