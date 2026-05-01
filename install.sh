#!/bin/sh

# Obhod VPN One-Line Installer v0.2.0-6
# Usage: sh <(wget -qO- "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh?$(date +%s)")

set -e

echo "--- Obhod VPN Installer v0.2.0-6 ---"

PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod_0.2.0-6_all.ipk"

# Download and Install
echo "Downloading Universal Package (Shell Backend + Vue UI)..."
wget --no-check-certificate -O /tmp/obhod.ipk "$PKG_URL"

echo "Installing Obhod VPN..."
opkg update
opkg install --force-reinstall /tmp/obhod.ipk

echo "--- Installation Complete ---"
echo "Please refresh your browser. Menu: Services -> Obhod"
