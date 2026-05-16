#!/bin/sh

# Universal Installer for Obhod VPN v0.3.1-final
# Clean state, dependency handling, and architecture detection.

set -e

REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="0.3.1-1"
LUCI_PKG="luci-app-obhod_${VERSION}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Clean Installer v0.3.1-3        "
echo "=================================================="

# 1. Complete Cleanup
echo "Cleaning up previous installations..."
/etc/init.d/obhod stop 2>/dev/null || true
opkg remove obhod luci-app-obhod --force-removal-of-dependent-packages 2>/dev/null || true
# We don't remove sing-box here to avoid breaking other services, but we will try to upgrade it.

# 2. Detect Architecture
echo "Detecting system architecture..."
ARCH_RAW=$(opkg print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')
if [ -z "$ARCH_RAW" ]; then ARCH_RAW=$(uname -m); fi

case "$ARCH_RAW" in
    aarch64*|arm64*) ARCH="arm64" ;;
    armv7*|arm_v7*|arm_cortex-a7*|arm_cortex-a9*|arm_cortex-a15*) ARCH="arm_v7" ;;
    mips_24kc*|mips_softfloat*|mips_mips32*) ARCH="mips_softfloat" ;;
    mipsel_24kc*|mipsle_softfloat*|mipsel_74kc*|mipsel_24k*) ARCH="mipsle_softfloat" ;;
    x86_64*|amd64*) ARCH="amd64" ;;
    *) echo "Error: Unsupported architecture '$ARCH_RAW'."; exit 1 ;;
esac

CORE_PKG="obhod_${VERSION}_${ARCH}.ipk"
echo "Architecture: $ARCH_RAW"

# 3. Update and Install Dependencies
echo "Updating package lists..."
opkg update

echo "Installing/Upgrading core dependencies..."
# We try to install full sing-box. If space is tight, opkg will tell us.
opkg install jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle ca-certificates
opkg install sing-box || {
    echo "--------------------------------------------------"
    echo "WARNING: Full 'sing-box' could not be installed."
    echo "Probably not enough space in /overlay (needs ~40MB)."
    echo "Trying to install 'sing-box-tiny' (~2MB) instead..."
    echo "--------------------------------------------------"
    opkg install sing-box-tiny
}

# 4. Download and Install Obhod (Explicitly from file)
cd /tmp
echo "Downloading latest Obhod packages..."
rm -f /tmp/obhod_new.ipk /tmp/luci_obhod_new.ipk

wget -q -O "obhod_new.ipk" "$REPO_URL/$CORE_PKG"
wget -q -O "luci_obhod_new.ipk" "$REPO_URL/$LUCI_PKG"

echo "Installing Obhod core from local file..."
opkg install "/tmp/obhod_new.ipk" --force-reinstall --force-overwrite

echo "Installing Obhod LuCI app from local file..."
opkg install "/tmp/luci_obhod_new.ipk" --force-reinstall --force-overwrite

# 5. Cleanup and Finish
rm -f "obhod_new.ipk" "luci_obhod_new.ipk"

echo "Finalizing configuration..."
/etc/init.d/rpcd restart 2>/dev/null || true
/etc/init.d/uhttpd restart 2>/dev/null || true

if [ -f "/usr/bin/obhod" ]; then
    echo "=================================================="
    echo "           INSTALLATION SUCCESSFUL!               "
    echo "=================================================="
    echo "Obhod Version: $(/usr/bin/obhod show_version 2>/dev/null || echo $VERSION)"
    echo "Sing-box Version: $(sing-box version | head -n1 | awk '{print $3}' 2>/dev/null)"
    echo "=================================================="
else
    echo "Error: Installation failed. Check opkg output above."
    exit 1
fi
