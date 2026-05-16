#!/bin/sh

# Universal Installer for Obhod VPN v0.3.1
# Automatically detects architecture, handles dependencies, and installs from GitHub

set -e

REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="0.3.1-1"
LUCI_PKG="luci-app-obhod_${VERSION}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v0.3.1      "
echo "=================================================="

# 1. Detect Architecture
echo "Detecting system architecture..."

# Get the most specific architecture (highest priority in opkg)
# We exclude 'all', 'noarch', and pick the last one which usually has the highest priority
ARCH_RAW=$(opkg print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')

if [ -z "$ARCH_RAW" ]; then
    echo "Warning: Could not detect architecture via opkg. Falling back to uname."
    ARCH_RAW=$(uname -m)
fi

case "$ARCH_RAW" in
    aarch64*|arm64*)
        ARCH="arm64"
        ;;
    armv7*|arm_v7*|arm_cortex-a7*|arm_cortex-a9*|arm_cortex-a15*)
        ARCH="arm_v7"
        ;;
    mips_24kc*|mips_softfloat*|mips_mips32*)
        ARCH="mips_softfloat"
        ;;
    mipsel_24kc*|mipsle_softfloat*|mipsel_74kc*|mipsel_24k*)
        ARCH="mipsle_softfloat"
        ;;
    x86_64*|amd64*)
        ARCH="amd64"
        ;;
    *)
        echo "Error: Unsupported architecture '$ARCH_RAW'."
        echo "Please install manually from: $REPO_URL"
        exit 1
        ;;
esac

CORE_PKG="obhod_${VERSION}_${ARCH}.ipk"
echo "System architecture: $ARCH_RAW"
echo "Selected package: $CORE_PKG"

# 2. Update opkg
echo "Updating package lists..."
opkg update || echo "Warning: opkg update failed, attempting to proceed anyway..."

# 3. Install core dependencies
echo "Installing dependencies..."
# Added ca-certificates and ca-bundle to fix unresolved dependency issues seen in logs
# We use --force-overwrite just in case of conflicts with existing files
opkg install sing-box jq curl nftables coreutils-base64 kmod-nft-tproxy ca-certificates ca-bundle || {
    echo "Warning: Some dependencies could not be installed automatically."
    echo "Attempting to install sing-box-tiny if sing-box is missing..."
    opkg install sing-box-tiny jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle || true
}

# 4. Download and Install Obhod
cd /tmp

echo "Downloading Obhod core..."
if ! wget -q -O "obhod.ipk" "$REPO_URL/$CORE_PKG"; then
    echo "Error: Failed to download $CORE_PKG from $REPO_URL"
    exit 1
fi

echo "Downloading Obhod LuCI app..."
if ! wget -q -O "luci-obhod.ipk" "$REPO_URL/$LUCI_PKG"; then
    echo "Error: Failed to download $LUCI_PKG"
    exit 1
fi

echo "Installing Obhod packages..."
# We use --force-reinstall to ensure we get the latest version if 0.3.0 was present
opkg install "obhod.ipk" "luci-obhod.ipk" --force-reinstall

# 5. Cleanup
rm -f "obhod.ipk" "luci-obhod.ipk"

# 6. Finalize
echo "Applying final configurations..."
/etc/init.d/rpcd restart 2>/dev/null || true
/etc/init.d/uhttpd restart 2>/dev/null || true

# Check if obhod binary is present
if [ -f "/usr/bin/obhod" ]; then
    echo "=================================================="
    echo "           INSTALLATION SUCCESSFUL!               "
    echo "=================================================="
    echo "Version: $(/usr/bin/obhod show_version 2>/dev/null || echo 0.3.1)"
    echo "Access Obhod VPN in LuCI: Services -> Obhod VPN"
    echo "Or use command: /usr/bin/obhod start"
    echo "=================================================="
else
    echo "Error: /usr/bin/obhod not found after installation. Check logs above."
    exit 1
fi
