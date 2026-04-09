#!/bin/bash
# Initialize Rust environment for mapj migration

set -e

# Ensure Rust toolchain is available
if ! command -v cargo &> /dev/null; then
    echo "Rust toolchain not found. Please install from https://rustup.rs"
    exit 1
fi

# Navigate to the project directory
cd "$(dirname "$0")/.."

# Create the rust/minimax-2.7 branch
if ! git rev-parse --verify rust/minimax-2.7 &>/dev/null; then
    git checkout -b rust/minimax-2.7
    echo "Created and checked out branch: rust/minimax-2.7"
else
    git checkout rust/minimax-2.7
    echo "Checked out existing branch: rust/minimax-2.7"
fi

echo "Environment initialized for mapj Rust migration"
