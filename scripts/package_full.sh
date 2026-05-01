#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL - VERSION 0.2.0-6
# Fixes missing library paths and ensures all dependencies.

VERSION="0.2.0"
RELEASE="6"

BUILD_DIR="/tmp/obhod_v0206_build"
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
cp "/root/Obhod project/obhod-core/files/usr/bin/obhod-watchdog" "$BUILD_DIR/data/usr/bin/obhod-watchdog"
chmod +x "$BUILD_DIR/data/usr/bin/obhod-watchdog"

# CRITICAL: Copy libraries to /usr/lib/obhod/ (Fixing the previous missing path bug)
cp "/root/Obhod project/obhod-core/files/usr/lib/"* "$BUILD_DIR/data/usr/lib/obhod/"
rm -f "$BUILD_DIR/data/usr/lib/obhod/obhod" # Remove accidental copy of binary if any

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
cp "/root/Obhod project/luci-app-obhod/root/etc/uci-defaults/50_luci-obhod" "$BUILD_DIR/data/etc/uci-defaults/"

# 3. Control & Post-Install Script
# Added rpcd-mod-file and coreutils-base64 to dependencies
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, jq, bind-dig, luci-base, luci-compat, rpcd-mod-file, coreutils-base64
Section: net
Architecture: all
Maintainer: Obhod Team
Description: Obhod VPN (Fixed Library Paths and RPC access)
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
tar -czf "/root/Obhod project/dist/obhod_0.2.0-6_all.ipk" debian-binary data.tar.gz control.tar.gz

echo "Successfully built obhod_0.2.0-6_all.ipk"
