#!/bin/sh
# Regenerates every icon derived from build/icon/wireguide.svg.
#
# Requires rsvg-convert (Fedora: librsvg2-tools, Debian: librsvg2-bin) and
# Go. Run from anywhere; outputs are committed, so this only needs re-running
# after the master SVG changes.
#
# The macOS .icns is not produced here — `task common:generate:icons` builds
# it from build/appicon.png with wails3.
set -eu

root=$(cd "$(dirname "$0")/../.." && pwd)
src="$root/build/icon/wireguide.svg"
hicolor="$root/build/linux/icons/hicolor"

png() { rsvg-convert -w "$2" -h "$2" "$1" -o "$3"; }

# Cross-platform sources: the 1024px master raster (embedded in the binary
# for the tray, input for .ico/.icns) and the copies the frontend/README use.
png "$src" 1024 "$root/build/appicon.png"
png "$src" 256 "$root/frontend/public/appicon.png"
png "$src" 256 "$root/docs/appicon.png"
cp "$src" "$root/frontend/public/wireguide.svg"

# Linux: the master keeps the macOS grid's 92px inset, which makes the tile
# look undersized next to other icons in GNOME's app grid and dash. Crop the
# viewBox so the tile fills ~92% of the slot instead.
mkdir -p "$hicolor/scalable/apps"
sed 's|viewBox="0 0 1024 1024"|viewBox="56 56 912 912"|' "$src" \
    > "$hicolor/scalable/apps/wireguide.svg"
for size in 16 24 32 48 64 128 256 512; do
    mkdir -p "$hicolor/${size}x${size}/apps"
    png "$hicolor/scalable/apps/wireguide.svg" "$size" \
        "$hicolor/${size}x${size}/apps/wireguide.png"
done

# Windows: multi-size .ico with the rounded mask baked into each entry.
(cd "$root/build" && go run ./tools/genicon -in appicon.png -out windows/icon.ico)
