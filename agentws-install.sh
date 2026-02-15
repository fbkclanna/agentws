#!/bin/sh
# agentws-install.sh — agentws installer
# Usage: curl -fsSL https://github.com/fbkclanna/agentws/releases/latest/download/agentws-install.sh | sh
#
# Environment variables:
#   VERSION     — version to install (e.g. "0.2.0"). Defaults to latest release.
#   INSTALL_DIR — installation directory (default: /usr/local/bin)
#
# Supported platforms: Linux (amd64, arm64), macOS (amd64, arm64)
# Windows is not supported — use "go install" or download from GitHub Releases.

set -eu

REPO="fbkclanna/agentws"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

usage() {
    cat <<EOF
Usage: agentws-install.sh [--help]

Install agentws from GitHub Releases.

Environment variables:
  VERSION      Version to install (default: latest)
  INSTALL_DIR  Installation directory (default: /usr/local/bin)

Examples:
  curl -fsSL https://github.com/$REPO/releases/latest/download/agentws-install.sh | sh
  VERSION=0.2.0 INSTALL_DIR=~/.local/bin curl -fsSL ... | sh
EOF
}

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --help|-h)
            usage
            exit 0
            ;;
    esac
done

# Detect OS
detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux)  echo "linux" ;;
        Darwin) echo "darwin" ;;
        *)
            printf "Error: unsupported OS: %s\n" "$os" >&2
            printf "This installer supports Linux and macOS only.\n" >&2
            printf "For Windows, use 'go install' or download from GitHub Releases.\n" >&2
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            printf "Error: unsupported architecture: %s\n" "$arch" >&2
            exit 1
            ;;
    esac
}

# Detect downloader (curl or wget)
detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        echo "curl"
    elif command -v wget >/dev/null 2>&1; then
        echo "wget"
    else
        printf "Error: neither curl nor wget found. Please install one of them.\n" >&2
        exit 1
    fi
}

# Download a URL to a file
download() {
    url="$1"
    dest="$2"
    downloader="$3"
    case "$downloader" in
        curl) curl -fsSL -o "$dest" "$url" ;;
        wget) wget -q -O "$dest" "$url" ;;
    esac
}

# Fetch content from a URL to stdout
fetch() {
    url="$1"
    downloader="$2"
    case "$downloader" in
        curl) curl -fsSL "$url" ;;
        wget) wget -q -O - "$url" ;;
    esac
}

# Resolve latest version from GitHub API
resolve_latest_version() {
    downloader="$1"
    api_url="https://api.github.com/repos/${REPO}/releases/latest"
    tag="$(fetch "$api_url" "$downloader" | grep '"tag_name"' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/')"
    if [ -z "$tag" ]; then
        printf "Error: failed to determine latest version from GitHub API.\n" >&2
        printf "Please set the VERSION environment variable manually.\n" >&2
        exit 1
    fi
    echo "$tag"
}

main() {
    os="$(detect_os)"
    arch="$(detect_arch)"
    downloader="$(detect_downloader)"

    if [ -n "${VERSION:-}" ]; then
        version="$VERSION"
    else
        printf "Fetching latest version...\n"
        version="$(resolve_latest_version "$downloader")"
    fi

    printf "Platform: %s/%s\n" "$os" "$arch"
    printf "Version:  %s\n" "$version"
    printf "Install:  %s\n" "$INSTALL_DIR"

    archive="agentws_${version}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/v${version}/${archive}"

    # Create temp directory with cleanup trap
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT INT TERM

    printf "Downloading %s...\n" "$archive"
    download "$url" "${tmpdir}/${archive}" "$downloader"

    # Extract
    tar xzf "${tmpdir}/${archive}" -C "$tmpdir"

    # Verify binary exists
    if [ ! -f "${tmpdir}/agentws" ]; then
        printf "Error: agentws binary not found in archive.\n" >&2
        exit 1
    fi

    # Install binary
    mkdir -p "$INSTALL_DIR"
    if [ -w "$INSTALL_DIR" ]; then
        install -m 755 "${tmpdir}/agentws" "${INSTALL_DIR}/agentws"
    else
        printf "Elevated permissions required to install to %s\n" "$INSTALL_DIR"
        sudo install -m 755 "${tmpdir}/agentws" "${INSTALL_DIR}/agentws"
    fi

    printf "\nagentws %s installed to %s/agentws\n" "$version" "$INSTALL_DIR"
    printf "Run 'agentws --version' to verify.\n"
}

main
