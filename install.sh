#!/bin/sh

# Obhod VPN One-Line Installer (Universal)
# Usage: sh <(wget -qO- "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh?$(date +%s)")

set -e

echo "--- Obhod VPN Installer v0.2.0-8 ---"

# Use fixed universal filename to avoid 404s due to version changes
PKG_URL="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/dist/obhod_universal.ipk"

# Download and Install
echo "Downloading Universal Package..."
wget --no-check-certificate -O /tmp/obhod.ipk "$PKG_URL"

echo "Installing Obhod VPN..."
opkg update
opkg install --force-reinstall /tmp/obhod.ipk

echo "--- Installation Complete ---"
echo "Please refresh your browser (Ctrl+F5). Menu: Services -> Obhod"
