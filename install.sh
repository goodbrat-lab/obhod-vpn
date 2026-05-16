#!/bin/sh

# Universal Installer for Obhod VPN v0.3.1
# High compatibility with OpenWrt standard architecture names and OpenWrt 25.xx (OneWrt/apk).

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
APK_CMD=$(command -v apk || echo "/usr/bin/apk")

if [ ! -x "$OPKG_CMD" ] && [ ! -x "$APK_CMD" ]; then
    echo "CRITICAL ERROR: No package manager found (opkg/apk)."
    echo "Please install Obhod manually by downloading the packages:"
    echo "1. Download Luci: $REPO_URL/$LUCI_PKG"
    echo "2. Download Core: $REPO_URL/obhod_${VERSION}_[ARCH].ipk"
    exit 1
fi

# 3. Precise Architecture Detection
echo "Detecting architecture..."
ARCH=""

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
elif [ -x "$APK_CMD" ]; then
    # apk doesn't handle .ipk metadata, so we just try to stop the service
    echo "Note: Using apk, manual removal of old files may be needed if previously installed."
fi

# 5. Dependency handling
echo "Updating package lists..."
if [ -x "$APK_CMD" ] && [ ! -x "$OPKG_CMD" ]; then
    echo "Detected apk (OpenWrt 25+). Installing dependencies..."
    $APK_CMD update || echo "Warning: apk update failed."
    # Try to install opkg for better .ipk handling
    $APK_CMD add opkg || echo "Note: Could not install opkg via apk."
    if command -v opkg >/dev/null 2>&1; then
        OPKG_CMD=$(command -v opkg)
    fi
fi

if [ -x "$OPKG_CMD" ]; then
    $OPKG_CMD update || echo "Warning: opkg update failed."
    echo "Installing dependencies via opkg..."
    $OPKG_CMD install jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle ca-certificates || true
    $OPKG_CMD install sing-box || $OPKG_CMD install sing-box-tiny || true
elif [ -x "$APK_CMD" ]; then
    echo "Installing dependencies via apk..."
    $APK_CMD add jq curl nftables coreutils-base64 kmod-nft-tproxy ca-bundle ca-certificates sing-box || \
    $APK_CMD add sing-box-tiny || true
fi

# 6. Download and Install Obhod
cd /tmp
echo "Downloading Obhod..."
rm -f /tmp/obhod.ipk /tmp/luci.ipk

if ! wget -q -O "obhod.ipk" "$REPO_URL/$CORE_PKG"; then
    echo "Error: Package $CORE_PKG not found for your architecture."
    exit 1
fi
wget -q -O "luci.ipk" "$REPO_URL/$LUCI_PKG"

echo "Installing Obhod packages..."
if [ -x "$OPKG_CMD" ]; then
    $OPKG_CMD install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
    $OPKG_CMD install "/tmp/luci.ipk" --force-reinstall --force-overwrite
elif [ -x "$APK_CMD" ]; then
    echo "Installing via manual extraction (apk fallback)..."
    for pkg in "obhod.ipk" "luci.ipk"; do
        echo "Extracting $pkg..."
        EXTRACT_DIR="/tmp/extract_$pkg"
        mkdir -p "$EXTRACT_DIR"
        # ipk is a tar.gz containing data.tar.gz and control.tar.gz
        tar -xzf "/tmp/$pkg" -C "$EXTRACT_DIR"
        echo "  - Deploying files..."
        tar -xzf "$EXTRACT_DIR/data.tar.gz" -C /
        echo "  - Running post-install..."
        tar -xzf "$EXTRACT_DIR/control.tar.gz" -C "$EXTRACT_DIR"
        if [ -f "$EXTRACT_DIR/postinst" ]; then
            chmod +x "$EXTRACT_DIR/postinst"
            sh "$EXTRACT_DIR/postinst" || echo "Warning: postinst failed for $pkg"
        fi
        rm -rf "$EXTRACT_DIR"
    done
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
    # Try to find sing-box version
    SB_VER=$(sing-box version 2>/dev/null | head -n1 | awk '{print $3}')
    echo "Sing-box Version: ${SB_VER:-unknown}"
    echo "=================================================="
else
    echo "Error: Installation failed. Binary /usr/bin/obhod is missing."
    exit 1
fi
