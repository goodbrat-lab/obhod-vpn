#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL - VUE ORIGINAL UI RESTORED
VERSION="0.1.4"
RELEASE="1"
ARCH=$1
BINARY=$2

if [ -z "$ARCH" ] || [ -z "$BINARY" ]; then
    echo "Usage: $0 <arch> <binary_path>"
    exit 1
fi

BUILD_DIR="/tmp/obhod_v014_build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

# 1. Daemon & System Files
mkdir -p "$BUILD_DIR/data/usr/bin"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/var/run/obhod"

cp "$BINARY" "$BUILD_DIR/data/usr/bin/obhoud"
chmod +x "$BUILD_DIR/data/usr/bin/obhoud"
cp "/root/Obhod project/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
# Use original podkop-style config but named obhod
cat <<EOF > "$BUILD_DIR/data/etc/config/obhod"
config settings 'settings'
        option dns_type 'udp'
        option dns_server '8.8.8.8'
        option fwmark '255'
        option dns_port '15353'
        option tproxy_port '11080'
        option log_level 'info'

config section 'main'
        option connection_type 'proxy'
        option proxy_config_type 'url'
        option proxy_string ''
        list user_domains 'google.com'
EOF

# 2. LuCI Files (Original Vue.js UI from Podkop)
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/obhod"
cp -r "/root/Obhod project/luci-app-obhod/htdocs/luci-static/resources/view/obhod/"* "$BUILD_DIR/data/www/luci-static/resources/view/obhod/"

# Menu and ACL
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/luci/menu.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp "/root/Obhod project/luci-app-obhod/root/usr/share/rpcd/acl.d/luci-app-obhod.json" "$BUILD_DIR/data/usr/share/rpcd/acl.d/"

# 3. Control & Post-Install Script
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod-full
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, luci-lua-runtime, luci-base, luci-compat
Section: net
Architecture: $ARCH
Maintainer: Obhod Team
Description: Obhod VPN Full Package (Original Vue UI)
EOF

cat <<EOF > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "\${IPKG_INSTROOT}" ] && exit 0
rm -rf /tmp/luci-indexcache*
rm -rf /tmp/luci-modulecache/
/etc/init.d/rpcd restart
/etc/init.d/uhttpd restart
exit 0
EOF
chmod +x "$BUILD_DIR/control/postinst"

# 4. Assembly
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary
tar -czf "/root/Obhod project/dist/obhod-full_${VERSION}-${RELEASE}_${ARCH}.ipk" debian-binary data.tar.gz control.tar.gz

echo "Built FINAL package: obhod-full_${VERSION}-${RELEASE}_${ARCH}.ipk"
