#!/bin/sh

# Universal Installer for Obhod VPN v0.3.1
# High compatibility with OpenWrt standard architecture names and OpenWrt 25.xx (OneWrt).

set -e

# 1. Environment and Debugging
export PATH=/bin:/sbin:/usr/bin:/usr/sbin:$PATH
REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="0.3.1-1"
LUCI_PKG="luci-app-obhod_${VERSION}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v0.3.1      "
echo "=================================================="
echo "System Debug Info:"
echo "  PATH: $PATH"
echo "  Commands: $(command -v opkg || echo 'opkg not found'), $(command -v apk || echo 'apk not found')"
echo "  Uname: $(uname -a)"
echo "=================================================="

# 2. Package Manager Detection
OPKG_CMD=$(command -v opkg || echo "/bin/opkg")
if [ ! -x "$OPKG_CMD" ]; then
    echo "Warning: opkg not found at $OPKG_CMD."
    if command -v apk >/dev/null 2>&1; then
        echo "Detected 'apk' package manager (OpenWrt 25+ / OneWrt)."
        echo "Note: Obhod currently provides .ipk packages. Installation via apk is experimental."
        # We might need a different install path for apk in the future.
    else
        echo "CRITICAL ERROR: No package manager found (opkg/apk)."
        echo "Please install Obhod manually by downloading the packages:"
        echo "1. Download Luci: $REPO_URL/$LUCI_PKG"
        echo "2. Download Core: $REPO_URL/obhod_${VERSION}_[ARCH].ipk"
        exit 1
    fi
fi

# 3. Precise Architecture Detection
echo "Detecting architecture..."
ARCH=""

# Try opkg first
if [ -x "$OPKG_CMD" ]; then
    ARCH=$($OPKG_CMD print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')
fi

if [ -z "$ARCH" ]; then
    echo "Warning: opkg detection failed, falling back to uname -m mapping."
    UNAME_M=$(uname -m)
    case "$UNAME_M" in
        aarch64) ARCH="aarch64_cortex-a53" ;;
        armv7l)  ARCH="arm_cortex-a7_neon-vfpv4" ;;
        mips)    ARCH="mips_24kc" ;;
        mipsel)  ARCH="mipsel_24kc" ;;
        x86_64)  ARCH="x86_64" ;;
        *)
            echo "Error: Unknown architecture '$UNAME_M'."
            echo "Available architectures in this repo:"
            echo "  - aarch64_cortex-a53 (arm64)"
            echo "  - arm_cortex-a7_neon-vfpv4 (armv7)"
            echo "  - mips_24kc (mips be)"
            echo "  - mipsel_24kc (mips le)"
            echo "  - x86_64 (amd64)"
            exit 1
            ;;
    esac
fi

CORE_PKG="obhod_${VERSION}_${ARCH}.ipk"
echo "Architecture detected: $ARCH"
echo "Targeting package: $CORE_PKG"

# 4. Service Cleanup
echo "Preparing system for clean installation..."
/etc/init.d/obhod stop 2>/dev/null || true
if [ -x "$OPKG_CMD" ]; then
    $OPKG_CMD remove obhod luci-app-obhod --force-removal-of-dependent-packages 2>/dev/null || true
fi

# 5. Dependency handling
echo "Updating package lists..."
if [ -x "$OPKG_CMD" ]; then
    $OPKG_CMD update || echo "Warning: opkg update failed."
    echo "Installing/Upgrading dependencies..."
    $OPKG_CMD install jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle ca-certificates || true
    
    # Try to install full sing-box. We use --force-overwrite to be safe.
    $OPKG_CMD install sing-box || {
        echo "Space is low. Installing 'sing-box-tiny' instead..."
        $OPKG_CMD install sing-box-tiny || true
    }
fi

# 6. Download and Install Obhod (EXACT FILENAMES)
cd /tmp
echo "Downloading Obhod..."
rm -f /tmp/obhod.ipk /tmp/luci.ipk

if ! wget -q -O "obhod.ipk" "$REPO_URL/$CORE_PKG"; then
    echo "Error: Package $CORE_PKG not found for your architecture at $REPO_URL."
    echo "Manual download link: $REPO_URL/$CORE_PKG"
    exit 1
fi
wget -q -O "luci.ipk" "$REPO_URL/$LUCI_PKG"

echo "Installing Obhod packages..."
if [ -x "$OPKG_CMD" ]; then
    $OPKG_CMD install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
    $OPKG_CMD install "/tmp/luci.ipk" --force-reinstall --force-overwrite
else
    echo "Error: Cannot install .ipk files without opkg. Packages downloaded to /tmp."
    echo "LuCI: /tmp/luci.ipk"
    echo "Core: /tmp/obhod.ipk"
    exit 1
fi

# 7. Cleanup and Finish
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
