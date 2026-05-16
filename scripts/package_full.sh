#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL - ARCH SPECIFIC
# Usage: ./scripts/package_full.sh <arch>
# Example: ./scripts/package_full.sh arm64

set -e

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

VERSION="0.3.1"
RELEASE="4"

ARCH=$1
if [ -z "$ARCH" ]; then
    echo "Usage: $0 <arch>"
    echo "Common architectures: mipsle_softfloat, mips_softfloat, arm64, arm_v7, amd64"
    exit 1
fi

echo "=== Packaging obhod $VERSION-$RELEASE for $ARCH ==="
echo "Base dir: $BASE_DIR"

BUILD_DIR="/tmp/obhod_build_$ARCH"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

# --- 1. Shell Backend Files ---
mkdir -p "$BUILD_DIR/data/usr/bin"
mkdir -p "$BUILD_DIR/data/usr/lib/obhod"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/etc/uci-defaults"

# Main bash script
cp "$BASE_DIR/obhod-core/files/usr/bin/obhod" "$BUILD_DIR/data/usr/bin/obhod"
chmod +x "$BUILD_DIR/data/usr/bin/obhod"

# Go binary for the specific architecture
GO_BINARY="$BASE_DIR/dist/binaries/obhoud_linux_$ARCH"
if [ ! -f "$GO_BINARY" ]; then
    echo "Error: Go binary not found for $ARCH at $GO_BINARY"
    echo "Run ./scripts/build_go.sh first."
    exit 1
fi
cp "$GO_BINARY" "$BUILD_DIR/data/usr/bin/obhoud"
chmod +x "$BUILD_DIR/data/usr/bin/obhoud"

# Libraries (all .sh and .jq files from usr/lib)
cp "$BASE_DIR/obhod-core/files/usr/lib/"* "$BUILD_DIR/data/usr/lib/obhod/"
chmod +x "$BUILD_DIR/data/usr/lib/obhod/"*.sh
# Remove any accidentally copied binaries from lib
rm -f "$BUILD_DIR/data/usr/lib/obhod/obhod"
rm -f "$BUILD_DIR/data/usr/lib/obhod/obhoud"

# Init script and config
cp "$BASE_DIR/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
cp "$BASE_DIR/files/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"
chmod 644 "$BUILD_DIR/data/etc/config/obhod"

# --- 2. LuCI Files ---
LUCI_VIEW_SRC="$BASE_DIR/luci-app-obhod/htdocs/luci-static/resources/view/obhod"
if [ -d "$LUCI_VIEW_SRC" ]; then
    mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
    cp -r "$LUCI_VIEW_SRC/"* "$BUILD_DIR/data/www/luci-static/resources/view/obhod/"
fi

# Menu, ACL and UCI defaults
MENU_SRC="$BASE_DIR/luci-app-obhod/root/usr/share/luci/menu.d/luci-app-obhod.json"
ACL_SRC="$BASE_DIR/luci-app-obhod/root/usr/share/rpcd/acl.d/luci-app-obhod.json"
UCIDEFAULTS_SRC="$BASE_DIR/luci-app-obhod/root/etc/uci-defaults/50_luci-obhod"

[ -f "$MENU_SRC" ] && { mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"; cp "$MENU_SRC" "$BUILD_DIR/data/usr/share/luci/menu.d/"; }
[ -f "$ACL_SRC"  ] && { mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"; cp "$ACL_SRC" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"; }
[ -f "$UCIDEFAULTS_SRC" ] && { cp "$UCIDEFAULTS_SRC" "$BUILD_DIR/data/etc/uci-defaults/"; chmod +x "$BUILD_DIR/data/etc/uci-defaults/50_luci-obhod"; }


# Localization: compile .po -> .lmo and install to both paths for max compatibility
PO_FILE="$BASE_DIR/luci-app-obhod/po/ru/obhod.po"
if [ -f "$PO_FILE" ]; then
    mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci/i18n"
    mkdir -p "$BUILD_DIR/data/usr/share/luci/i18n"
    python3 "$SCRIPTS_DIR/compile_lmo.py" \
        "$PO_FILE" \
        "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo"
    cp "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo" \
       "$BUILD_DIR/data/usr/lib/lua/luci/i18n/obhod.ru.lmo"
fi

# --- 3. Control & Post-Install Script ---
# Map our build arch suffix to opkg architecture field
PKG_ARCH=$(echo "$ARCH" | cut -d'_' -f1)

cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, jq, bind-dig, luci-base, luci-compat, rpcd-mod-file, coreutils-base64
Section: net
Architecture: $PKG_ARCH
Maintainer: Obhod Team <obhod@itdog.info>
Description: Obhod VPN - selective routing for OpenWrt ($ARCH)
EOF

cat <<'POSTINST' > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
# Clear LuCI caches so the new menu appears immediately
rm -rf /tmp/luci-indexcache*
rm -rf /tmp/luci-modulecache/
/etc/init.d/rpcd restart 2>/dev/null
/etc/init.d/uhttpd restart 2>/dev/null
# Enable and start obhod service
/etc/init.d/obhod enable
echo "Obhod installed. Configure it in LuCI under Services -> Obhod, then run:"
echo "  /etc/init.d/obhod start"
exit 0
POSTINST
chmod +x "$BUILD_DIR/control/postinst"

cat <<'PRERM' > "$BUILD_DIR/control/prerm"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/obhod stop 2>/dev/null
/etc/init.d/obhod disable 2>/dev/null
# Clean up routing table entry
grep -q "105 obhod" /etc/iproute2/rt_tables && sed -i "/105 obhod/d" /etc/iproute2/rt_tables
exit 0
PRERM
chmod +x "$BUILD_DIR/control/prerm"

# --- 4. Assembly ---
DIST_PKG_DIR="$BASE_DIR/dist/packages"
mkdir -p "$DIST_PKG_DIR"

cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
OUTPUT_FILE="$DIST_PKG_DIR/obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
tar -czf "$OUTPUT_FILE" debian-binary data.tar.gz control.tar.gz

echo "Built: $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" | cut -f1))"

# Cleanup
rm -rf "$BUILD_DIR"
