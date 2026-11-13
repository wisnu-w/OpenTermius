#!/usr/bin/env bash
set -e

# OpenTermius Android APK Build Script
if [ -x "/usr/libexec/java_home" ]; then
  export JAVA_HOME="$(/usr/libexec/java_home)"
else
  export JAVA_HOME="/Library/Java/JavaVirtualMachines/jdk-23.jdk/Contents/Home"
fi
export ANDROID_HOME="$HOME/Library/Android/sdk"
export NDK_HOME="$ANDROID_HOME/ndk/27.2.12479018"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"

echo "=== OpenTermius Android Build Environment ==="
echo "ANDROID_HOME: $ANDROID_HOME"
echo "NDK_HOME:     $NDK_HOME"

cd "$(dirname "$0")/.."

MODE="${1:-debug}"

if [ "$MODE" = "release" ]; then
  echo "Building signed/release Android APK..."
  npx tauri android build --apk
else
  echo "Building debug Android APK..."
  npx tauri android build --debug
fi

echo "✓ Android APK build complete!"
find src-tauri/gen/android/app/build/outputs/apk -name "*.apk" 2>/dev/null
