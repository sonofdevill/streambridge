#!/usr/bin/env bash
# Builds StreamBridge for every platform and packages the release files into dist/.
# Usage: ./build.sh            (version is read from main.go)
#        ./build.sh --site     (also copies the packages + SHA256SUMS.txt into site/downloads)
set -euo pipefail
cd "$(dirname "$0")"
ROOT=$(pwd)

VERSION=$(sed -n 's/^var version = "\(.*\)"/\1/p' main.go)   # e.g. 1.3.0
SHORT=${VERSION%.0}                                           # e.g. 1.3
OUT=dist
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

export CGO_ENABLED=0
LDFLAGS="-s -w"
DOCS=(README.txt THIRD_PARTY_LICENSES.txt)

echo "StreamBridge $VERSION"
rm -rf "$OUT" && mkdir -p "$OUT"

build() { # GOOS GOARCH [GOARM] OUTPUT [extra ldflags]
  local os=$1 arch=$2 arm=$3 out=$4 extra=${5:-}
  echo "  building $os/$arch${arm:+ v$arm}"
  GOOS=$os GOARCH=$arch GOARM=$arm go build -trimpath -ldflags "$LDFLAGS $extra" -o "$out" .
}

# Windows: windowsgui hides the console; icon + version info come from rsrc_windows_amd64.syso
mkdir -p "$STAGE/win/StreamBridge"
build windows amd64 "" "$STAGE/win/StreamBridge/StreamBridge.exe" "-H=windowsgui"
cp "${DOCS[@]}" "$STAGE/win/StreamBridge/"
(cd "$STAGE/win" && zip -q -9 -r "$ROOT/$OUT/StreamBridge-$SHORT-Windows-x64.zip" StreamBridge)

# macOS: Apple Silicon + Intel in one zip
mkdir -p "$STAGE/mac/StreamBridge"
build darwin arm64 "" "$STAGE/mac/StreamBridge/StreamBridge-macOS-AppleSilicon"
build darwin amd64 "" "$STAGE/mac/StreamBridge/StreamBridge-macOS-Intel"
cp "${DOCS[@]}" "$STAGE/mac/StreamBridge/"
(cd "$STAGE/mac" && zip -q -9 -r "$ROOT/$OUT/StreamBridge-$SHORT-macOS.zip" StreamBridge)

# Linux: x86-64, ARM64, ARMv7 (Raspberry Pi)
mkdir -p "$STAGE/linux/StreamBridge"
build linux amd64 "" "$STAGE/linux/StreamBridge/streambridge-linux-amd64"
build linux arm64 "" "$STAGE/linux/StreamBridge/streambridge-linux-arm64"
build linux arm 7 "$STAGE/linux/StreamBridge/streambridge-linux-armv7"
cp "${DOCS[@]}" "$STAGE/linux/StreamBridge/"
tar -C "$STAGE/linux" -czf "$OUT/StreamBridge-$SHORT-Linux.tar.gz" StreamBridge

# Source
mkdir -p "$STAGE/src/StreamBridge-source"
cp -r *.go go.mod go.sum web assets winres rsrc_windows_amd64.syso "${DOCS[@]}" "$STAGE/src/StreamBridge-source/"
(cd "$STAGE/src" && zip -q -9 -r "$ROOT/$OUT/StreamBridge-$SHORT-source.zip" StreamBridge-source)

(cd "$OUT" && sha256sum StreamBridge-* > SHA256SUMS.txt && cat SHA256SUMS.txt)

if [[ "${1:-}" == "--site" ]]; then
  cp "$OUT"/StreamBridge-* "$OUT"/SHA256SUMS.txt site/downloads/
  echo "Copied to site/downloads. Update file names, sizes, hashes and version in site/index.html and site/assets/site.js."
fi
