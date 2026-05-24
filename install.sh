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

BINARY="reqlog-${PLATFORM}-${ARCH}"
TAR_FILE="${BINARY}.tar.gz"

# Resolve release version
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -s \
    "https://api.github.com/repos/$REPO/releases/latest" |
    grep tag_name |
    cut -d '"' -f 4)
fi

URL="https://github.com/$REPO/releases/download/$VERSION/${TAR_FILE}"

echo "Downloading $BINARY ($VERSION)..."

curl -fL "$URL" -o "$TAR_FILE"

tar -xzf "$TAR_FILE"

chmod +x "$BINARY"
sudo mv "$BINARY" "$INSTALL_PATH"

rm "$TAR_FILE"

echo "Installed reqlog $VERSION at $INSTALL_PATH"