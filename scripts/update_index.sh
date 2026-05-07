#!/bin/bash

# Script to generate OpenWrt Package index
cd "/root/Obhod project/dist"

# Remove old index files
rm -f Packages Packages.gz

echo "Generating Packages index..."

# For each IPK, extract control info and append to Packages file
for pkg in *.ipk; do
    # Get filename
    filename=$(basename "$pkg")
    # Extract control file from IPK
    tar -xOzf "$pkg" ./control.tar.gz | tar -xOzf - ./control >> Packages
    # Add Filename and Size
    echo "Filename: $filename" >> Packages
    echo "Size: $(stat -c%s "$pkg")" >> Packages
    echo "SHA256sum: $(sha256sum "$pkg" | cut -d' ' -f1)" >> Packages
    echo "" >> Packages
done

# Compress
gzip -c Packages > Packages.gz

echo "Done. Packages and Packages.gz updated."
