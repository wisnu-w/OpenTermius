#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$DIR/src-tauri/target/release/bundle/macos"
DMG_DIR="$DIR/src-tauri/target/release/bundle/dmg"
APP_NAME="opentermius.app"
DMG_NAME="OpenTermius_0.1.1_aarch64.dmg"
STAGING_DIR="/tmp/opentermius-dmg-staging"

echo "1. Building release binary..."
cd "$DIR"
npx tauri build --bundles app

echo "2. Preparing clean staging..."
rm -rf "$STAGING_DIR"
mkdir -p "$STAGING_DIR"
mkdir -p "$DMG_DIR"

ditto "$APP_DIR/$APP_NAME" "$STAGING_DIR/OpenTermius.app"

echo "3. Creating optimized macOS DMG installer..."
rm -f "$DMG_DIR/$DMG_NAME"
diskutil image create from --format UDZO --volumeName "OpenTermius" "$STAGING_DIR" "$DMG_DIR/$DMG_NAME"
rm -rf "$STAGING_DIR"

echo "✓ OpenTermius DMG installer successfully created:"
echo "  $DMG_DIR/$DMG_NAME"
