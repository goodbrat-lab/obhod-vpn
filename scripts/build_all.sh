#!/bin/bash

# Build all packages for all supported architectures.
# This script only builds artifacts locally.

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

echo "=================================================="
echo "  Obhod Full Build"
echo "  Base: $BASE_DIR"
echo "=================================================="

# Ensure Go is in PATH
export PATH="$PATH:/usr/local/go/bin"

# Step 1: Building Go binaries
echo ""
echo "Step 1: Building Go binaries (obhod)..."
./scripts/build_go.sh || { echo "ERROR: Go build failed."; exit 1; }

# Step 2: Building .ipk packages
echo ""
echo "Step 2: Building .ipk packages..."
# Architecture list matches OpenWrt standard names used in build_go.sh
ARCH_LIST="mipsel_24kc mips_24kc aarch64_cortex-a53 arm_cortex-a7_neon-vfpv4 x86_64"

for ARCH in $ARCH_LIST; do
    ./scripts/package_full.sh "$ARCH" || { echo "ERROR: Packaging failed for $ARCH."; exit 1; }
done

# Step 3: LuCI Package
./scripts/package_luci.sh || { echo "ERROR: LuCI packaging failed."; exit 1; }

# Step 4: Update Index
./scripts/update_index.sh || { echo "ERROR: Index update failed."; exit 1; }

echo ""
echo "========================================="
echo "  BUILD COMPLETE"
echo "  Packages: $BASE_DIR/dist/packages/"
echo "========================================="
ls -lh "$BASE_DIR/dist/packages/"*.ipk

echo "========================================="
echo "  BUILD ARTIFACTS READY"
echo "  Review and publish separately"
echo "========================================="
