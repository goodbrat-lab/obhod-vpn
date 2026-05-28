#!/bin/bash
# Build script for Obhod packages
# Usage: ./scripts/build.sh [architecture]

set -e

ARCH="${1:-x86_64}"
SDK_PATH="${SDK_PATH:-/opt/openwrt-sdk}"
VERSION="1.1.27"
RELEASE="1"

echo "=================================================="
echo "      Obhod VPN Build Script v1.1.27"
echo "=================================================="
echo "Architecture: $ARCH"
echo "SDK Path: $SDK_PATH"
echo "Version: $VERSION-$RELEASE"
echo "=================================================="

# Check SDK exists
if [ ! -d "$SDK_PATH" ]; then
    echo "ERROR: OpenWrt SDK not found at $SDK_PATH"
    echo "Please set SDK_PATH or install OpenWrt SDK"
    exit 1
fi

# Build luci-app-obhod
echo "Building luci-app-obhod..."
cd luci-app-obhod
make -C "$SDK_PATH" \
    -f "$(pwd)/Makefile" \
    package/index \
    package/clean \
    package/compile || {
    echo "ERROR: Failed to build luci-app-obhod"
    exit 1
}

# Build obhod-core
echo "Building obhod-core..."
cd ../obhod-core
make -C "$SDK_PATH" \
    -f "$(pwd)/Makefile" \
    package/index \
    package/clean \
    package/compile || {
    echo "ERROR: Failed to build obhod-core"
    exit 1
}

# Collect packages
echo "Collecting packages..."
mkdir -p dist/packages
find "$SDK_PATH/bin/packages" -name "*obhod*" -type f | while read -r pkg; do
    echo "Found package: $pkg"
    cp "$pkg" dist/packages/
done

# Generate sha256sums
echo "Generating checksums..."
cd dist/packages
sha256sum ./* > SHA256SUMS

echo "=================================================="
echo "Build completed successfully!"
echo "Packages location: $(pwd)"
echo "=================================================="

# Show package sizes
echo "Package sizes:"
ls -lh ./*.ipk
