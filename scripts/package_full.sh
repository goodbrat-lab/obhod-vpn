#!/bin/bash

# ULTIMATE ROBUST PACKAGING FOR OBHOD FULL
VERSION="0.1.1"
RELEASE="1"
ARCH=$1
BINARY=$2

if [ -z "$ARCH" ] || [ -z "$BINARY" ]; then
    echo "Usage: $0 <arch> <binary_path>"
    exit 1
fi

BUILD_DIR="/tmp/obhod_v011_build"
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
cp "/root/Obhod project/luci-app-obhod/root/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"

# 2. LuCI Files (Stable CBI)
LUCI_BASE="$BUILD_DIR/data/usr/lib/lua/luci"
mkdir -p "$LUCI_BASE/controller"
mkdir -p "$LUCI_BASE/model/cbi/obhod"

cp "/root/Obhod project/luci-app-obhod/luasrc/controller/obhod.lua" "$LUCI_BASE/controller/"
cp "/root/Obhod project/luci-app-obhod/luasrc/model/cbi/obhod/"*.lua "$LUCI_BASE/model/cbi/obhod/"
cp "/root/Obhod project/luci-app-obhod/luasrc/obhod_api.lua" "$BUILD_DIR/data/usr/lib/lua/"

# ACL
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
cat <<EOF > "$BUILD_DIR/data/usr/share/rpcd/acl.d/luci-app-obhod.json"
{
	"luci-app-obhod": {
		"description": "Grant access to Obhod VPN",
		"read": {
			"cgi-bin": [ "luci/admin/services/obhod/*" ],
			"uci": [ "obhod" ]
		},
		"write": {
			"cgi-bin": [ "luci/admin/services/obhod/*" ],
			"uci": [ "obhod" ]
		}
	}
}
EOF

# 3. Control & Post-Install Script
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod-full
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, luci-lua-runtime, luci-base, luci-compat
Section: net
Architecture: $ARCH
Maintainer: Obhod Team
Description: Obhod VPN Full Package (Final Ultra-Stable)
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
