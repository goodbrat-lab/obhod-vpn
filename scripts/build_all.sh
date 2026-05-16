#!/bin/bash

# Build all packages for all supported architectures
# Automatically compiles Go, packages IPKs, and pushes to GitHub.

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

echo "=================================================="
echo "  Obhod Full Build & Auto-Publish"
echo "  Base: $BASE_DIR"
echo "=================================================="

# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin

# Step 1: Building Go binaries
echo ""
echo "Step 1: Building Go binaries (obhoud)..."
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

# Step 5: Automatic Publishing to GitHub
echo ""
echo "Step 5: Publishing to GitHub..."
git add .
git commit -m "Build and release: Obhod v0.3.3 (Validation Support)" || echo "No changes to commit"
git push origin main || echo "Warning: git push failed."

echo "========================================="
echo "  RELEASE PUBLISHED TO GITHUB"
echo "  Installer: sh <(wget -q -O - https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)"
echo "========================================="
