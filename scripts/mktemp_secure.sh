#!/bin/sh
# Create a secure temp directory script for Obhod
# Usage: mktemp_secure.sh [prefix]

TMP_PREFIX="${1:-obhod_secure}"

# Try to use mktemp if available
if command -v mktemp >/dev/null 2>&1; then
    # Create directory with restricted permissions
    mktemp -d -t "${TMP_prefix}_XXXXXX" 2>/dev/null && exit 0
fi

# Fallback: create with date and random
TMP_DIR="/tmp/${TMP_PREFIX}_$(date +%s)_$$"

# Create directory with safe permissions
if mkdir -m 700 "$TMP_DIR" 2>/dev/null; then
    echo "$TMP_DIR"
else
    echo "Failed to create secure temp directory" >&2
    exit 1
fi