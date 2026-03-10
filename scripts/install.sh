#!/bin/sh
set -e

echo "Installing GVM (Git Version Manager)..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)       ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    darwin|linux) ;;
    *)
        echo "Error: Unsupported OS: $OS. GVM supports macOS and Linux."
        exit 1
        ;;
esac

# Get latest release version
VERSION=$(curl -sS https://api.github.com/repos/abhikb101/gvm/releases/latest | grep '"tag_name"' | cut -d '"' -f 4)
if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version. Check your internet connection."
    exit 1
fi

DOWNLOAD_URL="https://github.com/abhikb101/gvm/releases/download/${VERSION}/gvm_${OS}_${ARCH}.tar.gz"
echo "Downloading GVM ${VERSION} for ${OS}/${ARCH}..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

HTTP_CODE=$(curl -sL -o "$TMPDIR/gvm.tar.gz" -w '%{http_code}' "$DOWNLOAD_URL")
if [ "$HTTP_CODE" != "200" ]; then
    echo "Error: Download failed (HTTP $HTTP_CODE). URL: $DOWNLOAD_URL"
    exit 1
fi

tar xzf "$TMPDIR/gvm.tar.gz" -C "$TMPDIR"

INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$TMPDIR/gvm" "$INSTALL_DIR/gvm"
else
    mv "$TMPDIR/gvm" "$INSTALL_DIR/gvm"
fi

chmod +x "$INSTALL_DIR/gvm"

echo ""
echo "GVM ${VERSION} installed successfully!"
echo "Run 'gvm init' to get started."
