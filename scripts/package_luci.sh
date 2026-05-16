#!/bin/bash

# Packaging script for luci-app-obhod (standalone LuCI package)
# Usage: ./scripts/package_luci.sh

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

VERSION=$(grep "OBHOD_VERSION=" "$BASE_DIR/obhod-core/files/usr/lib/constants.sh" | cut -d'"' -f2)
RELEASE="1"
ARCH="all"

BUILD_DIR="/tmp/luci_obhod_build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

echo "Packaging luci-app-obhod $VERSION-$RELEASE..."

LUCI_SRC="$BASE_DIR/luci-app-obhod"

# --- Data ---
# JS-based LuCI views (modern ucode-style)
LUCI_VIEW_SRC="$LUCI_SRC/htdocs/luci-static/resources/view/obhod"
if [ -d "$LUCI_VIEW_SRC" ]; then
    mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
    cp -r "$LUCI_VIEW_SRC/"* "$BUILD_DIR/data/www/luci-static/resources/view/obhod/"
fi

# Lua controller (legacy LuCI compat)
LUA_CTRL="$LUCI_SRC/root/usr/lib/lua/luci"
if [ -d "$LUA_CTRL" ]; then
    mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci"
    cp -r "$LUA_CTRL/"* "$BUILD_DIR/data/usr/lib/lua/luci/"
fi

# Menu and ACL
MENU_SRC="$LUCI_SRC/root/usr/share/luci/menu.d/luci-app-obhod.json"
ACL_SRC="$LUCI_SRC/root/usr/share/rpcd/acl.d/luci-app-obhod.json"
[ -f "$MENU_SRC" ] && { mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"; cp "$MENU_SRC" "$BUILD_DIR/data/usr/share/luci/menu.d/"; }
[ -f "$ACL_SRC"  ] && { mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"; cp "$ACL_SRC" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"; }

# UCI defaults
UCIDEF="$LUCI_SRC/root/etc/uci-defaults/50_luci-obhod"
[ -f "$UCIDEF" ] && { mkdir -p "$BUILD_DIR/data/etc/uci-defaults"; cp "$UCIDEF" "$BUILD_DIR/data/etc/uci-defaults/"; }

# Localization: compile .po → .lmo (dual install for compatibility)
PO_FILE="$LUCI_SRC/po/ru/obhod.po"
if [ -f "$PO_FILE" ]; then
    mkdir -p "$BUILD_DIR/data/usr/share/luci/i18n"
    mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci/i18n"
    python3 "$SCRIPTS_DIR/compile_lmo.py" \
        "$PO_FILE" \
        "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo"
    cp "$BUILD_DIR/data/usr/share/luci/i18n/obhod.ru.lmo" \
       "$BUILD_DIR/data/usr/lib/lua/luci/i18n/obhod.ru.lmo"
    echo "Compiled: obhod.ru.lmo"
else
    echo "Warning: PO file not found: $PO_FILE (skipping localization)"
fi

# --- Control ---
cat <<EOF > "$BUILD_DIR/control/control"
Package: luci-app-obhod
Version: $VERSION-$RELEASE
Depends: obhod, luci-base, luci-compat, rpcd-mod-file
Section: luci
Architecture: $ARCH
Maintainer: Obhod Team <obhod@itdog.info>
Description: LuCI web interface for Obhod VPN
EOF

cat <<'POSTINST' > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
rm -rf /tmp/luci-indexcache*
rm -rf /tmp/luci-modulecache/
/etc/init.d/rpcd restart 2>/dev/null
/etc/init.d/uhttpd restart 2>/dev/null
exit 0
POSTINST
chmod +x "$BUILD_DIR/control/postinst"

# --- Assembly ---
mkdir -p "$BASE_DIR/dist/packages"
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary

OUTPUT="$BASE_DIR/dist/packages/luci-app-obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
tar -czf "$OUTPUT" debian-binary data.tar.gz control.tar.gz

echo "Done: $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
rm -rf "$BUILD_DIR"
