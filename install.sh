#!/bin/bash

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="Ccccraz/cogmoteGO"
BINARY_NAME="cogmoteGO"
INSTALL_DIR="$HOME/.local/bin"
TMP_DIR=$(mktemp -d -t "${BINARY_NAME}-XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM
INSTALLED_CMD="$INSTALL_DIR/$BINARY_NAME"

echo -e "${BLUE}=== ${BINARY_NAME} ===${NC}"
echo -e "${YELLOW}Temporary directory: ${TMP_DIR}${NC}"

# Detect system architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture ${ARCH}${NC}"
        exit 1
        ;;
esac
echo -e "System architecture: ${GREEN}${ARCH}${NC}"

# Get installed version with 'v' prefix
get_installed_version() {
    if [[ -x "$INSTALLED_CMD" ]]; then
        echo "v$($INSTALLED_CMD --version 2>/dev/null | awk '{print $2}' | head -1)"
    else
        echo ""
    fi
}

INSTALLED_VERSION=$(get_installed_version)
if [[ -n "$INSTALLED_VERSION" ]]; then
    echo -e "Installed version: ${GREEN}${INSTALLED_VERSION}${NC}"
fi

# Get latest release
echo -e "${BLUE}[1/4] ${NC}Checking latest version..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
echo -e "Latest version: ${GREEN}${LATEST_RELEASE}${NC}"

# Check if installation/update is needed
if [[ -n "$INSTALLED_VERSION" ]]; then
    if [[ "$INSTALLED_VERSION" == "$LATEST_RELEASE" ]]; then
        echo -e "${GREEN}Already up to date${NC}"
        exit 0
    else
        echo -e "${YELLOW}New version available ${INSTALLED_VERSION} → ${LATEST_RELEASE}${NC}"
    fi
else
    echo -e "${YELLOW}No installation detected, performing fresh install${NC}"
fi

# Build download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}-linux-${ARCH}-${LATEST_RELEASE}.tar.gz"
echo -e "Download URL: ${YELLOW}${DOWNLOAD_URL}${NC}"

# Download
echo -e "${BLUE}[2/4] ${NC}Downloading ${BINARY_NAME}..."
curl --progress-bar -L "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME.tar.gz" || {
    echo -e "${RED}Download failed!${NC}"
    rm -rf "$TMV_DIR"
    exit 1
}

# Extract
echo -e "${BLUE}[3/4] ${NC}Extracting..."
tar -xzvf "$TMP_DIR/$BINARY_NAME.tar.gz" -C "$TMP_DIR" || {
    echo -e "${RED}Extraction failed!${NC}"
    rm -rf "$TMP_DIR"
    exit 1
}

# Install
echo -e "${BLUE}[4/4] ${NC}Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR" && \
mv -v "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME" && \
chmod +x "$INSTALL_DIR/$BINARY_NAME" || {
    echo -e "${RED}Installation failed!${NC}"
    rm -rf "$TMP_DIR"
    exit 1
}

# Cleanup
rm -rf "$TMP_DIR"

# Verify installation
NEW_VERSION="v$($INSTALLED_CMD --version 2>/dev/null | awk '{print $2}' | head -1)"
if [[ "$NEW_VERSION" == "$LATEST_RELEASE" ]]; then
    if [[ -n "$INSTALLED_VERSION" ]]; then
        echo -e "${GREEN}Update successful! ${BINARY_NAME} updated from ${INSTALLED_VERSION} to ${NEW_VERSION}${NC}"
    else
        echo -e "${GREEN}Installation successful! ${BINARY_NAME} ${NEW_VERSION} (${ARCH}) installed to ${INSTALL_DIR}${NC}"
    fi
else
    echo -e "${RED}Installation verification failed!${NC}"
    exit 1
fi

# Check PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: ${INSTALL_DIR} is not in your PATH${NC}"
    echo -e "You can temporarily add it with:"
    echo -e "  ${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    echo -e "Or add it to your ~/.bashrc or ~/.zshrc for permanent access"
fi