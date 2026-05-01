#!/bin/sh

# Obhod VPN One-Line Installer
# Usage: sh <(wget -qO- https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)

set -e

echo "--- Obhod VPN Installation Script ---"

# 1. Detect architecture
ARCH=$(opkg print-architecture | awk 'NR==1 {print $2}')
echo "Detected architecture: $ARCH"

PKG_URL=""
if echo "$ARCH" | grep -q "aarch64"; then
    echo "Selecting ARM64 package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-arm64.ipk"
elif echo "$ARCH" | grep -q "mips"; then
    echo "Selecting MIPS package..."
    PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod-mips.ipk"
else
    echo "Error: Unsupported architecture $ARCH"
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
