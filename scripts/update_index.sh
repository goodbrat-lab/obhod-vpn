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
    
    # Extract control file from IPK (Support both 'ar' and 'tar' formats)
    if ar t "$pkg" >/dev/null 2>&1; then
        ar p "$pkg" control.tar.gz | tar -xOzf - ./control 2>/dev/null || \
        ar p "$pkg" control.tar.gz | tar -xOzf - control >> Packages
    else
        tar -xOzf "$pkg" control.tar.gz 2>/dev/null | tar -xOzf - ./control 2>/dev/null || \
        tar -xOzf "$pkg" control.tar.gz 2>/dev/null | tar -xOzf - control >> Packages
    fi
    
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

# Generate standalone SHA256SUMS file
rm -f SHA256SUMS
for f in *.ipk *.tar.gz; do
    [ -e "$f" ] || continue
    sha256sum "$f" >> SHA256SUMS
done

echo "Done. Packages, Packages.gz and SHA256SUMS updated."
