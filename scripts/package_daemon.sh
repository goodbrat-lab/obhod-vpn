#!/bin/bash

# Packaging script for Obhod
# Manual .ipk creation without OpenWrt SDK

VERSION="0.1.0"
RELEASE="1"
ARCH=$1
BINARY=$2

if [ -z "$ARCH" ] || [ -z "$BINARY" ]; then
    echo "Usage: $0 <arch> <binary_path>"
    exit 1
fi

BUILD_DIR="/tmp/obhod_build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

# 1. Prepare Data
echo "Preparing data for $ARCH..."
mkdir -p "$BUILD_DIR/data/usr/bin"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/var/run/obhod"

cp "$BINARY" "$BUILD_DIR/data/usr/bin/obhoud"
chmod +x "$BUILD_DIR/data/usr/bin/obhoud"
cp "/root/Obhod project/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
cp "/root/Obhod project/files/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"

# 2. Prepare Control
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl
Section: net
Architecture: $ARCH
Maintainer: Obhod Team
Description: Reliable selective VPN routing for OpenWrt
EOF

# 3. Create IPK
echo "Creating .ipk..."
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
tar -czf "/root/Obhod project/dist/obhod_${VERSION}-${RELEASE}_${ARCH}.ipk" debian-binary data.tar.gz control.tar.gz

echo "Done: obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
