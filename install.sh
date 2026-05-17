#!/bin/sh

# Universal Installer for Obhod VPN v1.1.0 (Apk Support & DNS Fix)
# High compatibility with OpenWrt standard architecture names and OpenWrt 25.xx (OneWrt/apk).

set -e

# 1. Environment and Debugging
export PATH=/bin:/sbin:/usr/bin:/usr/sbin:$PATH
REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="1.1.0"
RELEASE="1"
LUCI_PKG="luci-app-obhod_${VERSION}-${RELEASE}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v1.1.0      "
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
    exit 1
fi

# 3. Check for sing-box DNS support
check_sb_dns() {
    if ! command -v sing-box >/dev/null 2>&1; then return 1; fi
    # Check if binary is 1.12+ (which we want)
    SB_VER=$(sing-box version 2>/dev/null | head -n1 | awk '{print $3}')
    echo "Current sing-box version: $SB_VER"
    
    echo '{"inbounds":[{"type":"dns","tag":"dns-in","listen":"127.0.0.1","listen_port":5353}]}' > /tmp/obhod_sb_test.json
    if sing-box check -c /tmp/obhod_sb_test.json >/dev/null 2>&1; then
        rm -f /tmp/obhod_sb_test.json
        return 0
    else
        echo "Validation failed for 'dns' inbound type."
        rm -f /tmp/obhod_sb_test.json
        return 1
    fi
}

# 4. Architecture Detection
echo "Detecting architecture..."
ARCH=""
if [ "$OPKG_WORKS" -eq 1 ]; then
    ARCH=$($OPKG_CMD print-architecture | grep -vE 'all|noarch' | tail -n 1 | awk '{print $2}')
fi

if [ -z "$ARCH" ]; then
    UNAME_M=$(uname -m)
    case "$UNAME_M" in
        aarch64) ARCH="aarch64_cortex-a53" ;;
        armv7l)  ARCH="arm_cortex-a7_neon-vfpv4" ;;
        mips)    ARCH="mips_24kc" ;;
        mipsel)  ARCH="mipsel_24kc" ;;
        x86_64)  ARCH="x86_64" ;;
    esac
fi
CORE_PKG="obhod_${VERSION}-${RELEASE}_${ARCH}.ipk"
echo "Architecture: $ARCH"

# 5. Install Dependencies
install_deps() {
    echo "Updating package lists..."
    if [ -x "$APK_CMD" ] && [ "$OPKG_WORKS" -eq 0 ]; then
        $APK_CMD update
        $APK_CMD add jq curl nftables kmod-nft-tproxy coreutils-base64 bind-dig ca-bundle sing-box || true
    else
        $OPKG_CMD update
        $OPKG_CMD install jq curl nftables kmod-nft-tproxy coreutils-base64 bind-dig ca-bundle sing-box || true
    fi

    if ! check_sb_dns; then
        echo "⚠️  Detected limited 'sing-box' (no DNS support). Replacing with full version..."
        local ARCH_M=$(uname -m)
        local SB_ARCH=""
        case "$ARCH_M" in
            x86_64) SB_ARCH="amd64" ;;
            aarch64) SB_ARCH="arm64" ;;
            armv7*) SB_ARCH="armv7" ;;
            mips) SB_ARCH="mips" ;;
            mipsel) SB_ARCH="mipsel" ;;
        esac

        if [ -n "$SB_ARCH" ]; then
            echo "  -> Downloading full sing-box 1.12.0 for $SB_ARCH..."
            wget "https://github.com/SagerNet/sing-box/releases/download/v1.12.0/sing-box-1.12.0-linux-$SB_ARCH.tar.gz" -O /tmp/sb.tar.gz
            tar -xzf /tmp/sb.tar.gz -C /tmp
            cp /tmp/sing-box-*/sing-box /usr/bin/sing-box
            chmod +x /usr/bin/sing-box
            echo "  ✅ Full sing-box installed successfully."
        else
            echo "  ❌ Unknown architecture for manual install. Obhod might fail."
        fi
    else
        echo "  ✅ sing-box already supports DNS. Skipping replacement."
    fi
}

# 6. Service Cleanup
echo "Cleaning up..."
/etc/init.d/obhod stop 2>/dev/null || true

# 7. Main Install
install_deps

# Download and manual extraction for apk/OneWrt or opkg install
cd /tmp
echo "Downloading Obhod packages..."
wget -q --no-check-certificate "$REPO_URL/$CORE_PKG" -O obhod.ipk
wget -q --no-check-certificate "$REPO_URL/$LUCI_PKG" -O luci.ipk

if [ "$OPKG_WORKS" -eq 1 ]; then
    $OPKG_CMD install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
    $OPKG_CMD install "/tmp/luci.ipk" --force-reinstall --force-overwrite
else
    # Manual extraction logic (simplified for brevity here, assume existing)
    echo "Extracting packages via manual mode..."
    # (Extract logic same as 1.0.9 but with VERSION 1.1.0)
    for p in obhod.ipk luci.ipk; do
       EXT="/tmp/ex_$p"
       rm -rf "$EXT" && mkdir -p "$EXT" && cd "$EXT"
       ar x "/tmp/$p"
       tar -xzf data.tar.gz -C /
       # Run postinst if exists
       tar -xzf control.tar.gz ./postinst 2>/dev/null && chmod +x postinst && ./postinst || true
    done
fi

echo "Finalizing..."
/etc/init.d/rpcd restart 2>/dev/null || true
/etc/init.d/uhttpd restart 2>/dev/null || true

if [ -f /usr/bin/obhod ]; then
    echo "=================================================="
    echo "           INSTALLATION SUCCESSFUL!               "
    echo "=================================================="
    echo "Obhod Version: $VERSION"
    echo "Sing-box: $(sing-box version 2>/dev/null | head -n1)"
    echo "=================================================="
else
    echo "Error: Installation failed."
    exit 1
fi
