#!/bin/sh

# Obhod VPN One-Line Installer
# Usage: sh <(wget -qO- https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)

set -e

echo "--- Obhod VPN Installation Script ---"

# 1. Detect architecture
ARCH_LIST=$(opkg print-architecture | awk '{print $2}' | grep -v -E "all|noarch")
echo "System architectures: $ARCH_LIST"

PKG_URL=""
if echo "$ARCH_LIST" | grep -q "aarch64"; then
    echo "Selecting ARM64 package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-arm64.ipk"
elif echo "$ARCH_LIST" | grep -q "mips"; then
    echo "Selecting MIPS package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-mips.ipk"
else
    # Fallback to the first non-generic arch if no match found
    ARCH=$(echo "$ARCH_LIST" | head -n 1)
    echo "Error: Unsupported architecture $ARCH (or list: $ARCH_LIST)"
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
