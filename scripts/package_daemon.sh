#!/bin/bash

# Packaging script for the obhod daemon-only package (without LuCI)
# Used for minimal installations or when LuCI is not needed.
# Usage: ./scripts/package_daemon.sh <arch> <binary_path>

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"

VERSION="0.3.0"
RELEASE="1"
ARCH=$1
BINARY=$2

if [ -z "$ARCH" ] || [ -z "$BINARY" ]; then
    echo "Usage: $0 <arch> <binary_path>"
    echo "Example: $0 arm64 ./dist/binaries/obhoud_linux_arm64"
    exit 1
fi

if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found: $BINARY"
    exit 1
fi

BUILD_DIR="/tmp/obhod_daemon_build_$ARCH"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control"

echo "Packaging obhod daemon $VERSION-$RELEASE for $ARCH..."

# --- Data ---
mkdir -p "$BUILD_DIR/data/usr/bin"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/etc/config"

cp "$BINARY" "$BUILD_DIR/data/usr/bin/obhoud"
chmod +x "$BUILD_DIR/data/usr/bin/obhoud"

cp "$BASE_DIR/obhod-core/files/etc/init.d/obhod" "$BUILD_DIR/data/etc/init.d/obhod"
chmod +x "$BUILD_DIR/data/etc/init.d/obhod"
cp "$BASE_DIR/obhod-core/files/etc/config/obhod" "$BUILD_DIR/data/etc/config/obhod"

# --- Control ---
cat <<EOF > "$BUILD_DIR/control/control"
Package: obhod-daemon
Version: $VERSION-$RELEASE
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, jq, coreutils-base64
Section: net
Architecture: $ARCH
Maintainer: Obhod Team <obhod@itdog.info>
Description: Obhod VPN daemon (without LuCI UI)
EOF

cat <<'POSTINST' > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/obhod enable
exit 0
POSTINST
chmod +x "$BUILD_DIR/control/postinst"

cat <<'PRERM' > "$BUILD_DIR/control/prerm"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/obhod stop 2>/dev/null
/etc/init.d/obhod disable 2>/dev/null
grep -q "105 obhod" /etc/iproute2/rt_tables && sed -i "/105 obhod/d" /etc/iproute2/rt_tables
exit 0
PRERM
chmod +x "$BUILD_DIR/control/prerm"

# --- Assembly ---
mkdir -p "$BASE_DIR/dist"
cd "$BUILD_DIR/data" && tar -czf "../data.tar.gz" .
cd "$BUILD_DIR/control" && tar -czf "../control.tar.gz" .
cd "$BUILD_DIR"
echo "2.0" > debian-binary

OUTPUT="$BASE_DIR/dist/obhod-daemon_${VERSION}-${RELEASE}_${ARCH}.ipk"
tar -czf "$OUTPUT" debian-binary data.tar.gz control.tar.gz

echo "Done: $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
rm -rf "$BUILD_DIR"
