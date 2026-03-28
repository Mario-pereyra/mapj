# Mapj CLI - Plan de Mejoras

## TL;DR

> **Quick Summary**: Mejorar la seguridad (encryption key), corregir error handling (exit codes), y limpiar UX del CLI mapj.
>
> **Deliverables**:
> - Key de encriptación desde environment variable con fallback
> - Error handling consistente con exit codes apropiados
> - Flags funcionales (--no-color, validación de --limit)
> - Auth wiring explícito
> - Tests adicionales para nuevas funcionalidades
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves + final verification
> **Critical Path**: T1 (key extraction) → Wave 2 (error handling) → Wave 3 (UX) → Final

---

## Context

### Original Request
Analizar si la CLI mapj es usable. El análisis reveló problemas críticos de seguridad y error handling que deben resolverse.

### User Decisions
- **Priority**: TODO en paralelo (plan comprehensivo)
- **Encryption Key**: Environment variable `MAPJ_ENCRYPTION_KEY` + fallback machine-derived
- **Tests**: AGREGAR tests para nuevas funcionalidades

### Findings Summary

| Category | Issues | Severity |
|----------|--------|----------|
| Security | Key hardcodeada en 11 lugares | CRITICAL |
| Error Handling | Comandos retornan nil, exit codes no usados | HIGH |
| CLI/UX | --no-color no funciona, auth via side-effect | MEDIUM |
| Validation | No validation en --limit, no max-rows | LOW |

---

## Work Objectives

### Core Objective
Hacer la CLI production-ready: segura, con error handling correcto, y UX consistente.

### Concrete Deliverables
1. Encryption key desde `MAPJ_ENCRYPTION_KEY` env var
2. Fallback a machine-derived key (host ID + username hashed)
3. Exit codes apropiados según tipo de error
4. Commands retornan errores correctamente
5. Flags funcionales
6. Tests para nuevas funcionalidades

### Definition of Done
- [ ] `MAPJ_ENCRYPTION_KEY=xxx mapj auth login tdn --token X` funciona
- [ ] Sin env var, usa machine-derived key automáticamente
- [ ] `mapj tdn search "x"` con credenciales inválidas retorna exit 3 (AUTH)
- [ ] `mapj tdn search "x"` sin args retorna exit 2 (USAGE)
- [ ] `go test ./...` pasa con nuevos tests
- [ ] `go build` successful

### Must Have
- Seguridad: Key de encryption no hardcodeada
- Consistencia: Todos los comandos manejan errores igual
- Funcionalidad: Flags funcionan como se espera

### Must NOT Have
- No cambiar estructura de comandos existente (agregar OK, no cambiar flujos)
- No romper backward compatibility con credenciales existentes (migration)
- No agregar dependencies externas nuevas

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: YES (tests-after para nuevas features)
- **Framework**: Go native + testify

### QA Policy
Every task includes agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/`.

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - 5 tasks):
├── T1: Extract encryption key to env var + fallback
├── T2: Create key derivation functions
├── T3: Create error-to-exit-code mapper
├── T4: Add tests for key derivation
└── T5: Add tests for exit codes

Wave 2 (Error Handling - 5 tasks):
├── T6: Fix tdn.go error returns
├── T7: Fix confluence.go error returns
├── T8: Fix protheus.go error returns
├── T9: Update root.go Execute() exit codes
└── T10: Add tests for error handling

Wave 3 (UX Improvements - 5 tasks):
├── T11: Implement --no-color or remove
├── T12: Make auth wiring explicit in root.go
├── T13: Add --limit validation (> 0)
├── T14: Add --max-rows flag to protheus query
└── T15: Fix json.MarshalIndent error in export.go

Wave FINAL (Verification - 4 tasks):
├── T16: Run all tests
├── T17: go build verification
├── T18: Manual CLI verification
└── T19: Review plan compliance
```

### Dependency Matrix

| Task | Blocks | Blocked By |
|------|--------|------------|
| T1 | T2 | - |
| T2 | T4 | T1 |
| T3 | T5, T9 | - |
| T4 | T16 | T2 |
| T5 | T16 | T3 |
| T6 | T10, T16 | T3 |
| T7 | T10, T16 | T3 |
| T8 | T10, T16 | T3 |
| T9 | T16 | T3 |
| T10 | T16 | T6, T7, T8 |
| T11 | T16 | - |
| T12 | T16 | - |
| T13 | T16 | - |
| T14 | T16 | - |
| T15 | T16 | - |
| T16 | T17, T18 | T4, T5, T10, T11, T12, T13, T14, T15 |
| T17 | T18 | T16 |
| T18 | T19 | T17 |
| T19 | - | T18 |

### Agent Dispatch Summary

- **Wave 1**: 5 tasks → `ultrabrain` (crypto/security work)
- **Wave 2**: 5 tasks → `deep` (error handling refactor)
- **Wave 3**: 5 tasks → `quick` (cleanup/improvements)
- **FINAL**: 4 tasks → `unspecified-high` + `oracle`

---

## TODOs

---

- [ ] 1. Extract encryption key to environment variable with fallback

  **What to do**:
  - Modify `internal/auth/store.go`:
    - Add function `GetEncryptionKey() ([]byte, error)` that:
      1. Checks `MAPJ_ENCRYPTION_KEY` env var
      2. If set and 32 bytes, use it
      3. If not set, derive from machine: hash of `hostname + username + homedir`
      4. Hash with SHA256 to get 32 bytes
    - Replace all 11 `store.SetKey("mapj-cred-key-32bytes-padded!!!!")` calls with `store.SetKey(string(key))`
  - Update `NewStore()` to call `GetEncryptionKey()` automatically
  - Keep backward compat: if old credentials exist with old key, still work (migration path)

  **Must NOT do**:
  - Don't remove the hardcoded key entirely (backward compat)
  - Don't change credential file format
  - Don't break existing credentials

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
    - Reason: Security-sensitive crypto work, need careful implementation
  - **Skills**: []
    - No specific skills needed for this task

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4, 5)
  - **Blocks**: Task 2
  - **Blocked By**: None

  **References**:
  - `internal/auth/store.go:57` - SetKey method location
  - `internal/auth/login.go:49,70,95` - Where key is set in login
  - `internal/cli/tdn.go:50` - Where key is set in tdn command
  - `internal/cli/confluence.go:54` - Where key is set in confluence command
  - `internal/cli/protheus.go:51` - Where key is set in protheus command

  **Acceptance Criteria**:
  - [ ] `MAPJ_ENCRYPTION_KEY=test-key-32-bytes-here!!!! mapj auth status` works
  - [ ] Without env var, still works using machine-derived key
  - [ ] No hardcoded key string in source code (grep should find 0 matches)

  **QA Scenarios**:

  \`\`\`
  Scenario: Key from environment variable
    Tool: Bash
    Preconditions: Fresh credentials file, MAPJ_ENCRYPTION_KEY set
    Steps:
      1. export MAPJ_ENCRYPTION_KEY="test-key-32-bytes-for-env!!!!"
      2. mapj auth login tdn --token "test-token"
      3. mapj auth status
    Expected Result: TDN shows ✓ (authenticated)
    Failure Indicators: "AUTH_ERROR" or "failed to decrypt"
    Evidence: .sisyphus/evidence/task-1-env-key.log

  Scenario: Key fallback to machine-derived
    Tool: Bash
    Preconditions: No MAPJ_ENCRYPTION_KEY set
    Steps:
      1. unset MAPJ_ENCRYPTION_KEY
      2. mapj auth login tdn --token "test-token"
      3. mapj auth status
    Expected Result: TDN shows ✓ (authenticated)
    Failure Indicators: "failed to decrypt" or "no such file"
    Evidence: .sisyphus/evidence/task-1-fallback-key.log
  \`\`\`

  **Commit**: YES
  - Message: `feat(auth): derive encryption key from env or machine ID`
  - Files: `internal/auth/store.go`
  - Pre-commit: `go test ./internal/auth/...`

---

- [ ] 2. Create machine-derived key fallback function

  **What to do**:
  - In `internal/auth/store.go`:
    - Add function `deriveMachineKey() []byte`:
      ```go
      func deriveMachineKey() []byte {
          hostname, _ := os.Hostname()
          username := os.Getenv("USER")
          homedir, _ := os.UserHomeDir()
          data := hostname + username + homedir
          hash := sha256.Sum256([]byte(data))
          return hash[:]
      }
      ```
    - This provides consistent key per machine

  **Must NOT do**:
  - Don't change the hash algorithm (SHA256 is fine)
  - Don't make it user-configurable (env var already is)

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
    - Reason: Simple crypto helper, straightforward implementation
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4, 5)
  - **Blocks**: Task 4
  - **Blocked By**: Task 1

  **References**:
  - `internal/auth/store.go` - Add function near SetKey

  **Acceptance Criteria**:
  - [ ] Same machine always generates same key
  - [ ] Different machines generate different keys

  **QA Scenarios**:

  \`\`\`
  Scenario: Same machine generates same key
    Tool: Bash
    Preconditions: Two different shell sessions
    Steps:
      1. Session A: Run key derivation, record result
      2. Session B: Run key derivation, record result
    Expected Result: Both keys are identical
    Failure Indicators: Keys differ
    Evidence: .sisyphus/evidence/task-2-same-machine.log

  Scenario: Key is 32 bytes
    Tool: Bash
    Preconditions: Go REPL available
    Steps:
      1. Import auth package
      2. Call deriveMachineKey()
      3. Check len(result)
    Expected Result: len == 32
    Failure Indicators: len != 32
    Evidence: .sisyphus/evidence/task-2-key-length.log
  \`\`\`

  **Commit**: YES (grouped with T1)
  - Message: `feat(auth): machine-derived key fallback`
  - Files: `internal/auth/store.go`

---

- [ ] 3. Create error-to-exit-code mapper

  **What to do**:
  - In `internal/errors/codes.go`:
    - Add function `MapErrorToCode(err error) int`:
      ```go
      func MapErrorToCode(err error) int {
          if err == nil {
              return ExitSuccess
          }
          errStr := err.Error()
          switch {
          case strings.Contains(errStr, "AUTH_ERROR"), strings.Contains(errStr, "NOT_AUTHENTICATED"):
              return ExitAuth
          case strings.Contains(errStr, "USAGE_ERROR"), strings.Contains(errStr, "INVALID"):
              return ExitUsage
          case strings.Contains(errStr, "RETRY"):
              return ExitRetry
          case strings.Contains(errStr, "CONFLICT"):
              return ExitConflict
          default:
              return ExitError
          }
      }
      ```
  - Also add string constants for error codes in `internal/errors/codes.go`:
    ```go
    const (
        ErrCodeAuth = "AUTH_ERROR"
        ErrCodeNotAuth = "NOT_AUTHENTICATED"
        ErrCodeUsage = "USAGE_ERROR"
        ErrCodeInvalid = "INVALID_URL"
        ErrCodeRetry = "RETRY"
        ErrCodeConflict = "CONFLICT"
    )
    ```

  **Must NOT do**:
  - Don't remove existing constants
  - Don't change ExitSuccess/ExitError values

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Error handling design, affects all commands
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4, 5)
  - **Blocks**: Tasks 5, 9
  - **Blocked By**: None

  **References**:
  - `internal/errors/codes.go` - Where exit codes are defined
  - `internal/output/envelope.go` - Error codes used in envelopes
  - Various RunE functions in `internal/cli/*.go` - Where errors originate

  **Acceptance Criteria**:
  - [ ] AUTH_ERROR maps to ExitAuth (3)
  - [ ] USAGE_ERROR maps to ExitUsage (2)
  - [ ] RETRY maps to ExitRetry (4)
  - [ ] Unknown errors map to ExitError (1)
  - [ ] nil error maps to ExitSuccess (0)

  **QA Scenarios**:

  \`\`\`
  Scenario: Error code mapping
    Tool: Bash (Go test)
    Preconditions: Test file created
    Steps:
      1. Create test with mock errors
      2. Call MapErrorToCode with each
      3. Assert return values
    Expected Result: Correct exit code for each error type
    Failure Indicators: Wrong exit code returned
    Evidence: .sisyphus/evidence/task-3-error-mapping.log
  \`\`\`

  **Commit**: YES (grouped with T5)
  - Message: `feat(errors): add error-to-exit-code mapper`
  - Files: `internal/errors/codes.go`

---

- [ ] 4. Add tests for key derivation

  **What to do**:
  - In `internal/auth/`:
    - Create `key_derivation_test.go`:
      ```go
      func TestGetEncryptionKey(t *testing.T) {
          // Test env var takes precedence
          os.Setenv("MAPJ_ENCRYPTION_KEY", "test-key-32-bytes-for-env!!!!")
          defer os.Unsetenv("MAPJ_ENCRYPTION_KEY")
          
          key, err := GetEncryptionKey()
          assert.NoError(t, err)
          assert.Equal(t, []byte("test-key-32-bytes-for-env!!!!"), key)
      }
      
      func TestDeriveMachineKey(t *testing.T) {
          key := deriveMachineKey()
          assert.Len(t, key, 32)
          
          // Same machine = same key
          key2 := deriveMachineKey()
          assert.Equal(t, key, key2)
      }
      ```

  **Must NOT do**:
  - Don't test with real credentials (use mocks)
  - Don't hardcode specific machine values

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Test writing, straightforward
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 5)
  - **Blocks**: Task 16
  - **Blocked By**: Task 2

  **References**:
  - `internal/auth/auth_store_test.go` - Existing test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -run Key` passes
  - [ ] All existing tests still pass

  **QA Scenarios**:

  \`\`\`
  Scenario: Key derivation tests pass
    Tool: Bash
    Preconditions: Test file created
    Steps:
      1. go test ./internal/auth/... -v -run "Key"
    Expected Result: All tests pass (2 tests)
    Failure Indicators: Test failures
    Evidence: .sisyphus/evidence/task-4-key-tests.log
  \`\`\`

  **Commit**: YES
  - Message: `test(auth): add key derivation tests`
  - Files: `internal/auth/key_derivation_test.go`

---

- [ ] 5. Add tests for exit codes

  **What to do**:
  - In `internal/errors/`:
    - Create `codes_test.go`:
      ```go
      func TestMapErrorToCode(t *testing.T) {
          tests := []struct {
              errMsg    string
              expected  int
          }{
              {"AUTH_ERROR: invalid token", ExitAuth},
              {"NOT_AUTHENTICATED: please login", ExitAuth},
              {"USAGE_ERROR: missing argument", ExitUsage},
              {"INVALID_URL: could not parse", ExitUsage},
              {"RETRY: rate limited", ExitRetry},
              {"CONFLICT: resource exists", ExitConflict},
              {"some other error", ExitError},
          }
          
          for _, tt := range tests {
              err := errors.New(tt.errMsg)
              got := MapErrorToCode(err)
              assert.Equal(t, tt.expected, got, "for error: %s", tt.errMsg)
          }
          
          // nil error
          assert.Equal(t, ExitSuccess, MapErrorToCode(nil))
      }
      ```

  **Must NOT do**:
  - Don't test with real CLI execution (unit tests only)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Test writing, straightforward
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 4)
  - **Blocks**: Task 16
  - **Blocked By**: Task 3

  **References**:
  - `internal/errors/codes_test.go` - Existing test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/errors/...` passes
  - [ ] 7 test cases all pass

  **QA Scenarios**:

  \`\`\`
  Scenario: Exit code mapping tests
    Tool: Bash
    Preconditions: Test file created
    Steps:
      1. go test ./internal/errors/... -v -run MapErrorToCode
    Expected Result: All 7 test cases pass
    Failure Indicators: Test failures with wrong exit code
    Evidence: .sisyphus/evidence/task-5-exit-tests.log
  \`\`\`

  **Commit**: YES (grouped with T3)
  - Message: `test(errors): add exit code mapping tests`
  - Files: `internal/errors/codes_test.go`

---

- [ ] 6. Fix tdn.go error returns

  **What to do**:
  - In `internal/cli/tdn.go`:
    - Change all `return nil` after error envelope printing to `return err`
    - Specifically lines 48, 56, 62, 73, 82:
      ```go
      // BEFORE (line 46-48):
      if err != nil {
          env := output.NewErrorEnvelope(...)
          fmt.Println(formatter.Format(env))
          return nil  // <-- WRONG
      }
      
      // AFTER:
      if err != nil {
          env := output.NewErrorEnvelope(...)
          fmt.Println(formatter.Format(env))
          return err  // <-- CORRECT
      }
      ```
    - Same pattern for all error cases

  **Must NOT do**:
  - Don't change the envelope printing (keep user-visible output)
  - Don't forget to return the error after printing

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Error handling refactor across multiple files
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 7, 8, 9, 10)
  - **Blocks**: Task 10
  - **Blocked By**: Task 3

  **References**:
  - `internal/cli/tdn.go:46-48` - First error handling
  - `internal/cli/tdn.go:53-56` - Auth error
  - `internal/cli/tdn.go:59-63` - Not authenticated
  - `internal/cli/tdn.go:70-74` - Base URL check
  - `internal/cli/tdn.go:80-83` - Search error

  **Acceptance Criteria**:
  - [ ] `mapj tdn search` with no credentials returns exit 3
  - [ ] `mapj tdn search` with invalid args returns exit 2
  - [ ] `mapj tdn search` with API error returns exit 1

  **QA Scenarios**:

  \`\`\`
  Scenario: TDN search without auth returns exit 3
    Tool: Bash
    Preconditions: No credentials configured
    Steps:
      1. mapj auth logout tdn 2>/dev/null || true
      2. mapj tdn search "test"
    Expected Result: Exit code 3 (AUTH_ERROR)
    Failure Indicators: Exit code 0 or 1
    Evidence: .sisyphus/evidence/task-6-tdn-no-auth.log

  Scenario: TDN search with usage error returns exit 2
    Tool: Bash
    Preconditions: Valid credentials configured
    Steps:
      1. mapj tdn search (no args)
    Expected Result: Exit code 2 (USAGE)
    Failure Indicators: Exit code 0 or panic
    Evidence: .sisyphus/evidence/task-6-tdn-usage.log
  \`\`\`

  **Commit**: YES
  - Message: `fix(cli): return errors instead of nil in tdn.go`
  - Files: `internal/cli/tdn.go`

---

- [ ] 7. Fix confluence.go error returns

  **What to do**:
  - In `internal/cli/confluence.go`:
    - Change all `return nil` after error envelope printing to `return err`
    - Lines 50-52, 57-60, 64-66, 71-73, 92-95, 99-102

  **Must NOT do**:
  - Same as T6

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Error handling refactor
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 8, 9, 10)
  - **Blocks**: Task 10
  - **Blocked By**: Task 3

  **References**:
  - `internal/cli/confluence.go` - All RunE error handling

  **Acceptance Criteria**:
  - [ ] `mapj confluence export` without auth returns exit 3
  - [ ] `mapj confluence export` with invalid URL returns exit 2
  - [ ] `mapj confluence export` with API error returns exit 1

  **QA Scenarios**:

  \`\`\`
  Scenario: Confluence export without auth returns exit 3
    Tool: Bash
    Preconditions: No credentials configured
    Steps:
      1. mapj confluence export 12345
    Expected Result: Exit code 3 (AUTH_ERROR)
    Failure Indicators: Exit code 0
    Evidence: .sisyphus/evidence/task-7-confluence-no-auth.log

  Scenario: Confluence export with invalid URL returns exit 2
    Tool: Bash
    Preconditions: Valid credentials
    Steps:
      1. mapj confluence export "not-a-valid-url"
    Expected Result: Exit code 2 (USAGE)
    Failure Indicators: Exit code 0 or 1
    Evidence: .sisyphus/evidence/task-7-confluence-bad-url.log
  \`\`\`

  **Commit**: YES
  - Message: `fix(cli): return errors instead of nil in confluence.go`
  - Files: `internal/cli/confluence.go`

---

- [ ] 8. Fix protheus.go error returns

  **What to do**:
  - In `internal/cli/protheus.go`:
    - Change all `return nil` after error envelope printing to `return err`
    - Lines 46-49, 54-57, 60-63, 76-79, 81-84

  **Must NOT do**:
  - Same as T6

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Error handling refactor
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 9, 10)
  - **Blocks**: Task 10
  - **Blocked By**: Task 3

  **References**:
  - `internal/cli/protheus.go` - All RunE error handling

  **Acceptance Criteria**:
  - [ ] `mapj protheus query` without auth returns exit 3
  - [ ] `mapj protheus query "INSERT..."` returns exit 2 (USAGE)
  - [ ] `mapj protheus query` with connection error returns exit 1

  **QA Scenarios**:

  \`\`\`
  Scenario: Protheus query without auth returns exit 3
    Tool: Bash
    Preconditions: No credentials configured
    Steps:
      1. mapj protheus query "SELECT 1"
    Expected Result: Exit code 3 (AUTH_ERROR)
    Failure Indicators: Exit code 0
    Evidence: .sisyphus/evidence/task-8-protheus-no-auth.log

  Scenario: Protheus query with INSERT returns exit 2
    Tool: Bash
    Preconditions: Valid credentials
    Steps:
      1. mapj protheus query "INSERT INTO table VALUES(1)"
    Expected Result: Exit code 2 (USAGE_ERROR)
    Failure Indicators: Exit code 0 or 1
    Evidence: .sisyphus/evidence/task-8-protheus-insert.log
  \`\`\`

  **Commit**: YES
  - Message: `fix(cli): return errors instead of nil in protheus.go`
  - Files: `internal/cli/protheus.go`

---

- [ ] 9. Update root.go Execute() to use exit codes

  **What to do**:
  - In `internal/cli/root.go`:
    - Modify `Execute()` to use `MapErrorToCode()`:
      ```go
      func Execute() int {
          rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", ...)
          rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, ...)
          
          if err := rootCmd.Execute(); err != nil {
              fmt.Fprintf(os.Stderr, "Error: %v\n", err)
              return errors.MapErrorToCode(err)  // <-- USE MAPPER
          }
          return errors.ExitSuccess
      }
      ```

  **Must NOT do**:
  - Don't remove the stderr print (useful for debugging)
  - Don't forget to import errors package

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Central error handling change
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8, 10)
  - **Blocks**: Task 16
  - **Blocked By**: Task 3

  **References**:
  - `internal/cli/root.go:37-46` - Current Execute function
  - `internal/errors/codes.go` - MapErrorToCode function

  **Acceptance Criteria**:
  - [ ] `mapj tdn search` (no args) returns exit 2
  - [ ] `mapj tdn search` (no auth) returns exit 3
  - [ ] `mapj tdn search` (API error) returns exit 1 or 4

  **QA Scenarios**:

  \`\`\`
  Scenario: Execute returns correct exit codes
    Tool: Bash
    Preconditions: Build completed
    Steps:
      1. mapj tdn search; echo "Exit: $?"
      2. mapj confluence export invalid-url; echo "Exit: $?"
      3. mapj protheus query "bad"; echo "Exit: $?"
    Expected Result: Correct exit codes (2, 2, 2) respectively
    Failure Indicators: Exit code 0 for all
    Evidence: .sisyphus/evidence/task-9-exit-codes.log
  \`\`\`

  **Commit**: YES
  - Message: `fix(cli): wire exit code mapper into Execute()`
  - Files: `internal/cli/root.go`

---

- [ ] 10. Add tests for error handling

  **What to do**:
  - In `internal/cli/`:
    - Create `error_handling_test.go`:
      ```go
      func TestTdnSearchErrorHandling(t *testing.T) {
          // Test that errors are returned, not swallowed
          // Mock auth store to return error
          // Verify command returns error
      }
      ```

  **Must NOT do**:
  - Don't test with real API calls (use mocks)
  - Don't test auth commands (already tested elsewhere)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Test writing
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8, 9)
  - **Blocks**: Task 16
  - **Blocked By**: Tasks 6, 7, 8

  **References**:
  - `internal/auth/auth_store_test.go` - Mock patterns
  - `pkg/confluence/confluence_url_test.go` - Test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/...` passes
  - [ ] Error return tests pass

  **QA Scenarios**:

  \`\`\`
  Scenario: CLI error handling tests pass
    Tool: Bash
    Preconditions: Test file created
    Steps:
      1. go test ./internal/cli/... -v
    Expected Result: All tests pass
    Failure Indicators: Test failures
    Evidence: .sisyphus/evidence/task-10-cli-tests.log
  \`\`\`

  **Commit**: YES
  - Message: `test(cli): add error handling tests`
  - Files: `internal/cli/error_handling_test.go`

---

- [ ] 11. Implement --no-color flag or remove

  **What to do**:
  - Option A (Implement):
    - Modify `internal/output/formatter.go`:
      - Pass `noColor` bool to Formatter
      - In TableFormatter, strip ANSI codes if `noColor=true`
    - Modify `internal/cli/root.go`:
      - Read `--no-color` flag value
      - Pass to `GetFormatter()`
    - Modify all `fmt.Println(formatter.Format(env))` calls to pass `noColor`

  - Option B (Remove - simpler):
    - Remove `--no-color` flag from `root.go`
    - Remove from `Long` description in `root.go`
    - Update SKILL.md

  **Decision needed**: Ask user or choose Option B (simpler)

  **Must NOT do**:
  - If implementing, ensure it actually works

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple flag change
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 12, 13, 14, 15)
  - **Blocks**: Task 16
  - **Blocked By**: None

  **References**:
  - `internal/cli/root.go:39` - Flag definition
  - `internal/output/formatter.go` - Formatter interface

  **Acceptance Criteria**:
  - [ ] Flag either works correctly OR is removed
  - [ ] `mapj --help` shows correct flag description

  **QA Scenarios**:

  \`\`\`
  Scenario: --no-color works if implemented
    Tool: Bash
    Preconditions: Option A chosen
    Steps:
      1. mapj auth status --output table --no-color
      2. Check output has no ANSI codes
    Expected Result: Plain text without color codes
    Failure Indicators: ANSI codes still present
    Evidence: .sisyphus/evidence/task-11-no-color.log
  \`\`\`

  **Commit**: YES
  - Message: `feat(cli): implement or remove --no-color flag`
  - Files: `internal/cli/root.go`, possibly `internal/output/formatter.go`

---

- [ ] 12. Make auth wiring explicit in root.go

  **What to do**:
  - In `internal/cli/root.go`:
    - Add `authCmd` to rootCmd in `init()` or at top:
      ```go
      func init() {
          rootCmd.AddCommand(tdnCmd, confluenceCmd, protheusCmd, authCmd)  // Add authCmd
      }
      ```
    - Remove `auth.AddCommands(rootCmd)` from `internal/cli/auth.go`
    - Import auth package in root.go
    - Move auth command definitions to a place accessible from root.go

  **Alternative (simpler)**:
    - Keep `auth.go` init() but add comment explaining why
    - Or move authCmd creation to `internal/cli/` package directly

  **Must NOT do**:
  - Don't break `auth login`, `auth status`, `auth logout` commands

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Structural cleanup
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 13, 14, 15)
  - **Blocks**: Task 16
  - **Blocked By**: None

  **References**:
  - `internal/cli/root.go:34` - Current AddCommand
  - `internal/cli/auth.go:8` - Current implicit wiring
  - `internal/auth/login.go:117-120` - auth.AddCommands function

  **Acceptance Criteria**:
  - [ ] `mapj auth status` still works
  - [ ] `mapj auth login tdn --token X` still works
  - [ ] No init() side effects in auth.go OR clear comment explaining it

  **QA Scenarios**:

  \`\`\`
  Scenario: Auth commands work after refactor
    Tool: Bash
    Preconditions: Refactor completed
    Steps:
      1. mapj auth status
      2. mapj auth login tdn --token test
      3. mapj auth logout tdn
    Expected Result: All commands work, proper output
    Failure Indicators: "unknown command" errors
    Evidence: .sisyphus/evidence/task-12-auth-wiring.log
  \`\`\`

  **Commit**: YES
  - Message: `refactor(cli): make auth wiring explicit in root.go`
  - Files: `internal/cli/root.go`, `internal/cli/auth.go`

---

- [ ] 13. Add --limit validation (> 0)

  **What to do**:
  - In `internal/cli/tdn.go`:
    - In `tdnSearchRun`, validate `tdnLimit > 0`:
      ```go
      if tdnLimit <= 0 {
          env := output.NewErrorEnvelope(cmd.CommandPath(), "USAGE_ERROR", "--limit must be > 0", false)
          fmt.Println(formatter.Format(env))
          return errors.New("USAGE_ERROR: --limit must be > 0")
      }
      ```
    - Use constants for error codes

  **Must NOT do**:
  - Don't break existing functionality (limit defaults to 10)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple validation addition
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 12, 14, 15)
  - **Blocks**: Task 16
  - **Blocked By**: None

  **References**:
  - `internal/cli/tdn.go:31` - tdnLimit variable
  - `internal/cli/tdn.go:36` - Flag definition

  **Acceptance Criteria**:
  - [ ] `mapj tdn search "x" --limit 5` works (exit 0)
  - [ ] `mapj tdn search "x" --limit -1` returns exit 2

  **QA Scenarios**:

  \`\`\`
  Scenario: Negative limit rejected
    Tool: Bash
    Preconditions: Valid credentials
    Steps:
      1. mapj tdn search "test" --limit -5
    Expected Result: Exit code 2 (USAGE)
    Failure Indicators: Exit code 0 or accepts negative
    Evidence: .sisyphus/evidence/task-13-limit-validation.log
  \`\`\`

  **Commit**: YES
  - Message: `feat(cli): validate --limit > 0 in tdn search`
  - Files: `internal/cli/tdn.go`

---

- [ ] 14. Add --max-rows flag to protheus query

  **What to do**:
  - In `internal/cli/protheus.go`:
    - Add flag: `protheusMaxRows int`
    - Add to init():
      ```go
      protheusQueryCmd.Flags().IntVar(&protheusMaxRows, "max-rows", 10000, "Max rows to return")
      ```
    - In `protheusQueryRun`, after getting results, truncate if exceeds:
      ```go
      if protheusMaxRows > 0 && result.Count > protheusMaxRows {
          result.Rows = result.Rows[:protheusMaxRows]
          result.Count = protheusMaxRows
      }
      ```

  **Must NOT do**:
  - Don't change default behavior significantly (default 10000 is safe)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple flag addition
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 12, 13, 15)
  - **Blocks**: Task 16
  - **Blocked By**: None

  **References**:
  - `internal/cli/protheus.go:37` - Format flag location
  - `pkg/protheus/query.go:122` - QueryResult structure

  **Acceptance Criteria**:
  - [ ] `mapj protheus query "SELECT 1"` works (default max-rows)
  - [ ] `mapj protheus query "SELECT 1" --max-rows 5` limits results
  - [ ] Flag appears in help

  **QA Scenarios**:

  \`\`\`
  Scenario: Max-rows limits results
    Tool: Bash
    Preconditions: Valid Protheus credentials
    Steps:
      1. mapj protheus query "SELECT TOP 100 * FROM some_table" --max-rows 10
    Expected Result: Result shows count <= 10
    Failure Indicators: All 100 rows returned
    Evidence: .sisyphus/evidence/task-14-max-rows.log
  \`\`\`

  **Commit**: YES
  - Message: `feat(cli): add --max-rows flag to protheus query`
  - Files: `internal/cli/protheus.go`

---

- [ ] 15. Fix json.MarshalIndent error in export.go

  **What to do**:
  - In `pkg/confluence/export.go:60`:
    - Change:
      ```go
      data, _ := json.MarshalIndent(page, "", "  ")
      result.Content = string(data)
      ```
    - To:
      ```go
      data, err := json.MarshalIndent(page, "", "  ")
      if err != nil {
          return nil, fmt.Errorf("failed to marshal page to JSON: %w", err)
      }
      result.Content = string(data)
      ```

  **Must NOT do**:
  - Don't break the export functionality

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple error handling fix
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 12, 13, 14)
  - **Blocks**: Task 16
  - **Blocked By**: None

  **References**:
  - `pkg/confluence/export.go:59-62` - Current code

  **Acceptance Criteria**:
  - [ ] Export to JSON still works for valid pages
  - [ ] Errors are properly returned (not ignored)

  **QA Scenarios**:

  \`\`\`
  Scenario: JSON export handles errors
    Tool: Bash
    Preconditions: Valid credentials
    Steps:
      1. mapj confluence export 12345 --format json
    Expected Result: Valid JSON output or proper error
    Failure Indicators: Panic or silent failure
    Evidence: .sisyphus/evidence/task-15-json-marshal.log
  \`\`\`

  **Commit**: YES
  - Message: `fix(confluence): handle json.MarshalIndent errors`
  - Files: `pkg/confluence/export.go`

---

## Final Verification Wave

- [ ] F1. **Run all tests** — `go test ./...`
  Output: `All tests pass | VERDICT`

- [ ] F2. **Build verification** — `go build -o mapj ./cmd/mapj/`
  Output: `Build successful | VERDICT`

- [ ] F3. **CLI integration test** — Execute key scenarios from all tasks
  Output: `Scenarios [N/N pass] | VERDICT`

- [ ] F4. **Plan compliance audit** — Verify all Must Have items implemented
  Output: `Compliance [N/N] | VERDICT`

---

## Commit Strategy

- Wave 1: `feat(auth): environment variable key + machine-derived fallback`
- Wave 2: `fix(cli): proper error returns and exit codes`
- Wave 3: `feat(cli): UX improvements (validation, flags)`
- Final: `chore: update skills for new behavior`

---

## Success Criteria

### Verification Commands
```bash
go test ./...                          # All pass
go build -o mapj ./cmd/mapj/          # Success
grep -r "mapj-cred-key" --include="*.go"  # 0 matches
```

### Final Checklist
- [ ] MAPJ_ENCRYPTION_KEY works
- [ ] Fallback key works
- [ ] Exit codes return correctly (1, 2, 3, 4)
- [ ] --limit validation works
- [ ] --max-rows flag works
- [ ] --no-color works or removed
- [ ] Auth wiring explicit
- [ ] All tests pass
- [ ] Build succeeds
- [ ] No hardcoded key found
