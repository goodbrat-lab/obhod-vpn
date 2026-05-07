#!/bin/sh

# Obhod VPN Installer v0.3.0
# Usage: sh <(wget -qO- "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh")

set -e

REPO_RAW="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main"
VERSION="0.3.0-1"

echo "=========================================="
echo "  Obhod VPN Installer v${VERSION}"
echo "=========================================="

# 1. Detect architecture (opkg canonical name)
ARCH=$(opkg print-architecture | awk '{print $2}' | grep -v "all" | head -n1)
echo "Detected architecture: $ARCH"

# 2. Map to package name
case "$ARCH" in
    mipsel_24kc|mipsel_74kc|mipsel_mips32)
        PKG_ARCH="mipsel_24kc" ;;
    mips_24kc|mips_74kc|mips_mips32)
        PKG_ARCH="mips_24kc" ;;
    aarch64_cortex-a53|aarch64_cortex-a72|aarch64_generic)
        PKG_ARCH="aarch64_cortex-a53" ;;
    arm_cortex-a7_neon-vfpv4|arm_cortex-a15_neon-vfpv4|arm_cortex-a9)
        PKG_ARCH="arm_cortex-a7_neon-vfpv4" ;;
    x86_64)
        PKG_ARCH="x86_64" ;;
    *)
        echo ""
        echo "WARNING: Unknown architecture '$ARCH'"
        echo "Available packages: mipsel_24kc, mips_24kc, aarch64_cortex-a53,"
        echo "                    arm_cortex-a7_neon-vfpv4, x86_64"
        echo ""
        echo "Set PKG_ARCH manually and re-run:"
        echo "  PKG_ARCH=mipsel_24kc sh <(wget -qO- $REPO_RAW/install.sh)"
        exit 1 ;;
esac

PKG_NAME="obhod_${VERSION}_${PKG_ARCH}.ipk"
PKG_URL="${REPO_RAW}/dist/packages/${PKG_NAME}"

echo "Package: $PKG_NAME"

# 3. Download
echo "Downloading..."
wget --no-check-certificate -q --show-progress -O /tmp/obhod.ipk "$PKG_URL" 2>/dev/null || \
wget --no-check-certificate -O /tmp/obhod.ipk "$PKG_URL"

if [ ! -s /tmp/obhod.ipk ]; then
    echo "ERROR: Download failed or file is empty!"
    echo "URL: $PKG_URL"
    exit 1
fi

echo "Downloaded: $(wc -c < /tmp/obhod.ipk) bytes"

# 4. Check dependencies
echo "Updating package lists..."
opkg update 2>/dev/null || true

# Install sing-box if not present
if ! opkg list-installed | grep -q "^sing-box "; then
    echo "Installing sing-box..."
    opkg install sing-box || echo "WARNING: could not install sing-box, please install manually"
fi

# 5. Install Obhod
echo "Installing Obhod..."
opkg install --force-reinstall /tmp/obhod.ipk
rm -f /tmp/obhod.ipk

# 6. Post-install
echo ""
echo "=========================================="
echo "  Obhod installed successfully!"
echo "=========================================="
echo ""
echo "  Next steps:"
echo "  1. Edit /etc/config/obhod — set your VPN proxy_string"
echo "  2. Enable: /etc/init.d/obhod enable"
echo "  3. Start:  /etc/init.d/obhod start"
echo "  4. Check logs: logread | grep obhod"
echo ""
