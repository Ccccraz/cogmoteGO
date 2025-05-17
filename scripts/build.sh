#!/bin/bash

# .SYNOPSIS
#     Build script for Linux/macOS platforms with version embedding and CGO support
# .DESCRIPTION
#     Builds Go project with:
#     - Version info from git (local mode) or provided parameters (CI mode)
#     - CGO enabled with ZMQ library from system
# .PARAMETER CI
#     Indicates this is a CI/CD build (will use provided version/commit instead of git)
# .PARAMETER Version
#     Version to embed in binary (required in CI mode)
# .PARAMETER Commit
#     Commit hash to embed in binary (required in CI mode)
# .EXAMPLE
#     # Local development build
#     ./build.sh
# .EXAMPLE
#     # CI/CD build with specified version
#     ./build.sh --ci --version "v0.0.1" --commit "abc123"

# Initialize variables
CI=false
VERSION=""
COMMIT=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --ci)
            CI=true
            shift
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --commit)
            COMMIT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Determine target platform
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

TARGET="$OS-$ARCH"
ARCHIVE_TYPE="tar.gz"
BINARY_NAME="cogmoteGO"

WORKSPACE=$(dirname "$(dirname "$(realpath "$0")")")
OUTPUT_DIR="$WORKSPACE/build/$TARGET"
DIST_DIR="$WORKSPACE/dist"

# Verify CI mode parameters
if [ "$CI" = true ]; then
    if [ -z "$VERSION" ] || [ -z "$COMMIT" ]; then
        echo -e "\033[31m❌ Need to provide Version and Commit parameters in CI mode\033[0m"
        exit 1
    fi
    echo -e "\033[36m🏗️ Running in CI/CD mode...\033[0m"
fi

# Create necessary directories
mkdir -p "$OUTPUT_DIR" "$DIST_DIR" || {
    echo -e "\033[31m❌ Failed to create directories\033[0m"
    exit 1
}

# Get build info
function get_version_info {
    if [ "$CI" = true ]; then
        # CI mode: use provided version info
        CLEAN_VERSION=$(echo "$VERSION" | sed -E 's/^v([0-9.]+).*/\1/')
        echo "$CLEAN_VERSION $COMMIT"
    else
        # Local mode: get version info from git
        GIT_DESC=$(git describe --tags 2>/dev/null)
        if [ $? -ne 0 ] || [ -z "$GIT_DESC" ]; then
            echo "dev none"
        else
            CLEAN_VERSION=$(echo "$GIT_DESC" | sed -E 's/^v([0-9.]+).*/\1/')
            COMMIT_SHORT=$(git rev-parse --short HEAD)
            echo "$CLEAN_VERSION $COMMIT_SHORT"
        fi
    fi
}

read -r VERSION COMMIT <<< "$(get_version_info)"
DATETIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-w \
-X 'github.com/Ccccraz/cogmoteGO/cmd.version=$VERSION' \
-X 'github.com/Ccccraz/cogmoteGO/cmd.commit=$COMMIT' \
-X 'github.com/Ccccraz/cogmoteGO/cmd.datetime=$DATETIME'"

# Print build info
echo -e "\n\033[36m🔧 Setup CGO environment...\033[0m"
echo -e "\033[90m├─ \$Env:CGO_ENABLED = \"1\"\033[0m"
echo -e "\033[90m└─ Using system ZMQ library\033[0m"

echo -e "\n\033[36m🚀 Start building cogmoteGO...\033[0m"
echo -e "\033[36m├─ version: $VERSION\033[0m"
echo -e "\033[36m├─ commit: $COMMIT\033[0m"
echo -e "\033[36m├─ datetime: $DATETIME\033[0m"
echo -e "\033[36m├─ target: $TARGET\033[0m"

cd "$WORKSPACE" || exit 1

# Enable CGO and build
export CGO_ENABLED=1
go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/$BINARY_NAME" "$WORKSPACE"

if [ $? -ne 0 ]; then
    echo -e "\033[31m❌ Build failed\033[0m"
    exit 1
fi

# Package archive
ARCHIVE_NAME="cogmoteGO-$TARGET-v$VERSION.$ARCHIVE_TYPE"
tar -czf "$DIST_DIR/$ARCHIVE_NAME" -C "$OUTPUT_DIR" "$BINARY_NAME"

echo -e "\033[32m└─ ✅ build successful!\033[0m"
echo -e "   \033[32m├─ binary file: $OUTPUT_DIR/$BINARY_NAME\033[0m"
echo -e "   \033[32m└─ dist: $DIST_DIR/$ARCHIVE_NAME\033[0m"