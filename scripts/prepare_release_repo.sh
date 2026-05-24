#!/bin/bash

set -e

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"
DIST_DIR="$BASE_DIR/dist/packages"

cd "$DIST_DIR" || exit 1

VERSION=$(grep "OBHOD_VERSION=" "$BASE_DIR/obhod-core/files/usr/lib/constants.sh" | cut -d'"' -f2)

for path in ./*; do
    [ -e "$path" ] || continue
    name=$(basename "$path")

    case "$name" in
        Packages|Packages.gz|index.txt|luci-app-obhod_${VERSION}-1_all.ipk|obhod_${VERSION}-1_*.ipk|luci-app-obhod_${VERSION}-1_all.tar.gz|obhod_${VERSION}-1_*.tar.gz)
            continue
            ;;
    esac

    if [ -d "$path" ]; then
        rm -rf "$path"
    else
        rm -f "$path"
    fi
done

echo "Release repository cleaned: $DIST_DIR"
