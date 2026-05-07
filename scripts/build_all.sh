#!/bin/bash

# BUILD EVERYTHING: Go binaries + .ipk packages for all architectures
# Usage: ./scripts/build_all.sh  (from any working directory)

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

echo "=================================================="
echo "  Obhod Full Build"
echo "  Base: $BASE_DIR"
echo "=================================================="

# Step 1: Compile Go daemon for all architectures
echo ""
echo "Step 1: Building Go binaries (obhoud)..."
bash "$SCRIPTS_DIR/build_go.sh"
if [ $? -ne 0 ]; then
    echo "ERROR: Go build failed. Aborting."
    exit 1
fi

# Step 2: Package for each architecture
echo ""
echo "Step 2: Building .ipk packages..."
ARCHS=("mipsle_softfloat" "mips_softfloat" "arm64" "arm_v7" "amd64")

for arch in "${ARCHS[@]}"; do
    echo ""
    bash "$SCRIPTS_DIR/package_full.sh" "$arch"
    if [ $? -ne 0 ]; then
        echo "ERROR: Packaging failed for $arch. Aborting."
        exit 1
    fi
done

# Step 3: Update dist index
echo ""
echo "Step 3: Updating dist/packages/index.txt..."
INDEX_FILE="$BASE_DIR/dist/packages/index.txt"
rm -f "$INDEX_FILE"
for arch in "${ARCHS[@]}"; do
    filename=$(ls "$BASE_DIR/dist/packages/" 2>/dev/null | grep "_${arch}.ipk" | head -n1)
    if [ -n "$filename" ]; then
        echo "$arch:$filename" >> "$INDEX_FILE"
    fi
done
echo "Index written to: $INDEX_FILE"
cat "$INDEX_FILE"

echo ""
echo "========================================="
echo "  BUILD COMPLETE"
echo "  Packages: $BASE_DIR/dist/packages/"
echo "========================================="
ls -lh "$BASE_DIR/dist/packages/"*.ipk
