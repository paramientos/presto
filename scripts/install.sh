#!/bin/bash

# Presto Installer for macOS and Linux
set -e

REPO="paramientos/presto"
BINARY_NAME="presto"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Normalize Architecture
case "${ARCH}" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Construct download URL
TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": "\([^"]*\)".*/\1/')
if [ -z "$TAG" ]; then
    TAG="latest"
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY_NAME}-${OS}-${ARCH}"

echo "🎵 Downloading Presto ${TAG} for ${OS}/${ARCH}..."
curl -L "${URL}" -o "${BINARY_NAME}"

chmod +x "${BINARY_NAME}"

# Move to /usr/local/bin
if [ -w "/usr/local/bin" ]; then
    mv "${BINARY_NAME}" "/usr/local/bin/"
else
    echo "📥 Need sudo permissions to install to /usr/local/bin"
    sudo mv "${BINARY_NAME}" "/usr/local/bin/"
fi

echo "✨ Presto installed successfully! Run 'presto --version' to verify."
