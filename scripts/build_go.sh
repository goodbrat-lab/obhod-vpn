#!/bin/bash

# Cross-compilation script for obhoud (Go daemon)
# Usage: ./build_go.sh

SRC_DIR="/root/Obhod project/obhod-core/src"
DIST_DIR="/root/Obhod project/dist/binaries"
GO_BIN="/usr/local/go/bin/go"

mkdir -p "$DIST_DIR"

build() {
    local os=$1
    local arch=$2
    local variant=$3 # GOMIPS or GOARM
    local output_name="obhoud_${os}_${arch}"
    
    if [ -n "$variant" ]; then
        output_name="${output_name}_${variant}"
    fi

    echo "Building for $os/$arch $variant..."
    
    cd "$SRC_DIR"
    
    # Reset env vars
    export GOOS=$os
    export GOARCH=$arch
    export CGO_ENABLED=0
    unset GOMIPS
    unset GOARM

    if [ "$arch" == "mips" ] || [ "$arch" == "mipsle" ]; then
        export GOMIPS=$variant
    elif [ "$arch" == "arm" ]; then
        export GOARM=${variant#v} # remove 'v' if exists
    fi

    "$GO_BIN" build -ldflags="-s -w" -o "$DIST_DIR/$output_name" .
}

# Common OpenWrt Architectures
build "linux" "mipsle" "softfloat"
build "linux" "mips" "softfloat"
build "linux" "arm64" ""
build "linux" "arm" "v7"
build "linux" "amd64" ""

echo "All binaries are in $DIST_DIR"
