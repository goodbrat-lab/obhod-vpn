#!/bin/bash

# Cross-compilation script for obhod (Go daemon)
# Usage: ./scripts/build_go.sh  (from any working directory)

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

SRC_DIR="$BASE_DIR/obhod-core/src"
DIST_DIR="$BASE_DIR/dist/binaries"
GO_BIN="/usr/local/go/bin/go"
[ ! -f "$GO_BIN" ] && GO_BIN="go"

echo "Base dir : $BASE_DIR"
echo "Source   : $SRC_DIR"
echo "Output   : $DIST_DIR"

mkdir -p "$DIST_DIR"

build() {
    local os=$1
    local arch=$2
    local variant=$3 # GOMIPS or GOARM value (e.g. "softfloat" or "v7")
    local suffix=$4  # OpenWrt architecture name (e.g. "aarch64_cortex-a53")
    local output_name="obhod_${os}_${suffix}"

    echo "Building for $os/$arch $variant (Target: $suffix) -> $output_name"

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

    if command -v upx > /dev/null; then
        echo "  -> Compressing with UPX..."
        upx -9 "$DIST_DIR/$output_name" > /dev/null 2>&1 || echo "  -> UPX failed, continuing..."
    fi

    echo "  -> $DIST_DIR/$output_name ($(du -h "$DIST_DIR/$output_name" | cut -f1))"
}

# Common OpenWrt architectures (Mapping Go builds to OpenWrt arch names)
build "linux" "mipsle" "softfloat" "mipsel_24kc"
build "linux" "mips"   "softfloat" "mips_24kc"
build "linux" "arm64"  ""          "aarch64_cortex-a53"
build "linux" "arm"    "v7"        "arm_cortex-a7_neon-vfpv4"
build "linux" "amd64"  ""          "x86_64"

echo ""
echo "=== All binaries built successfully ==="
echo "Output directory: $DIST_DIR"
ls -lh "$DIST_DIR"/obhod_*
