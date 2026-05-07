#!/bin/bash

# Cross-compilation script for obhoud (Go daemon)
# Usage: ./scripts/build_go.sh  (from any working directory)

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

SRC_DIR="$BASE_DIR/obhod-core/src"
DIST_DIR="$BASE_DIR/dist/binaries"
GO_BIN="${GO:-go}" # Use $GO env var or fall back to 'go' in PATH

echo "Base dir : $BASE_DIR"
echo "Source   : $SRC_DIR"
echo "Output   : $DIST_DIR"

mkdir -p "$DIST_DIR"

build() {
    local os=$1
    local arch=$2
    local variant=$3 # GOMIPS or GOARM value (e.g. "softfloat" or "v7")
    local output_name="obhoud_${os}_${arch}"

    if [ -n "$variant" ]; then
        output_name="${output_name}_${variant}"
    fi

    echo "Building for $os/$arch $variant -> $output_name"

    cd "$SRC_DIR" || { echo "Error: cannot cd to $SRC_DIR"; exit 1; }

    # Reset env vars from any previous iteration
    export GOOS=$os
    export GOARCH=$arch
    export CGO_ENABLED=0
    unset GOMIPS
    unset GOARM

    if [ "$arch" == "mips" ] || [ "$arch" == "mipsle" ]; then
        export GOMIPS=$variant
    elif [ "$arch" == "arm" ]; then
        export GOARM=${variant#v} # remove leading 'v' if present
    fi

    "$GO_BIN" build -ldflags="-s -w" -o "$DIST_DIR/$output_name" .
    local exit_code=$?
    if [ "$exit_code" -ne 0 ]; then
        echo "ERROR: Build failed for $os/$arch ($variant), exit code $exit_code"
        exit "$exit_code"
    fi
    echo "  -> $DIST_DIR/$output_name ($(du -h "$DIST_DIR/$output_name" | cut -f1))"
}

# Common OpenWrt architectures
build "linux" "mipsle" "softfloat"   # Xiaomi, TP-Link (32-bit LE MIPS)
build "linux" "mips"   "softfloat"   # Some Asus, D-Link (32-bit BE MIPS)
build "linux" "arm64"  ""            # Qualcomm IPQ, Raspberry Pi (64-bit ARM)
build "linux" "arm"    "v7"          # Older routers (32-bit ARMv7)
build "linux" "amd64"  ""            # x86_64 (PC/VM)

echo ""
echo "=== All binaries built successfully ==="
echo "Output directory: $DIST_DIR"
ls -lh "$DIST_DIR"/obhoud_*
