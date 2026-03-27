# Changelog

All notable changes to the `mapj` CLI tool will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [1.0.0] - 2026-03-26

### Added

#### Commands

- **`mapj tdn search`** - Search TOTVS Developer Network (TDN) documentation
  - CQL-based search with space and label filters
  - Configurable result limit (1-100)
  - JSON and table output formats

- **`mapj confluence export`** - Export Confluence pages
  - Supports URL, page ID, or relative path input
  - Export formats: markdown, HTML, JSON
  - Optional comment inclusion

- **`mapj protheus query`** - Execute SELECT queries on Protheus database
  - SQL Server connection
  - JSON and CSV output formats
  - SELECT-only enforcement (security constraint)

- **`mapj auth login`** - Authenticate to services
  - TDN/Confluence authentication (PAT or Basic Auth)
  - Protheus database authentication

- **`mapj auth status`** - Show authentication status for all services

- **`mapj auth logout`** - Remove stored credentials

#### Core Features

- **Secure credential storage** - AES-256-GCM encrypted storage at `~/.config/mapj/credentials.enc`
- **JSON output by default** - Machine-readable envelope format
- **Exit codes** - Standardized error codes (0-5) for agentic error handling
- **Structured errors** - Consistent error format with code, message, and retryable flag
- **Self-describing** - Built-in help and schema discovery

#### Documentation

- **SKILL.md** - Main manifest following agentskills.io standard
- **Individual command skills** - Detailed documentation for each command
- **Changelog** - Version tracking

### Fixed

- **`Version.By` parsing bug** - Confluence API returns object, not string
- **`Body.ExportView` parsing bug** - Confluence API returns object, not string
- **User-Agent header** - Added to bypass Cloudflare WAF
- **Authentication method** - Support for both Basic Auth and Bearer token

### Security

- **SELECT-only enforcement** - INSERT, UPDATE, DELETE, DROP, etc. are blocked
- **Credential encryption** - AES-256-GCM for stored credentials
- **No secrets in code** - All secrets via environment or secure storage

## Architecture

```
mapj/
├── cmd/mapj/main.go           # Entry point
├── internal/
│   ├── cli/                   # CLI command definitions
│   ├── auth/                  # Authentication & credential storage
│   ├── errors/                # Exit codes and error types
│   └── output/                # JSON envelope formatting
├── pkg/
│   ├── confluence/            # Confluence/TDN API client
│   └── protheus/             # Protheus SQL client
└── skills/                   # AI agent documentation
    ├── SKILL.md              # Main manifest
    ├── mapj-tdn-search.md
    ├── mapj-confluence-export.md
    ├── mapj-protheus-query.md
    └── CHANGELOG.md
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/):
- MAJOR version for incompatible API changes
- MINOR version for backwards-compatible functionality
- PATCH version for backwards-compatible bug fixes

## Deprecation Policy

- Deprecated commands will output warnings but continue to work
- Removal will only occur in major version updates
- Migration paths will be documented in skill files
