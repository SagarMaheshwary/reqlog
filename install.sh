#!/usr/bin/env bash

set -e

REPO="sagarmaheshwary/reqlog"
INSTALL_PATH="/usr/local/bin/reqlog"

echo "Installing reqlog..."

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

LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)

URL="https://github.com/$REPO/releases/download/$LATEST/${TAR_FILE}"

echo "Downloading $BINARY..."

curl -L "$URL" -o "$TAR_FILE"
tar -xzf "$TAR_FILE"

chmod +x "$BINARY"
sudo mv "$BINARY" "$INSTALL_PATH"

rm "$TAR_FILE"

echo "Installed reqlog at $INSTALL_PATH"