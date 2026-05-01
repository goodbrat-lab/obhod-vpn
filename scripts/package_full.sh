#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL - VUE UI + ORIGINAL SHELL BACKEND
VERSION="0.2.0"
RELEASE="3"

BUILD_DIR="/tmp/obhod_v020_build"
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
cp "/root/Obhod project/obhod-core/files/usr/bin/obhod-watchdog" "$BUILD_DIR/data/usr/bin/obhod-watchdog" 2>/dev/null || true
chmod +x "$BUILD_DIR/data/usr/bin/obhod-watchdog" 2>/dev/null || true
cp -r "/root/Obhod project/obhod-core/files/usr/lib/obhod/"* "$BUILD_DIR/data/usr/lib/obhod/" 2>/dev/null || true
cp "/root/Obhod project/obhod-core/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
cp "/root/Obhod project/obhod-core/files/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"

# 2. LuCI Files (Original Vue.js UI)
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
cp -r "/root/Obhod project/luci-app-obhod/htdocs/luci-static/resources/view/obhod/"* "$BUILD_DIR/data/www/luci-static/resources/view/obhod/"

# Menu and ACL
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/luci/menu.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/rpcd/acl.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"

# 3. Control & Post-Install Script
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, luci-base, luci-compat
Section: net
Architecture: all
Maintainer: Obhod Team
Description: Obhod VPN (Original Podkop functionality restored)
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
mkdir -p "/root/Obhod project/dist"
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
tar -czf "/root/Obhod project/dist/obhod_${VERSION}-${RELEASE}_all.ipk" debian-binary data.tar.gz control.tar.gz

echo "Built FINAL package: obhod_${VERSION}-${RELEASE}_all.ipk"
