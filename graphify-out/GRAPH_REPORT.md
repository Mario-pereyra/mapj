# Graph Report - .  (2026-04-24)

## Corpus Check
- 86 files · ~80,072 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 983 nodes · 2382 edges · 56 communities detected
- Extraction: 57% EXTRACTED · 43% INFERRED · 0% AMBIGUOUS · INFERRED: 1029 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Advpl Compilation Pipeline|Advpl Compilation Pipeline]]
- [[_COMMUNITY_Health & Client Connection|Health & Client Connection]]
- [[_COMMUNITY_SQL Injection Detection|SQL Injection Detection]]
- [[_COMMUNITY_Integration Tests|Integration Tests]]
- [[_COMMUNITY_Protheus Preset Tests|Protheus Preset Tests]]
- [[_COMMUNITY_Confluence Export System|Confluence Export System]]
- [[_COMMUNITY_Command Metrics|Command Metrics]]
- [[_COMMUNITY_Documentation & Knowledge|Documentation & Knowledge]]
- [[_COMMUNITY_Credential Store|Credential Store]]
- [[_COMMUNITY_TDS Protocol Client|TDS Protocol Client]]
- [[_COMMUNITY_TDS Protocol Types|TDS Protocol Types]]
- [[_COMMUNITY_Output Formatter|Output Formatter]]
- [[_COMMUNITY_TOON Formatter|TOON Formatter]]
- [[_COMMUNITY_Credential Encryption|Credential Encryption]]
- [[_COMMUNITY_Error Codes|Error Codes]]
- [[_COMMUNITY_Logging System|Logging System]]
- [[_COMMUNITY_Observability|Observability]]
- [[_COMMUNITY_Auth CLI Commands|Auth CLI Commands]]
- [[_COMMUNITY_Confluence URL Parsing|Confluence URL Parsing]]
- [[_COMMUNITY_Confluence Search|Confluence Search]]
- [[_COMMUNITY_Confluence Macro Test Data|Confluence Macro Test Data]]
- [[_COMMUNITY_Confluence Markdown Export|Confluence Markdown Export]]
- [[_COMMUNITY_CLI Entrypoint|CLI Entrypoint]]
- [[_COMMUNITY_Confluence Attachments|Confluence Attachments]]
- [[_COMMUNITY_Query Security Detection|Query Security Detection]]
- [[_COMMUNITY_Query Security Implementation|Query Security Implementation]]
- [[_COMMUNITY_Observable Command Interface|Observable Command Interface]]
- [[_COMMUNITY_Command Metrics Recording|Command Metrics Recording]]
- [[_COMMUNITY_TDN Client Integration|TDN Client Integration]]
- [[_COMMUNITY_Preset Store System|Preset Store System]]
- [[_COMMUNITY_LLM Formatter|LLM Formatter]]
- [[_COMMUNITY_LDFlags Build|LDFlags Build]]
- [[_COMMUNITY_String Escape Value|String Escape Value]]
- [[_COMMUNITY_Protheus Query|Protheus Query]]
- [[_COMMUNITY_Export Spaces Pipeline|Export Spaces Pipeline]]
- [[_COMMUNITY_TDN List Spaces|TDN List Spaces]]
- [[_COMMUNITY_Auth Store Test|Auth Store Test]]
- [[_COMMUNITY_Params Test|Params Test]]
- [[_COMMUNITY_Store Test|Store Test]]
- [[_COMMUNITY_Logger Test|Logger Test]]
- [[_COMMUNITY_Output Test|Output Test]]
- [[_COMMUNITY_Preset Edit Remove Test|Preset Edit Remove Test]]
- [[_COMMUNITY_Export Test|Export Test]]
- [[_COMMUNITY_Protheus Validation Test|Protheus Validation Test]]
- [[_COMMUNITY_Markdown Test|Markdown Test]]
- [[_COMMUNITY_Confluence URL Test|Confluence URL Test]]
- [[_COMMUNITY_Codes Test|Codes Test]]
- [[_COMMUNITY_Preset Integration Test|Preset Integration Test]]
- [[_COMMUNITY_Observability Test|Observability Test]]
- [[_COMMUNITY_TOON Formatter Test|TOON Formatter Test]]
- [[_COMMUNITY_Confluence Export Skill|Confluence Export Skill]]
- [[_COMMUNITY_Protheus Query Skill|Protheus Query Skill]]
- [[_COMMUNITY_TDN Search Skill|TDN Search Skill]]
- [[_COMMUNITY_Mapj Main Skill|Mapj Main Skill]]
- [[_COMMUNITY_CQL Reference|CQL Reference]]
- [[_COMMUNITY_Connection References|Connection References]]

## God Nodes (most connected - your core abstractions)
1. `NewEnvelope()` - 67 edges
2. `createTestStore()` - 47 edges
3. `GetFormatter()` - 47 edges
4. `NewErrorEnvelope()` - 46 edges
5. `Execute()` - 32 edges
6. `Client` - 29 edges
7. `NewStore()` - 27 edges
8. `NewErrorEnvelopeWithHint()` - 26 edges
9. `Client` - 26 edges
10. `withTDSClient()` - 23 edges

## Surprising Connections (you probably didn't know these)
- `presetRunRun()` --calls--> `GetInterpolationError()`  [INFERRED]
  D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\protheus_preset.go → D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\preset\escape.go
- `TestGetFormatter_WithJSON()` --calls--> `GetFormatter()`  [INFERRED]
  D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\protheus_preset_test.go → D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\root.go
- `TestGetFormatter_WithVerbose()` --calls--> `GetFormatter()`  [INFERRED]
  D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\protheus_preset_test.go → D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\internal\cli\root.go
- `TestNewClient()` --calls--> `NewClient()`  [INFERRED]
  D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\pkg\protheus\protheus_validation_test.go → D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\pkg\tds\client.go
- `TestClient_ManifestMutex_ConcurrentWrites()` --calls--> `WriteManifest()`  [INFERRED]
  D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\pkg\confluence\export_test.go → D:\Proyectos_Personales\CLI\mapj_cli\mapj_cli\pkg\confluence\writer.go

## Communities

### Community 0 - "Advpl Compilation Pipeline"
Cohesion: 0.04
Nodes (111): advplCompileRun(), collectSourceFiles(), resolveTDSProfile(), tdsConnAddRun(), tdsConnListRun(), tdsConnPingRun(), tdsConnRemoveRun(), tdsConnShowRun() (+103 more)

### Community 1 - "Health & Client Connection"
Cohesion: 0.04
Nodes (54): healthResult, serviceHealth, backoffWithJitter(), NewClient(), AncestorRef, APIError, APIResponse, Body (+46 more)

### Community 2 - "SQL Injection Detection"
Cohesion: 0.04
Nodes (87): DetectSQLInjection(), EscapeListValue(), EscapeStringValue(), formatValueForSQL(), GetInterpolationError(), InterpolateQuery(), containsPattern(), TestDetectSQLInjection_CombinedPatterns() (+79 more)

### Community 3 - "Integration Tests"
Cohesion: 0.09
Nodes (70): integrationTestEnv, Fatal(), ParamDef, PresetFile, QueryPreset, cleanupEditRemoveUseTest(), createEditCommand(), createRemoveCommand() (+62 more)

### Community 4 - "Protheus Preset Tests"
Cohesion: 0.08
Nodes (68): testFlags, PresetStore, ResetPresetStore(), SetPresetStoreForTest(), createPresetListCmdForTest(), createTestStore(), executePresetAdd(), executePresetAddError() (+60 more)

### Community 5 - "Confluence Export System"
Cohesion: 0.09
Nodes (45): TestClient_ManifestMutex_ConcurrentWrites(), TestExport_MarkdownFormat(), cleanupMarkdown(), ConvertToMarkdown(), extractStorageMacroBody(), extractText(), findChildByClass(), findChildByTag() (+37 more)

### Community 6 - "Command Metrics"
Cohesion: 0.08
Nodes (34): CommandMetricEntry, CommandMetrics, CounterEntry, LabeledCounters, Metrics, ObservableCommand, panicObservable, PromMetricsOutput (+26 more)

### Community 7 - "Documentation & Knowledge"
Cohesion: 0.06
Nodes (45): Auto-Healing Client, Auto-Pagination, CHANGELOG, Concurrent Worker Pool, Confluence Basic Auth, Confluence Bearer PAT Auth, mapj Confluence Export Skill, Confluence Service (+37 more)

### Community 8 - "Credential Store"
Cohesion: 0.09
Nodes (34): ConfluenceCreds, CredentialStore, AES-GCM encrypted file credential store, ProtheusProfile, ServiceCreds, TDNCreds, TDSProfile, outputFlagFromArgs() (+26 more)

### Community 9 - "TDS Protocol Client"
Cohesion: 0.11
Nodes (6): parseFunctions(), parseObjects(), Client, jsonRPCError, jsonRPCRequest, jsonRPCResponse

### Community 10 - "TDS Protocol Types"
Cohesion: 0.06
Nodes (33): AuthParams, AuthResult, CompileDiagnostic, CompileFileInfo, CompileParams, CompileResult, ConnectParams, ConnectResult (+25 more)

### Community 11 - "Output Formatter"
Cohesion: 0.11
Nodes (23): Auto Formatter, Response Envelope Struct, Error Detail Struct, Error Type Structs, Execute Function, Exit Code Convention Design, ExitCoder Interface, Exit Code Constants (+15 more)

### Community 12 - "TOON Formatter"
Cohesion: 0.22
Nodes (4): TOONFormatter, TestTOONFormatter_escapeString(), TestTOONFormatter_isUniformObjects(), TestTOONFormatter_needsQuoting()

### Community 13 - "Credential Encryption"
Cohesion: 0.13
Nodes (10): ConfluenceCreds, CredentialStore, ProtheusProfile, TestCredentialStore_EncryptDecrypt(), TestCredentialStore_HasService(), TestCredentialStore_LoadNonExistent(), TDNCreds, TDSProfile (+2 more)

### Community 14 - "Error Codes"
Cohesion: 0.1
Nodes (8): MapErrorToCode(), TestMapErrorToCode(), AuthError, ConflictError, ExitCoder, GeneralError, RetryableError, UsageError

### Community 15 - "Logging System"
Cohesion: 0.2
Nodes (15): GetLevel(), Init(), parseLevel(), SetLevel(), TestGenerateTraceID(), TestGenerateTraceIDWithEnvOverride(), TestGetLevel(), TestInit() (+7 more)

### Community 16 - "Observability"
Cohesion: 0.38
Nodes (9): Debug(), Error(), GetCommandPath(), Info(), LogCommandComplete(), LogCommandStart(), Warn(), WithCommandMetadata() (+1 more)

### Community 17 - "Auth CLI Commands"
Cohesion: 0.31
Nodes (6): init(), AddCommands(), confluenceLogin(), isCloudURL(), loginJSON(), tdnLogin()

### Community 18 - "Confluence URL Parsing"
Cohesion: 0.36
Nodes (6): ParseResult, TestParseConfluenceInput(), decodeURLComponent(), indexOf(), ParseConfluenceInput(), splitPath()

### Community 19 - "Confluence Search"
Cohesion: 0.33
Nodes (6): PageRef, SearchOpts, SearchResult, SpaceRef, buildCQL(), parseSinceExpr()

### Community 20 - "Confluence Macro Test Data"
Cohesion: 0.29
Nodes (7): Confluence Code Macro, Confluence Task List Macro, Confluence Table with Colspan, Test: Code Macro HTML, Test: Mixed Content HTML, Test: Complex Table HTML, Test: Task List HTML

### Community 21 - "Confluence Markdown Export"
Cohesion: 0.33
Nodes (6): Client, ConvertToMarkdown, ExportError, ExportLogger, ManifestEntry, Page

### Community 22 - "CLI Entrypoint"
Cohesion: 0.5
Nodes (3): Execute(), SilenceErrors for JSON envelope output, main()

### Community 23 - "Confluence Attachments"
Cohesion: 0.5
Nodes (3): AttachmentExtensions, AttachmentInfo, AttachmentLinks

### Community 24 - "Query Security Detection"
Cohesion: 0.5
Nodes (4): Parameter Detection, Parameter Type Validation, Query Interpolation, SQL Injection Detection

### Community 25 - "Query Security Implementation"
Cohesion: 0.5
Nodes (4): DetectParameters, DetectSQLInjection, InterpolateQuery, ValidateParamType

### Community 26 - "Observable Command Interface"
Cohesion: 0.67
Nodes (2): ObservableCommand interface, ObservableCommand opt-in interface

### Community 27 - "Command Metrics Recording"
Cohesion: 0.67
Nodes (3): CommandMetrics, LabeledCounters, RecordCommandMetrics()

### Community 28 - "TDN Client Integration"
Cohesion: 0.67
Nodes (3): Confluence Attachment Handling, Confluence API Client, Get TDN Client Helper

### Community 29 - "Preset Store System"
Cohesion: 0.67
Nodes (3): Preset Atomic Write Design, Preset Store, Query Preset Struct

### Community 32 - "LLM Formatter"
Cohesion: 1.0
Nodes (1): PipelineResult

### Community 33 - "LDFlags Build"
Cohesion: 1.0
Nodes (1): SpaceDetail

### Community 34 - "String Escape Value"
Cohesion: 1.0
Nodes (2): Client, ValidateReadOnly

### Community 35 - "Protheus Query"
Cohesion: 1.0
Nodes (2): Confluence Expand Macro, Test: Expand Macro HTML

### Community 36 - "Export Spaces Pipeline"
Cohesion: 1.0
Nodes (2): Confluence Link Macro, Test: Internal Links HTML

### Community 37 - "TDN List Spaces"
Cohesion: 1.0
Nodes (2): Confluence Image Macro, Test: Attachments HTML

### Community 38 - "Auth Store Test"
Cohesion: 1.0
Nodes (2): Confluence Info/Warning/Tip Macros, Test: Info/Warning Panel HTML

### Community 39 - "Params Test"
Cohesion: 1.0
Nodes (2): Confluence Status Macro, Test: Status Macro HTML

### Community 41 - "Store Test"
Cohesion: 1.0
Nodes (1): CLI Package

### Community 42 - "Logger Test"
Cohesion: 1.0
Nodes (1): Logging Package

### Community 43 - "Output Test"
Cohesion: 1.0
Nodes (1): ExportOpts

### Community 44 - "Preset Edit Remove Test"
Cohesion: 1.0
Nodes (1): ExportResult

### Community 45 - "Export Test"
Cohesion: 1.0
Nodes (1): APIError

### Community 46 - "Protheus Validation Test"
Cohesion: 1.0
Nodes (1): SearchOpts

### Community 47 - "Markdown Test"
Cohesion: 1.0
Nodes (1): SearchResult

### Community 48 - "Confluence URL Test"
Cohesion: 1.0
Nodes (1): PipelineResult

### Community 49 - "Codes Test"
Cohesion: 1.0
Nodes (1): SpaceDetail

### Community 50 - "Preset Integration Test"
Cohesion: 1.0
Nodes (1): ParseResult

### Community 51 - "Observability Test"
Cohesion: 1.0
Nodes (1): QueryResult

### Community 52 - "TOON Formatter Test"
Cohesion: 1.0
Nodes (1): Client

### Community 53 - "Confluence Export Skill"
Cohesion: 1.0
Nodes (1): ValidationResult

### Community 54 - "Protheus Query Skill"
Cohesion: 1.0
Nodes (1): ConnectResult

### Community 55 - "TDN Search Skill"
Cohesion: 1.0
Nodes (1): AuthResult

### Community 56 - "Mapj Main Skill"
Cohesion: 1.0
Nodes (1): PresetStore

### Community 57 - "CQL Reference"
Cohesion: 1.0
Nodes (1): QueryPreset

### Community 58 - "Connection References"
Cohesion: 1.0
Nodes (1): ParamDef

## Knowledge Gaps
- **157 isolated node(s):** `authStatusResult`, `serviceStatus`, `protheusStatus`, `TDNCreds`, `ConfluenceCreds` (+152 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Observable Command Interface`** (3 nodes): `ObservableCommand interface`, `RegisterObservable()`, `ObservableCommand opt-in interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `LLM Formatter`** (2 nodes): `PipelineResult`, `search_pipeline.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `LDFlags Build`** (2 nodes): `SpaceDetail`, `spaces.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `String Escape Value`** (2 nodes): `Client`, `ValidateReadOnly`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Protheus Query`** (2 nodes): `Confluence Expand Macro`, `Test: Expand Macro HTML`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Export Spaces Pipeline`** (2 nodes): `Confluence Link Macro`, `Test: Internal Links HTML`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `TDN List Spaces`** (2 nodes): `Confluence Image Macro`, `Test: Attachments HTML`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Auth Store Test`** (2 nodes): `Confluence Info/Warning/Tip Macros`, `Test: Info/Warning Panel HTML`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Params Test`** (2 nodes): `Confluence Status Macro`, `Test: Status Macro HTML`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Store Test`** (1 nodes): `CLI Package`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Logger Test`** (1 nodes): `Logging Package`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Output Test`** (1 nodes): `ExportOpts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Preset Edit Remove Test`** (1 nodes): `ExportResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Export Test`** (1 nodes): `APIError`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Protheus Validation Test`** (1 nodes): `SearchOpts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Markdown Test`** (1 nodes): `SearchResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Confluence URL Test`** (1 nodes): `PipelineResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Codes Test`** (1 nodes): `SpaceDetail`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Preset Integration Test`** (1 nodes): `ParseResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Observability Test`** (1 nodes): `QueryResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `TOON Formatter Test`** (1 nodes): `Client`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Confluence Export Skill`** (1 nodes): `ValidationResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Protheus Query Skill`** (1 nodes): `ConnectResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `TDN Search Skill`** (1 nodes): `AuthResult`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Mapj Main Skill`** (1 nodes): `PresetStore`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `CQL Reference`** (1 nodes): `QueryPreset`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Connection References`** (1 nodes): `ParamDef`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Execute()` connect `Integration Tests` to `Protheus Preset Tests`, `Command Metrics`, `Error Codes`, `Observability`, `CLI Entrypoint`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Why does `ConvertToMarkdown()` connect `Confluence Export System` to `Health & Client Connection`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Why does `observeCommand()` connect `Command Metrics` to `SQL Injection Detection`, `Integration Tests`?**
  _High betweenness centrality (0.052) - this node is a cross-community bridge._
- **Are the 66 inferred relationships involving `NewEnvelope()` (e.g. with `loginJSON()` and `logoutRun()`) actually correct?**
  _`NewEnvelope()` has 66 INFERRED edges - model-reasoned connections that need verification._
- **Are the 46 inferred relationships involving `GetFormatter()` (e.g. with `advplCompileRun()` and `tdsConnAddRun()`) actually correct?**
  _`GetFormatter()` has 46 INFERRED edges - model-reasoned connections that need verification._
- **Are the 45 inferred relationships involving `NewErrorEnvelope()` (e.g. with `advplCompileRun()` and `tdsConnAddRun()`) actually correct?**
  _`NewErrorEnvelope()` has 45 INFERRED edges - model-reasoned connections that need verification._
- **Are the 31 inferred relationships involving `Execute()` (e.g. with `main()` and `TestPresetEditSuccess()`) actually correct?**
  _`Execute()` has 31 INFERRED edges - model-reasoned connections that need verification._