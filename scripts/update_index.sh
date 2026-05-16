#!/bin/bash

# Script to generate OpenWrt Package index
SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"
DIST_DIR="$BASE_DIR/dist/packages"

cd "$DIST_DIR" || exit 1

# Remove old index files
rm -f Packages Packages.gz

echo "Generating Packages index in $DIST_DIR..."

# For each IPK, extract control info and append to Packages file
for pkg in *.ipk; do
    [ -e "$pkg" ] || continue
    # Get filename
    filename=$(basename "$pkg")
    # Extract control file from IPK
    tar -xOzf "$pkg" control.tar.gz | tar -xOzf - ./control >> Packages
    # Add Filename and Size
    echo "Filename: $filename" >> Packages
    echo "Size: $(stat -c%s "$pkg")" >> Packages
    echo "SHA256sum: $(sha256sum "$pkg" | cut -d' ' -f1)" >> Packages
    echo "" >> Packages
done

# Compress
if [ -f Packages ]; then
    gzip -c Packages > Packages.gz
fi

echo "Done. Packages and Packages.gz updated."
