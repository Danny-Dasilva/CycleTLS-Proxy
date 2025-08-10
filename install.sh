#!/bin/bash

# CycleTLS-Proxy Installation Script
# Supports Linux and macOS with auto-detection of OS and architecture
# Downloads latest release from GitHub and installs to /usr/local/bin/

set -euo pipefail

# Configuration
REPO_OWNER="Danny-Dasilva"
REPO_NAME="CycleTLS-Proxy"
BINARY_NAME="cycletls-proxy"
DEFAULT_INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
VERSION=""
INSTALL_DIR=""
GITHUB_TOKEN=""

# Print functions
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

print_usage() {
    cat << EOF
CycleTLS-Proxy Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -v, --version VERSION    Install specific version (default: latest)
    -d, --dir DIRECTORY     Install directory (default: ${DEFAULT_INSTALL_DIR})
    -t, --token TOKEN       GitHub personal access token for private repos
    -h, --help             Show this help message

EXAMPLES:
    $0                      # Install latest version to ${DEFAULT_INSTALL_DIR}
    $0 -v v1.2.3           # Install specific version
    $0 -d ~/bin            # Install to custom directory
    $0 -t ghp_xxx...       # Use GitHub token for authentication

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -t|--token)
                GITHUB_TOKEN="$2"
                shift 2
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    # Set defaults
    INSTALL_DIR="${INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""

    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            print_info "This script supports Linux and macOS only"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        armv7l|armv6l)
            arch="arm"
            ;;
        i386|i686)
            arch="386"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            print_info "Supported architectures: amd64, arm64, arm, 386"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Check if required tools are available
check_dependencies() {
    local missing_deps=()

    for cmd in curl tar; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        print_info "Please install the missing dependencies and try again"
        exit 1
    fi
}

# Get latest release version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local auth_header=""

    if [[ -n "$GITHUB_TOKEN" ]]; then
        auth_header="-H \"Authorization: token $GITHUB_TOKEN\""
    fi

    print_info "Fetching latest release information..."

    local response
    if ! response=$(eval "curl -s $auth_header \"$api_url\""); then
        print_error "Failed to fetch release information from GitHub API"
        exit 1
    fi

    # Extract tag name using basic text processing (avoiding jq dependency)
    local version
    if ! version=$(echo "$response" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4); then
        print_error "Failed to parse release information"
        print_info "Response: $response"
        exit 1
    fi

    if [[ -z "$version" ]]; then
        print_error "Could not determine latest version"
        print_info "Please specify a version manually with -v flag"
        exit 1
    fi

    echo "$version"
}

# Download and verify binary
download_binary() {
    local version="$1"
    local platform="$2"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${BINARY_NAME}_${version}_${platform}.tar.gz"
    local temp_dir
    local auth_header=""

    if [[ -n "$GITHUB_TOKEN" ]]; then
        auth_header="-H \"Authorization: token $GITHUB_TOKEN\""
    fi

    temp_dir=$(mktemp -d)
    local archive_file="${temp_dir}/${BINARY_NAME}.tar.gz"

    print_info "Downloading ${BINARY_NAME} ${version} for ${platform}..."
    print_info "URL: $download_url"

    # Download the archive
    if ! eval "curl -L -o \"$archive_file\" $auth_header \"$download_url\""; then
        print_error "Failed to download ${BINARY_NAME} from ${download_url}"
        print_info "Please check if the release exists and try again"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Verify download
    if [[ ! -f "$archive_file" ]] || [[ ! -s "$archive_file" ]]; then
        print_error "Downloaded file is empty or missing"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Extract archive
    print_info "Extracting archive..."
    if ! tar -xzf "$archive_file" -C "$temp_dir"; then
        print_error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Find binary (it might be in a subdirectory)
    local binary_path
    if [[ -f "${temp_dir}/${BINARY_NAME}" ]]; then
        binary_path="${temp_dir}/${BINARY_NAME}"
    elif [[ -f "${temp_dir}/bin/${BINARY_NAME}" ]]; then
        binary_path="${temp_dir}/bin/${BINARY_NAME}"
    else
        # Search for any executable file
        binary_path=$(find "$temp_dir" -type f -executable -name "*${BINARY_NAME}*" | head -1)
        if [[ -z "$binary_path" ]]; then
            print_error "Could not find ${BINARY_NAME} binary in the archive"
            print_info "Archive contents:"
            find "$temp_dir" -type f
            rm -rf "$temp_dir"
            exit 1
        fi
    fi

    echo "$binary_path"
}

# Install binary to destination
install_binary() {
    local binary_path="$1"
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    # Create install directory if it doesn't exist
    if [[ ! -d "$INSTALL_DIR" ]]; then
        print_info "Creating install directory: $INSTALL_DIR"
        if ! mkdir -p "$INSTALL_DIR"; then
            print_error "Failed to create install directory: $INSTALL_DIR"
            print_info "You might need to run this script with sudo or choose a different directory"
            exit 1
        fi
    fi

    # Check if we can write to the install directory
    if [[ ! -w "$INSTALL_DIR" ]]; then
        print_error "No write permission to install directory: $INSTALL_DIR"
        print_info "You might need to run this script with sudo or choose a different directory"
        exit 1
    fi

    # Install binary
    print_info "Installing ${BINARY_NAME} to ${install_path}..."
    if ! cp "$binary_path" "$install_path"; then
        print_error "Failed to copy binary to $install_path"
        exit 1
    fi

    # Make executable
    if ! chmod +x "$install_path"; then
        print_error "Failed to make binary executable"
        exit 1
    fi

    print_success "${BINARY_NAME} installed successfully to ${install_path}"
}

# Verify installation
verify_installation() {
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    print_info "Verifying installation..."

    # Check if binary exists and is executable
    if [[ ! -x "$install_path" ]]; then
        print_error "Binary is not executable: $install_path"
        exit 1
    fi

    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "Install directory $INSTALL_DIR is not in your PATH"
        print_info "Add the following line to your shell profile (.bashrc, .zshrc, etc.):"
        print_info "export PATH=\"$INSTALL_DIR:\$PATH\""
    fi

    # Try to run the binary to check version
    print_info "Testing binary..."
    if "$install_path" --version &>/dev/null || "$install_path" -v &>/dev/null || "$install_path" version &>/dev/null; then
        print_success "Binary is working correctly"
    else
        print_warning "Binary installed but could not verify functionality"
        print_info "You can try running: $install_path --help"
    fi
}

# Cleanup function
cleanup() {
    if [[ -n "${temp_dir:-}" ]] && [[ -d "$temp_dir" ]]; then
        rm -rf "$temp_dir"
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Main execution
main() {
    echo "CycleTLS-Proxy Installation Script"
    echo "=================================="
    echo

    # Parse arguments
    parse_args "$@"

    # Check dependencies
    check_dependencies

    # Detect platform
    local platform
    platform=$(detect_platform)
    print_info "Detected platform: $platform"

    # Get version
    if [[ -z "$VERSION" ]]; then
        VERSION=$(get_latest_version)
    fi
    print_info "Installing version: $VERSION"

    # Download binary
    local binary_path
    binary_path=$(download_binary "$VERSION" "$platform")

    # Install binary
    install_binary "$binary_path"

    # Verify installation
    verify_installation

    echo
    print_success "Installation completed successfully!"
    print_info "Run '${BINARY_NAME} --help' to get started"

    # Cleanup happens automatically via trap
}

# Only run main if script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi