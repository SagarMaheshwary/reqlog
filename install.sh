#!/usr/bin/env bash

set -e

REPO="sagarmaheshwary/reqlog"
BINARY="reqlog-linux-amd64"
INSTALL_PATH="/usr/local/bin/reqlog"

echo "Installing reqlog..."

LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)

URL="https://github.com/$REPO/releases/download/$LATEST/reqlog-linux-amd64.tar.gz"

curl -L $URL -o reqlog.tar.gz
tar -xzf reqlog.tar.gz

chmod +x $BINARY
sudo mv $BINARY $INSTALL_PATH

echo "Installed reqlog at $INSTALL_PATH"