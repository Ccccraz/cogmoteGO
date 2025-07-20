#!/bin/sh

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

printf "${BLUE}=== ${BINARY_NAME} ===${NC}\n"
printf "${YELLOW}Temporary directory: ${TMP_DIR}${NC}\n"

# Detect system architecture
ARCH=$(uname -m)
case "$ARCH" in
    arm64)
        ;;
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    *)
        printf "${RED}Error: Unsupported architecture ${ARCH}${NC}\n"
        exit 1
        ;;
esac
printf "System architecture: ${GREEN}${ARCH}${NC}\n"

# Detect os 
OS=$(uname -s)
case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        printf "${RED}Error: Unsupported OS ${ARCH}${NC}\n"
        exit 1
        ;;
esac
printf "System OS: ${GREEN}${ARCH}${NC}\n"

# Get installed version with 'v' prefix
get_installed_version() {
    if [ -x "$INSTALLED_CMD" ]; then
        echo "v$($INSTALLED_CMD --version 2>/dev/null | awk '{print $2}' | head -1)"
    else
        echo ""
    fi
}

INSTALLED_VERSION=$(get_installed_version)
if [ -n "$INSTALLED_VERSION" ]; then
    printf "Installed version: ${GREEN}${INSTALLED_VERSION}${NC}\n"
fi

# Get latest release
printf "${BLUE}[1/4] ${NC}Checking latest version...\n"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
printf "Latest version: ${GREEN}${LATEST_RELEASE}${NC}\n"

# Check if installation/update is needed
if [ -n "$INSTALLED_VERSION" ]; then
    if [ "$INSTALLED_VERSION" = "$LATEST_RELEASE" ]; then
        printf "${GREEN}Already up to date${NC}\n"
        exit 0
    else
        printf "${YELLOW}New version available ${INSTALLED_VERSION} → ${LATEST_RELEASE}${NC}\n"
    fi
else
    printf "${YELLOW}No installation detected, performing fresh install${NC}\n"
fi

# Build download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}-${OS}-${ARCH}-${LATEST_RELEASE}.tar.gz"
printf "Download URL: ${YELLOW}${DOWNLOAD_URL}${NC}\n"

# Download
printf "${BLUE}[2/4] ${NC}Downloading ${BINARY_NAME}...\n"
curl --progress-bar -L "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME.tar.gz" || {
    printf "${RED}Download failed!${NC}\n"
    rm -rf "$TMP_DIR"
    exit 1
}

# Extract
printf "${BLUE}[3/4] ${NC}Extracting...\n"
tar -xzvf "$TMP_DIR/$BINARY_NAME.tar.gz" -C "$TMP_DIR" || {
    printf "${RED}Extraction failed!${NC}\n"
    rm -rf "$TMP_DIR"
    exit 1
}

# Install
printf "${BLUE}[4/4] ${NC}Installing to ${INSTALL_DIR}...\n"
mkdir -p "$INSTALL_DIR" && \
mv -v "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME" && \
chmod +x "$INSTALL_DIR/$BINARY_NAME" || {
    printf "${RED}Installation failed!${NC}\n"
    rm -rf "$TMP_DIR"
    exit 1
}

# Cleanup
rm -rf "$TMP_DIR"

# Verify installation
NEW_VERSION="v$($INSTALLED_CMD --version 2>/dev/null | awk '{print $2}' | head -1)"
if [ "$NEW_VERSION" = "$LATEST_RELEASE" ]; then
    if [ -n "$INSTALLED_VERSION" ]; then
        printf "${GREEN}Update successful! ${BINARY_NAME} updated from ${INSTALLED_VERSION} to ${NEW_VERSION}${NC}\n"
    else
        printf "${GREEN}Installation successful! ${BINARY_NAME} ${NEW_VERSION} (${ARCH}) installed to ${INSTALL_DIR}${NC}\n"
    fi
else
    printf "${RED}Installation verification failed!${NC}\n"
    exit 1
fi

# Check PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    printf "${YELLOW}Warning: ${INSTALL_DIR} is not in your PATH${NC}\n"
    printf "You can temporarily add it with:\n"
    printf "  ${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}\n"
    printf "Or add it to your ~/.bashrc or ~/.zshrc for permanent access\n"
    ;;
esac
