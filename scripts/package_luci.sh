#!/bin/bash

# Packaging script for luci-app-obhod
# Manual .ipk creation

VERSION="0.1.0"
RELEASE="1"
ARCH="all"

BUILD_DIR="/tmp/luci_build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

# 1. Prepare Data
echo "Preparing data for LuCI..."
LUCI_DIR="$BUILD_DIR/data/usr/lib/lua/luci"
mkdir -p "$LUCI_DIR/controller"
mkdir -p "$LUCI_DIR/view/obhod"
mkdir -p "$LUCI_DIR/i18n"
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"

# Copy source files
cp "/root/Obhod project/luci-app-obhod/luasrc/controller/obhod.lua" "$LUCI_DIR/controller/"
cp "/root/Obhod project/luci-app-obhod/luasrc/obhod_api.lua" "$BUILD_DIR/data/usr/lib/lua/"

# Copy JS main app
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
cp "/root/Obhod project/luci-app-obhod/htdocs/luci-static/resources/view/obhod/main.js" "$BUILD_DIR/data/www/luci-static/resources/view/obhod/main.js"

# Copy Meta and ACL
cp "/root/Obhod project/luci-app-obhod/root/usr/share/luci/menu.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/rpcd/acl.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"

# Placeholder for compiled translations (LMO)
# In real SDK, this is done by po2lmo. 
# We copy the source .po as a fallback or for the user to compile.
mkdir -p "$BUILD_DIR/data/usr/share/luci/i18n"
cp "/root/Obhod project/luci-app-obhod/po/ru/obhod.po" "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.po"

# 2. Prepare Control
cat <<EOF > "$BUILD_DIR/control/control"
Package: luci-app-obhod
Version: $VERSION-$RELEASE
Depends: obhod, curl
Section: luci
Architecture: $ARCH
Maintainer: Obhod Team
Description: LuCI support for Obhod VPN
EOF

# 3. Create IPK
echo "Creating .ipk..."
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
tar -czf "/root/Obhod project/dist/luci-app-obhod_${VERSION}-${RELEASE}_${ARCH}.ipk" debian-binary data.tar.gz control.tar.gz

echo "Done: luci-app-obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
