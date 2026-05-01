#!/bin/sh

# Obhod VPN One-Line Installer v0.1.3
# Usage: sh <(wget -qO- "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh?$(date +%s)")

set -e

echo "--- Obhod VPN Installer v0.1.3 ---"

# 1. Detect architecture
# Filter out 'all' and 'noarch' and get the most specific one
ARCH_LIST=$(opkg print-architecture | awk '$2 !~ /all|noarch/ {print $2}')
echo "Available architectures: $ARCH_LIST"

PKG_URL=""

# Priority check for aarch64
if echo "$ARCH_LIST" | grep -q "aarch64"; then
    echo "Selecting ARM64 package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-arm64.ipk"
# Priority check for mips
elif echo "$ARCH_LIST" | grep -q "mips"; then
    echo "Selecting MIPS package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-mips.ipk"
fi

if [ -z "$PKG_URL" ]; then
    echo "Error: Could not find a suitable package for your architectures: $ARCH_LIST"
    exit 1
fi

# 2. Download and Install
echo "Downloading package from GitHub..."
wget --no-check-certificate -O /tmp/obhod.ipk "$PKG_URL"

echo "Installing Obhod VPN..."
opkg update
opkg install --force-reinstall /tmp/obhod.ipk

echo "--- Installation Complete ---"
echo "Please refresh your browser. Menu: Services -> Obhod VPN"
