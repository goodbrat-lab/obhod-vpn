#!/bin/bash

# BUILD EVERYTHING: Go binaries + .ipk packages
# Usage: ./scripts/build_all.sh

BASE_DIR="/root/Obhod project"
SCRIPTS_DIR="$BASE_DIR/scripts"

echo "Step 1: Building Go binaries..."
bash "$SCRIPTS_DIR/build_go.sh"

echo "Step 2: Building IPK packages..."
# List of architectures to package (must match build_go.sh outputs)
ARCHS=("mipsle_softfloat" "mips_softfloat" "arm64" "arm_v7" "amd64")

for arch in "${ARCHS[@]}"; do
    bash "$SCRIPTS_DIR/package_full.sh" "$arch"
done

echo "Step 3: Creating/Updating dist index (optional)..."
# You might want a simple text file that maps architectures to ipk filenames
INDEX_FILE="$BASE_DIR/dist/packages/index.txt"
rm -f "$INDEX_FILE"
for arch in "${ARCHS[@]}"; do
    filename=$(ls "$BASE_DIR/dist/packages/" | grep "_${arch}.ipk")
    if [ -n "$filename" ]; then
        echo "$arch:$filename" >> "$INDEX_FILE"
    fi
done

echo "=== BUILD COMPLETE ==="
echo "Packages are in dist/packages/"
