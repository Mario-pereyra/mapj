#!/bin/bash
# Init script for observability feature
cd "$(dirname "$0")/.."

# Ensure dependencies
go mod tidy

# Build to verify
go build -o mapj.exe ./cmd/mapj

echo "Setup complete"
