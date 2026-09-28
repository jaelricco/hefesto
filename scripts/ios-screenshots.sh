#!/usr/bin/env bash
# Screenshots of the iOS app's screens for design review, taken on a Mac with
# Xcode (the self-hosted runner, or by hand). The app runs in its demo mode
# (ios/Hefesto/DemoMode.swift): seeded local data, no server, no account.
#
#   scripts/ios-screenshots.sh [out-dir]    # default: ios/screenshots
#
# It builds the Debug app into ios/.derived-screenshots, boots one simulator
# named "Hefesto Screenshots" (created on the first run and reused), saves one
# PNG per screen in German, and shuts the simulator down again. Nothing else
# on the machine changes.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
out="${1:-$root/ios/screenshots}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
rm -f "$out"/*.png

device_name="Hefesto Screenshots"
bundle_id="fit.hefesto.ios"
# DemoScreen's raw values, in the order a reviewer walks through the app.
screens=(today logger composer finish map peek detail history session stats celebration signin)

# The newest installed iOS runtime, and the first of these iPhones it offers.
runtime=$(xcrun simctl list runtimes available | grep -E '^iOS ' | tail -1 | sed -E 's/.* - ([^ ]+)$/\1/')
if [ -z "$runtime" ]; then
  echo "no iOS simulator runtime installed (Xcode > Settings > Components)" >&2
  exit 1
fi
device_type=""
for model in "iPhone 17 Pro" "iPhone 17" "iPhone 16 Pro" "iPhone 16" "iPhone 15 Pro"; do
  device_type=$(xcrun simctl list devicetypes | grep -F "$model (" | head -1 | sed -E 's/.*\((com\.apple[^)]+)\)$/\1/' || true)
  [ -n "$device_type" ] && break
done
if [ -z "$device_type" ]; then
  echo "no iPhone simulator device type found" >&2
  exit 1
fi

udid=$(xcrun simctl list devices available | grep -F "    $device_name (" | head -1 \
  | sed -E 's/.*\(([0-9A-F-]{36})\).*/\1/' || true)
if [ -z "$udid" ]; then
  echo "creating the simulator \"$device_name\" ($device_type, $runtime)"
  udid=$(xcrun simctl create "$device_name" "$device_type" "$runtime")
fi
echo "simulator: $device_name $udid"

cd "$root/ios"
[ -d Hefesto.xcodeproj ] || xcodegen generate
xcodebuild build \
  -project Hefesto.xcodeproj -scheme Hefesto -configuration Debug \
  -destination "id=$udid" \
  -derivedDataPath .derived-screenshots \
  -skipPackagePluginValidation -skipMacroValidation \
  CODE_SIGNING_ALLOWED=NO 2>&1 | tee screenshots-build.log | grep -E 'error:|\*\* BUILD' || true
if ! grep -q '\*\* BUILD SUCCEEDED \*\*' screenshots-build.log; then
  grep -B2 -A12 -E 'error:|Undefined symbols' screenshots-build.log | tail -150
  exit 1
fi
app=".derived-screenshots/Build/Products/Debug-iphonesimulator/Hefesto.app"

cleanup() {
  xcrun simctl terminate "$udid" "$bundle_id" >/dev/null 2>&1 || true
  xcrun simctl status_bar "$udid" clear >/dev/null 2>&1 || true
  xcrun simctl shutdown "$udid" >/dev/null 2>&1 || true
}
trap cleanup EXIT

xcrun simctl boot "$udid" 2>/dev/null || true
xcrun simctl bootstatus "$udid" -b >/dev/null
xcrun simctl ui "$udid" appearance dark
xcrun simctl status_bar "$udid" override --time 9:41 --batteryState charged --batteryLevel 100 \
  --wifiBars 3 --cellularMode active --cellularBars 4
xcrun simctl install "$udid" "$app"

i=0
for screen in "${screens[@]}"; do
  i=$((i + 1))
  xcrun simctl terminate "$udid" "$bundle_id" >/dev/null 2>&1 || true
  xcrun simctl launch "$udid" "$bundle_id" \
    -HefestoDemo YES -HefestoDemoScreen "$screen" -AppleLanguages "(de)" -AppleLocale de_CH >/dev/null
  # Long enough for the fonts, the data and any sheet or push to settle.
  sleep 6
  file="$out/$(printf '%02d' "$i")-$screen.png"
  xcrun simctl io "$udid" screenshot --type=png "$file" >/dev/null 2>&1
  echo "saved $file"
done
