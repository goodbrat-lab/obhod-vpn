#!/bin/sh

# Universal Installer for Obhod VPN v1.1.1 (Apk Support & DNS Fix)
# High compatibility with OpenWrt standard architecture names and OpenWrt 25.xx (OneWrt/apk).

set -e

# 1. Environment and Debugging
export PATH=/bin:/sbin:/usr/bin:/usr/sbin:"$PATH"
REPO_URL="https://github.com/goodbrat-lab/obhod-vpn/raw/main/dist/packages"
VERSION="1.1.8"
RELEASE="1"
LUCI_PKG="luci-app-obhod_${VERSION}-${RELEASE}_all.ipk"

echo "=================================================="
echo "      Obhod VPN - Universal Installer v1.1.8      "
echo "=================================================="
echo "System Debug Info:"
echo "  PATH: $PATH"
echo "  Commands: $(command -v opkg || echo 'opkg not found'), $(command -v apk || echo 'apk not found')"
echo "  Uname: $(uname -a)"
echo "=================================================="

# 2. Temporary Files Cleanup
cleanup_tmp() {
    rm -f /tmp/sb.tar.gz /tmp/obhod.ipk /tmp/luci.ipk /tmp/obhod.tar.gz /tmp/luci.tar.gz 2>/dev/null || true
    rm -rf /tmp/sing-box-* /tmp/ex_obhod.ipk /tmp/ex_luci.ipk 2>/dev/null || true
}

# 3. Package Manager Detection
OPKG_CMD=$(command -v opkg || echo "/bin/opkg")
APK_CMD=$(command -v apk || echo "/usr/bin/apk")

if $OPKG_CMD --version >/dev/null 2>&1; then
    OPKG_WORKS=1
else
    OPKG_WORKS=0
fi

# 3. Check for sing-box DNS support
check_sb_dns_at() {
    sb_path="$1"
    if [ ! -x "$sb_path" ]; then return 1; fi
    echo '{"inbounds":[{"type":"dns","tag":"dns-in","listen":"127.0.0.1","listen_port":5353}]}' > /tmp/obhod_sb_test.json
    if "$sb_path" check -c /tmp/obhod_sb_test.json >/dev/null 2>&1; then
        rm -f /tmp/obhod_sb_test.json
        return 0
    else
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

# 5. Install Dependencies and Fix Binary
get_latest_singbox_version() {
    local default_version="v1.12.0"
    local version=""

    # 1. Check environment variable
    if [ -n "$SINGBOX_VERSION" ]; then
        echo "Using version from environment variable: $SINGBOX_VERSION" >&2
        echo "$SINGBOX_VERSION"
        return 0
    fi

    # 2. Check UCI configuration if exists
    if [ -x "$(command -v uci)" ]; then
        version=$(uci -q get obhod.settings.singbox_version)
        if [ -n "$version" ]; then
            echo "Using version from UCI: $version" >&2
            echo "$version"
            return 0
        fi
    fi

    # 3. Attempt to fetch from GitHub API using curl
    if [ -x "$(command -v curl)" ]; then
        echo "Querying GitHub API for latest sing-box version..." >&2
        local api_res
        set +e
        api_res=$(curl -sL --connect-timeout 3 -m 5 "https://api.github.com/repos/SagerNet/sing-box/releases/latest" 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$api_res" ]; then
            # Parse tag_name via grep + cut to avoid jq dependency before it is installed
            version=$(echo "$api_res" | grep -o '"tag_name": *"[^"]*"' | head -n 1 | cut -d'"' -f4)
        fi
        set -e
    fi

    if [ -n "$version" ] && [ "$version" != "null" ]; then
        echo "Latest version from GitHub: $version" >&2
        echo "$version"
    else
        echo "Failed to get version dynamically, falling back to default: $default_version" >&2
        echo "$default_version"
    fi
}

install_deps() {
    needs_fix=0
    ARCH_M=""
    SB_ARCH=""

    echo "Updating package lists..."
    if [ -x "$APK_CMD" ] && [ "$OPKG_WORKS" -eq 0 ]; then
        $APK_CMD update
        $APK_CMD add jq curl nftables kmod-nft-tproxy coreutils-base64 bind-dig ca-bundle sing-box || true
    else
        $OPKG_CMD update
        $OPKG_CMD install jq curl nftables kmod-nft-tproxy coreutils-base64 bind-dig ca-bundle sing-box || true
    fi

    [ -x /usr/bin/sing-box ] && ! check_sb_dns_at /usr/bin/sing-box && needs_fix=1
    [ -x /usr/sbin/sing-box ] && ! check_sb_dns_at /usr/sbin/sing-box && needs_fix=1

    if [ "$needs_fix" -eq 1 ]; then
        echo "⚠️  Detected limited 'sing-box' (no DNS support). Replacing with full version..."
        ARCH_M=$(uname -m)
        case "$ARCH_M" in
            x86_64) SB_ARCH="amd64" ;;
            aarch64) SB_ARCH="arm64" ;;
            armv7*) SB_ARCH="armv7" ;;
            mips) SB_ARCH="mips" ;;
            mipsel) SB_ARCH="mipsel" ;;
        esac

        if [ -n "$SB_ARCH" ]; then
            local sb_ver
            sb_ver=$(get_latest_singbox_version)
            local sb_ver_no_v="${sb_ver#v}"
            # Ensure sb_ver has 'v' prefix
            if [ "${sb_ver#v}" = "$sb_ver" ]; then
                sb_ver="v$sb_ver"
            fi

            echo "  -> Downloading full sing-box $sb_ver for $SB_ARCH..."
            local dl_ok=0
            set +e
            # Try downloading musl version first (recommended for OpenWrt)
            wget -q "https://github.com/SagerNet/sing-box/releases/download/${sb_ver}/sing-box-${sb_ver_no_v}-linux-${SB_ARCH}-musl.tar.gz" -O /tmp/sb.tar.gz
            if [ $? -eq 0 ]; then
                dl_ok=1
            else
                # Fallback to standard version if musl version doesn't exist
                wget -q "https://github.com/SagerNet/sing-box/releases/download/${sb_ver}/sing-box-${sb_ver_no_v}-linux-${SB_ARCH}.tar.gz" -O /tmp/sb.tar.gz
                if [ $? -eq 0 ]; then
                    dl_ok=1
                fi
            fi
            set -e

            if [ "$dl_ok" -eq 1 ]; then
                tar -xzf /tmp/sb.tar.gz -C /tmp
                [ -f /usr/bin/sing-box ] && cp /tmp/sing-box-*/sing-box /usr/bin/sing-box && chmod +x /usr/bin/sing-box
                [ -f /usr/sbin/sing-box ] && cp /tmp/sing-box-*/sing-box /usr/sbin/sing-box && chmod +x /usr/sbin/sing-box
                [ ! -f /usr/bin/sing-box ] && [ ! -f /usr/sbin/sing-box ] && cp /tmp/sing-box-*/sing-box /usr/bin/sing-box && chmod +x /usr/bin/sing-box
                echo "  ✅ Full sing-box binary replaced successfully."
                cleanup_tmp
            else
                echo "  ⚠️  Failed to download precompiled full sing-box binary for $SB_ARCH (possibly 404 on SagerNet releases)."
                echo "     Obhod will be installed, but it cannot start until you manually install"
                echo "     a full version of sing-box (with DNS inbound support)."
            fi
        fi
    fi
}

# 6. Service Cleanup
echo "Cleaning up..."
/etc/init.d/obhod stop 2>/dev/null || true
cleanup_tmp

# 7. Main Install
install_deps

USE_TARBALLS=0
if [ "$OPKG_WORKS" -eq 0 ] && ! command -v ar >/dev/null 2>&1; then
    USE_TARBALLS=1
    echo "Notice: 'ar' utility is missing. Will use backup tarball installation method."
fi

cd /tmp
echo "Downloading Obhod packages..."
if [ "$USE_TARBALLS" -eq 1 ]; then
    CORE_PKG_FILE="obhod_${VERSION}-${RELEASE}_${ARCH}.tar.gz"
    LUCI_PKG_FILE="luci-app-obhod_${VERSION}-${RELEASE}_all.tar.gz"
    wget -q --no-check-certificate "$REPO_URL/$CORE_PKG_FILE" -O obhod.tar.gz
    wget -q --no-check-certificate "$REPO_URL/$LUCI_PKG_FILE" -O luci.tar.gz
else
    wget -q --no-check-certificate "$REPO_URL/$CORE_PKG" -O obhod.ipk
    wget -q --no-check-certificate "$REPO_URL/$LUCI_PKG" -O luci.ipk
fi

if [ "$OPKG_WORKS" -eq 1 ]; then
    $OPKG_CMD install "/tmp/obhod.ipk" --force-reinstall --force-overwrite
    $OPKG_CMD install "/tmp/luci.ipk" --force-reinstall --force-overwrite
elif [ "$USE_TARBALLS" -eq 1 ]; then
    echo "Extracting packages via manual tarball mode..."
    tar -xzf /tmp/obhod.tar.gz -C /
    tar -xzf /tmp/luci.tar.gz -C /
    
    # Run post-install hooks manually
    chmod +x /usr/bin/obhod /usr/lib/obhod/obhod-backend.sh /etc/init.d/obhod 2>/dev/null
    /etc/init.d/obhod enable 2>/dev/null
    rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache/ 2>/dev/null
    /etc/init.d/rpcd restart 2>/dev/null
else
    echo "Extracting packages via manual ipk mode..."
    for p in obhod.ipk luci.ipk; do
       EXT="/tmp/ex_$p"
       rm -rf "$EXT" && mkdir -p "$EXT" && cd "$EXT"
       ar x "/tmp/$p"
       tar -xzf data.tar.gz -C /
       if tar -xzf control.tar.gz ./postinst 2>/dev/null; then
           chmod +x postinst
           ./postinst || true
       fi
    done
fi

echo "Finalizing..."
/etc/init.d/rpcd restart 2>/dev/null || true
/etc/init.d/uhttpd restart 2>/dev/null || true
cleanup_tmp

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
