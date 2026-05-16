#!/bin/sh

# Universal Installer for Obhod VPN v0.3.2 (Apk Support)
# High compatibility with OpenWrt standard architecture names and OpenWrt 25.xx (OneWrt/apk).

set -e

# 1. Environment and Debugging
export PATH=/bin:/sbin:/usr/bin:/usr/sbin:$PATH
REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="0.3.3"
LUCI_PKG="luci-app-obhod_${VERSION}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v0.3.3      "
echo "=================================================="
echo "System Debug Info:"
echo "  PATH: $PATH"
echo "  Commands: $(command -v opkg || echo 'opkg not found'), $(command -v apk || echo 'apk not found')"
echo "  Uname: $(uname -a)"
echo "=================================================="

# 2. Package Manager Detection
OPKG_CMD=$(command -v opkg || echo "/bin/opkg")
APK_CMD=$(command -v apk || echo "/usr/bin/apk")

# Precise check: does opkg actually work?
if ! $OPKG_CMD --version >/dev/null 2>&1; then
    OPKG_WORKS=0
else
    OPKG_WORKS=1
fi

if [ "$OPKG_WORKS" -eq 0 ] && [ ! -x "$APK_CMD" ]; then
    echo "CRITICAL ERROR: No package manager found (opkg/apk)."
    echo "Please install Obhod manually by downloading the packages:"
    echo "1. Download Luci: $REPO_URL/$LUCI_PKG"
    echo "2. Download Core: $REPO_URL/obhod_${VERSION}_[ARCH].ipk"
    exit 1
fi

# 3. Precise Architecture Detection
echo "Detecting architecture..."
ARCH=""

if [ "$OPKG_WORKS" -eq 1 ]; then
    ARCH=$($OPKG_CMD print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')
fi

if [ -z "$ARCH" ]; then
    echo "Warning: opkg detection failed or missing, falling back to uname -m mapping."
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
if [ "$OPKG_WORKS" -eq 1 ]; then
    $OPKG_CMD remove obhod luci-app-obhod --force-removal-of-dependent-packages 2>/dev/null || true
fi

# 5. Dependency handling
echo "Updating package lists..."
if [ -x "$APK_CMD" ] && [ "$OPKG_WORKS" -eq 0 ]; then
    echo "Detected apk (OpenWrt 25+). Installing dependencies..."
    $APK_CMD update || echo "Warning: apk update failed."
    # Try to install opkg for better .ipk handling
    $APK_CMD add opkg || echo "Note: Could not install opkg via apk."
    if command -v opkg >/dev/null 2>&1; then
        OPKG_CMD=$(command -v opkg)
        if $OPKG_CMD --version >/dev/null 2>&1; then
            OPKG_WORKS=1
        fi
    fi
fi

if [ "$OPKG_WORKS" -eq 1 ]; then
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
echo "Downloading Obhod packages..."

download_file() {
    local url="$1"
    local dest="$2"
    echo "  -> Downloading: $url"
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sL --connect-timeout 15 -o "$dest" "$url"; then
            echo "Error: curl failed to download $url"
            return 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q --no-check-certificate --timeout=15 -O "$dest" "$url"; then
            echo "Error: wget failed to download $url"
            return 1
        fi
    else
        echo "Error: No download tool found (curl or wget)."
        return 1
    fi
    return 0
}

rm -f /tmp/obhod.ipk /tmp/luci.ipk
if ! download_file "$REPO_URL/$CORE_PKG" "obhod.ipk"; then
    echo "CRITICAL ERROR: Failed to download $CORE_PKG"
    exit 1
fi

if ! download_file "$REPO_URL/$LUCI_PKG" "luci.ipk"; then
    echo "CRITICAL ERROR: Failed to download $LUCI_PKG"
    exit 1
fi

echo "Installing Obhod packages..."
if [ "$OPKG_WORKS" -eq 1 ]; then
    $OPKG_CMD install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
    $OPKG_CMD install "/tmp/luci.ipk" --force-reinstall --force-overwrite
elif [ -x "$APK_CMD" ]; then
    echo "Installing via manual extraction (apk fallback)..."
    for pkg in "obhod.ipk" "luci.ipk"; do
        echo "Extracting $pkg..."
        EXTRACT_DIR="/tmp/extract_$pkg"
        rm -rf "$EXTRACT_DIR"
        mkdir -p "$EXTRACT_DIR"
        
        # ipk is a tar.gz containing data.tar.gz and control.tar.gz
        if ! tar -xzf "/tmp/$pkg" -C "$EXTRACT_DIR"; then
            echo "Error: Failed to unpack $pkg"
            exit 1
        fi

        echo "  - Deploying files..."
        if ! tar -xzf "$EXTRACT_DIR/data.tar.gz" -C /; then
            echo "Error: Failed to deploy data from $pkg"
            exit 1
        fi

        echo "  - Running post-install..."
        tar -xzf "$EXTRACT_DIR/control.tar.gz" -C "$EXTRACT_DIR" || true
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
rm -rf /tmp/luci-indexcache*
rm -rf /tmp/luci-modulecache/
/etc/init.d/rpcd restart 2>/dev/null || true
/etc/init.d/uhttpd restart 2>/dev/null || true

if [ -f "/usr/bin/obhod" ]; then
    echo "=================================================="
    echo "           INSTALLATION SUCCESSFUL!               "
    echo "=================================================="
    echo "Obhod Version: $(/usr/bin/obhod show_version 2>/dev/null || echo $VERSION)"
    SB_VER=$(sing-box version 2>/dev/null | head -n1 | awk '{print $3}')
    echo "Sing-box Version: ${SB_VER:-unknown}"
    echo "=================================================="
else
    echo "Error: Installation failed. Binary /usr/bin/obhod is missing."
    exit 1
fi
