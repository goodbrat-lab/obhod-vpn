#!/bin/bash

# Obhod build automation script
# Usage: ./build.sh [mipsle|arm|aarch64]

ARCH=${1:-mipsle}
SDK_PATH=${SDK_PATH:-/opt/openwrt-sdk}

echo "Building Obhod for architecture: $ARCH"

if [ ! -d "$SDK_PATH" ]; then
    echo "Error: OpenWrt SDK not found at $SDK_PATH"
    echo "Please set SDK_PATH environment variable."
    exit 1
fi

# Prepare environment
export GOARCH=$ARCH
if [ "$ARCH" == "mipsle" ]; then export GOARCH=mipsle; fi
if [ "$ARCH" == "arm" ]; then export GOARCH=arm; fi
if [ "$ARCH" == "aarch64" ]; then export GOARCH=arm64; fi

echo "Using GOARCH=$GOARCH"

# Link package to SDK
mkdir -p "$SDK_PATH/package/obhod"
cp -r ../Makefile ../backend ../files "$SDK_PATH/package/obhod/"
cp -r ../luci-app-obhod "$SDK_PATH/package/"

# Build
cd "$SDK_PATH"
make package/obhod/compile V=s
make package/luci-app-obhod/compile V=s

# Collect results
mkdir -p "$(dirname "$0")/../dist"
find bin/packages -name "obhod*.ipk" -exec cp {} "$(dirname "$0")/../dist/" \;
find bin/packages -name "luci-app-obhod*.ipk" -exec cp {} "$(dirname "$0")/../dist/" \;

echo "Build complete. Packages are in dist/ directory."
