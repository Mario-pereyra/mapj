---
name: rust-worker
description: Rust implementation worker for mapj CLI migration
---

# Rust Worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

All implementation features for the mapj Rust migration, including:
- Project structure setup
- CLI commands with clap
- Output formatters (LLM, TOON, Auto)
- Authentication and credential storage
- TDN search and Confluence export
- Protheus SQL query and connection management
- Preset system with parameter handling

## Required Skills

None - Rust implementation uses standard Rust tooling only.

## Work Procedure

### Step 1: Understand the Feature
- Read the feature description in features.json
- Review the corresponding Go implementation for reference (in sibling directory)
- Identify all validation contract assertions this feature fulfills
- Write failing tests FIRST (TDD approach)

### Step 2: Implement the Feature

For each feature:

1. **Write tests first** (red phase):
   - Create test module or test file
   - Write tests that exercise the expected behavior
   - Run tests to verify they fail (compilation errors are OK at this stage)

2. **Implement to make tests pass** (green phase):
   - Write the implementation
   - Ensure all tests pass
   - Run `cargo check` for type errors
   - Run `cargo clippy` for lint issues

3. **Verify against Go behavior**:
   - Compare output with Go implementation for formatter features
   - Test SQL validation patterns against Go test cases
   - Ensure parameter escaping matches Go behavior

### Step 3: Verification Steps

For each feature, run the verification steps listed in features.json:
- `cargo test <feature_tests>`
- `cargo check`
- `cargo clippy -- -D warnings`
- Manual verification if specified

### Step 4: Document Discoveries

If the Go implementation reveals unclear behavior:
- Add comments explaining the behavior
- Document edge cases found
- Update the validation contract if new assertions are discovered

## Example Handoff

```json
{
  "salientSummary": "Implemented TOONFormatter with tabular array support. All 22 output format assertions covered. cargo test passes 47/47 tests.",
  "whatWasImplemented": "Output envelope system with LLMFormatter (compact JSON), TOONFormatter (tabular YAML-like with CSV rows for uniform arrays, inline for primitives, quoted strings for special chars), and AutoFormatter (auto-detection). --verbose flag adds schemaVersion and timestamp.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      {"command": "cargo test output_formatter_tests", "exitCode": 0, "observation": "47 tests passing"},
      {"command": "cargo check", "exitCode": 0, "observation": "No type errors"},
      {"command": "cargo clippy -- -D warnings", "exitCode": 0, "observation": "No warnings"}
    ],
    "interactiveChecks": []
  },
  "tests": {
    "added": [
      {"file": "src/output/toon_formatter_test.rs", "cases": ["test_tabulates_uniform_object_arrays", "test_inline_primitive_arrays", "test_string_escaping_quotes", "test_string_escaping_colons", "test_string_escaping_newlines", "test_empty_arrays"]},
      {"file": "src/output/envelope_test.rs", "cases": ["test_success_envelope_structure", "test_error_envelope_structure"]}
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- Feature depends on another feature not yet implemented
- Requirements are ambiguous or contradictory with Go behavior
- Go implementation reveals complexity not anticipated
- SQL injection pattern unclear whether it should be blocked
- Connection to real services needed but unavailable
