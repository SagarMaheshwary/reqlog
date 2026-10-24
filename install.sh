#!/usr/bin/env bash

set -e

REPO="sagarmaheshwary/reqlog"
INSTALL_PATH="/usr/local/bin/reqlog"

echo "Installing reqlog..."

# Version argument
VERSION="${1:-latest}"

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux) PLATFORM="linux" ;;
  Darwin) PLATFORM="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Normalize architecture
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Resolve release version
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -s \
    "https://api.github.com/repos/$REPO/releases/latest" |
    grep tag_name |
    cut -d '"' -f 4)
fi

if [ -z "$VERSION" ]; then
  echo "Failed to resolve release version"
  exit 1
fi

# Release tags are prefixed with "v"
case "$VERSION" in
  v*) ;;
  *) VERSION="v${VERSION}" ;;
esac

# Asset naming changed in v0.10.0 (GoReleaser):
#   <= v0.9.1: reqlog-linux-amd64.tar.gz containing "reqlog-linux-amd64"
#   >= v0.10.0: reqlog_0.10.0_linux_amd64.tar.gz containing "reqlog"
SEMVER="${VERSION#v}"
CORE="${SEMVER%%-*}"
IFS=. read -r MAJOR MINOR _ <<< "$CORE"

if [ "${MAJOR:-0}" -gt 0 ] || [ "${MINOR:-0}" -ge 10 ]; then
  BINARY="reqlog"
  TAR_FILE="reqlog_${SEMVER}_${PLATFORM}_${ARCH}.tar.gz"
else
  BINARY="reqlog-${PLATFORM}-${ARCH}"
  TAR_FILE="${BINARY}.tar.gz"
fi

URL="https://github.com/$REPO/releases/download/$VERSION/${TAR_FILE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading $TAR_FILE ($VERSION)..."

curl -fL "$URL" -o "$TMP_DIR/$TAR_FILE"

tar -xzf "$TMP_DIR/$TAR_FILE" -C "$TMP_DIR"

chmod +x "$TMP_DIR/$BINARY"
sudo mv "$TMP_DIR/$BINARY" "$INSTALL_PATH"

echo "Installed reqlog $VERSION at $INSTALL_PATH"