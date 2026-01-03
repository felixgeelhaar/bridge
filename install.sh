#!/bin/bash
#
# Bridge CLI Installer
# Usage: curl -sSL https://raw.githubusercontent.com/felixgeelhaar/bridge/main/install.sh | bash
#
# Environment variables:
#   BRIDGE_VERSION  - Specific version to install (default: latest)
#   BRIDGE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#

set -euo pipefail

# Configuration
REPO="felixgeelhaar/bridge"
BINARY_NAME="bridge"
INSTALL_DIR="${BRIDGE_INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)   OS="linux" ;;
        darwin)  OS="darwin" ;;
        msys*|mingw*|cygwin*) OS="windows" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest version from GitHub
get_latest_version() {
    if [ -n "${BRIDGE_VERSION:-}" ]; then
        VERSION="$BRIDGE_VERSION"
        info "Using specified version: $VERSION"
    else
        info "Fetching latest version..."
        VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version"
        fi
        info "Latest version: $VERSION"
    fi
}

# Download and install
install_binary() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"
    local checksum_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    if [ "$OS" = "windows" ]; then
        download_url="${download_url}.exe"
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    info "Downloading ${BINARY_NAME} ${VERSION}..."
    if ! curl -sSL -o "${tmp_dir}/${BINARY_NAME}" "$download_url"; then
        error "Failed to download binary from $download_url"
    fi

    # Verify checksum if available
    info "Verifying checksum..."
    if curl -sSL -o "${tmp_dir}/checksums.txt" "$checksum_url" 2>/dev/null; then
        local expected_checksum
        expected_checksum=$(grep "${BINARY_NAME}-${PLATFORM}" "${tmp_dir}/checksums.txt" | awk '{print $1}')
        if [ -n "$expected_checksum" ]; then
            local actual_checksum
            if command -v sha256sum &> /dev/null; then
                actual_checksum=$(sha256sum "${tmp_dir}/${BINARY_NAME}" | awk '{print $1}')
            elif command -v shasum &> /dev/null; then
                actual_checksum=$(shasum -a 256 "${tmp_dir}/${BINARY_NAME}" | awk '{print $1}')
            else
                warn "No checksum tool found, skipping verification"
                actual_checksum="$expected_checksum"
            fi

            if [ "$expected_checksum" != "$actual_checksum" ]; then
                error "Checksum verification failed!"
            fi
            success "Checksum verified"
        fi
    else
        warn "Checksums not available, skipping verification"
    fi

    # Install binary
    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    chmod +x "${tmp_dir}/${BINARY_NAME}"

    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Requesting sudo for installation..."
        sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    success "Installed ${BINARY_NAME} ${VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        success "Installation complete!"
        echo ""
        echo "Run 'bridge --help' to get started"
        echo ""
        "$BINARY_NAME" --version 2>/dev/null || true
    else
        warn "Installation complete, but '${BINARY_NAME}' is not in your PATH"
        echo "Add ${INSTALL_DIR} to your PATH:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
}

# Alternative installation methods
show_alternatives() {
    echo ""
    echo "Alternative installation methods:"
    echo ""
    echo "  # Homebrew (macOS/Linux)"
    echo "  brew install felixgeelhaar/tap/bridge"
    echo ""
    echo "  # Go install"
    echo "  go install github.com/felixgeelhaar/bridge/cmd/bridge@latest"
    echo ""
    echo "  # Docker"
    echo "  docker run --rm ghcr.io/felixgeelhaar/bridge:latest --help"
    echo ""
}

main() {
    echo ""
    echo "  ____       _     _            "
    echo " | __ ) _ __(_) __| | __ _  ___ "
    echo " |  _ \\| '__| |/ _\` |/ _\` |/ _ \\"
    echo " | |_) | |  | | (_| | (_| |  __/"
    echo " |____/|_|  |_|\\__,_|\\__, |\\___|"
    echo "                     |___/      "
    echo ""
    echo " AI Workflow Orchestration & Governance"
    echo ""

    detect_platform
    get_latest_version
    install_binary
    verify_installation
    show_alternatives
}

main "$@"
