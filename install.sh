#!/bin/sh

# Obhod VPN One-Line Installer (Architecture-Aware) v0.3.0
# Usage: sh <(wget -qO- "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh?$(date +%s)")

set -e

echo "--- Obhod VPN Installer v0.3.0 (with Go Core) ---"

# 1. Detect Architecture
ARCH_OPENWRT=$(opkg print-architecture | head -n 1 | awk '{print $2}')
echo "Detected OpenWrt architecture: $ARCH_OPENWRT"

# Map OpenWrt architecture to our package suffix
case "$ARCH_OPENWRT" in
    "mipsel_24kc"|"mipsel_74kc"|"mipsle")
        ARCH_SUFFIX="mipsle_softfloat"
        ;;
    "mips_24kc"|"mips_74kc"|"mips")
        ARCH_SUFFIX="mips_softfloat"
        ;;
    "aarch64_generic"|"arm64")
        ARCH_SUFFIX="arm64"
        ;;
    "arm_cortex-a7_neon-vfpv4"|"arm_cortex-a15_neon-vfpv4"|"arm_v7")
        ARCH_SUFFIX="arm_v7"
        ;;
    "x86_64"|"amd64")
        ARCH_SUFFIX="amd64"
        ;;
    *)
        echo "Warning: Unknown architecture $ARCH_OPENWRT. Attempting universal/mipsle fallback..."
        ARCH_SUFFIX="mipsle_softfloat"
        ;;
esac

echo "Selected binary variant: $ARCH_SUFFIX"

# 2. Base URL (pointing to GitHub dist/packages)
BASE_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/packages"

# 3. Download the specific package
# Note: In production, you would push the built packages to the GitHub repo.
# Filename pattern: obhod_0.3.0-1_<ARCH_SUFFIX>.ipk
PKG_NAME="obhod_0.3.0-1_${ARCH_SUFFIX}.ipk"
PKG_URL="${BASE_URL}/${PKG_NAME}"

echo "Downloading package: $PKG_NAME"
wget --no-check-certificate -O /tmp/obhod.ipk "$PKG_URL"

if [ ! -s /tmp/obhod.ipk ]; then
    echo "Error: Download failed or file is empty. Check your internet connection or GitHub repo state."
    exit 1
fi

# 4. Install
echo "Installing Obhod VPN..."
opkg update
# Force dependencies for 24.xx+
opkg install --force-reinstall /tmp/obhod.ipk

echo "--- Installation Complete ---"
echo "Please refresh your browser (Ctrl+F5). Menu: Services -> Obhod"
echo "Watchdog (Go core) is now active."
