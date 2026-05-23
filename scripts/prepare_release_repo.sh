#!/bin/bash

set -e

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"
DIST_DIR="$BASE_DIR/dist/packages"

cd "$DIST_DIR" || exit 1

for path in ./*; do
    [ -e "$path" ] || continue
    name=$(basename "$path")

    case "$name" in
        Packages|Packages.gz|index.txt|luci-app-obhod_1.1.5-1_all.ipk|obhod_1.1.5-1_*.ipk)
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
