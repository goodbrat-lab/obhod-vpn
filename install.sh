#!/bin/sh

# Universal Installer for Obhod VPN v0.3.1
# High compatibility with OpenWrt standard architecture names.

set -e

REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="0.3.1-1"
LUCI_PKG="luci-app-obhod_${VERSION}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v0.3.1      "
echo "=================================================="

# 1. Precise Cleanup
echo "Preparing system for clean installation..."
/etc/init.d/obhod stop 2>/dev/null || true
opkg remove obhod luci-app-obhod --force-removal-of-dependent-packages 2>/dev/null || true

# 2. Precise Architecture Detection
echo "Detecting architecture..."
# Pick the highest priority architecture from opkg that isn't 'all' or 'noarch'
ARCH=$(opkg print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')

if [ -z "$ARCH" ]; then
    echo "Warning: opkg detection failed, falling back to uname."
    ARCH=$(uname -m)
fi

CORE_PKG="obhod_${VERSION}_${ARCH}.ipk"
echo "Architecture detected: $ARCH"
echo "Targeting package: $CORE_PKG"

# 3. Full Update and Dependency handling
echo "Updating package lists..."
opkg update || echo "Warning: opkg update failed."

echo "Installing/Upgrading dependencies..."
# Install everything except sing-box first
opkg install jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle ca-certificates || true

# Try to install full sing-box. We use --force-overwrite to be safe.
opkg install sing-box || {
    echo "Space is low. Installing 'sing-box-tiny' instead..."
    opkg install sing-box-tiny || true
}

# 4. Download and Install Obhod (EXACT FILENAMES)
cd /tmp
echo "Downloading Obhod..."
rm -f /tmp/obhod.ipk /tmp/luci.ipk

if ! wget -q -O "obhod.ipk" "$REPO_URL/$CORE_PKG"; then
    echo "Error: Package $CORE_PKG not found for your architecture."
    echo "Visit $REPO_URL to check available packages."
    exit 1
fi
wget -q -O "luci.ipk" "$REPO_URL/$LUCI_PKG"

echo "Installing Obhod packages..."
opkg install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
opkg install "/tmp/luci.ipk" --force-reinstall --force-overwrite

# 5. Cleanup and Finish
rm -f "obhod.ipk" "luci.ipk"

echo "Finalizing..."
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
    echo "Error: Installation failed. Binary /usr/bin/obhod is missing."
    exit 1
fi
