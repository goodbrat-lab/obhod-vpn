#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL - ARCH SPECIFIC
VERSION="0.3.0"
RELEASE="1"

ARCH=$1
if [ -z "$ARCH" ]; then
    echo "Usage: $0 <arch>"
    echo "Common architectures: mipsle_softfloat, mips_softfloat, arm64, arm_v7, amd64"
    exit 1
fi

BUILD_DIR="/tmp/obhod_build_$ARCH"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

# 1. Shell Backend Files
mkdir -p "$BUILD_DIR/data/usr/bin"
mkdir -p "$BUILD_DIR/data/usr/lib/obhod"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/etc/uci-defaults"

# Copy core files
cp "/root/Obhod project/obhod-core/files/usr/bin/obhod" "$BUILD_DIR/data/usr/bin/obhod"
chmod +x "$BUILD_DIR/data/usr/bin/obhod"

# Copy Go binary for specific architecture
GO_BINARY="/root/Obhod project/dist/binaries/obhoud_linux_$ARCH"
if [ ! -f "$GO_BINARY" ]; then
    echo "Error: Go binary not found for $ARCH at $GO_BINARY"
    exit 1
fi
cp "$GO_BINARY" "$BUILD_DIR/data/usr/bin/obhoud"
chmod +x "$BUILD_DIR/data/usr/bin/obhoud"

# Copy libraries
cp "/root/Obhod project/obhod-core/files/usr/lib/"* "$BUILD_DIR/data/usr/lib/obhod/"
rm -f "$BUILD_DIR/data/usr/lib/obhod/obhod" 

cp "/root/Obhod project/obhod-core/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
cp "/root/Obhod project/obhod-core/files/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"

# 2. LuCI Files
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
cp -r "/root/Obhod project/luci-app-obhod/htdocs/luci-static/resources/view/obhod/"* "$BUILD_DIR/data/www/luci-static/resources/view/obhod/"

# Menu and ACL
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/luci/menu.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/rpcd/acl.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"
cp "/root/Obhod project/luci-app-obhod/root/etc/uci-defaults/50_luci-obhod" "$BUILD_DIR/data/etc/uci-defaults/"

# Localization (Compile PO to LMO and install to both paths for compatibility)
mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci/i18n"
mkdir -p "$BUILD_DIR/data/usr/share/luci/i18n"
python3 "/root/Obhod project/scripts/compile_lmo.py" \
    "/root/Obhod project/luci-app-obhod/po/ru/obhod.po" \
    "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo"
cp "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo" "$BUILD_DIR/data/usr/lib/lua/luci/i18n/obhod.ru.lmo"

# 3. Control & Post-Install Script
# Map OpenWrt arch names if needed (this ARCH is our binary suffix)
PKG_ARCH=$(echo $ARCH | cut -d'_' -f1)

cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, jq, bind-dig, luci-base, luci-compat, rpcd-mod-file, coreutils-base64
Section: net
Architecture: $PKG_ARCH
Maintainer: Obhod Team
Description: Obhod VPN with Go Core ($ARCH)
EOF

cat <<EOF > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "\${IPKG_INSTROOT}" ] && exit 0
rm -rf /tmp/luci-indexcache*
rm -rf /tmp/luci-modulecache/
/etc/init.d/rpcd restart
/etc/init.d/uhttpd restart
/etc/init.d/obhod enable
exit 0
EOF
chmod +x "$BUILD_DIR/control/postinst"

# 4. Assembly
mkdir -p "/root/Obhod project/dist/packages"
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
tar -czf "/root/Obhod project/dist/packages/obhod_${VERSION}-${RELEASE}_${ARCH}.ipk" debian-binary data.tar.gz control.tar.gz

echo "Built: obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
