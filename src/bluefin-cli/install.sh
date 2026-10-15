#!/bin/sh
#-----------------------------------------------------------------------------------------------------------------------
# Bluefin CLI Devcontainer Feature Install Script
#
# Installs the bluefin-cli binary from GitHub releases.
#
# Options (set via devcontainer-feature.json):
#   VERSION  - bluefin-cli version to install (default: latest)
#-----------------------------------------------------------------------------------------------------------------------
set -eu

VERSION="${VERSION:-latest}"
REPO="tuna-os/bluefin-cli"
BINARY="bluefin-cli"
BIN_DIR="/usr/local/bin"

# Resolve the tag to install.
if [ "${VERSION}" = "latest" ]; then
    echo "[bluefin-cli] Resolving latest release..."
    tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
        grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [ -z "${tag}" ]; then
        echo "[bluefin-cli] ERROR: could not determine latest release" >&2
        exit 1
    fi
else
    tag="v${VERSION#v}"
fi

version="${tag#v}"

# Detect OS and architecture.
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "${os}" in
    linux|darwin) ;;
    *)
        echo "[bluefin-cli] ERROR: unsupported OS: ${os}" >&2
        exit 1
        ;;
esac

arch=$(uname -m)
case "${arch}" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)
        echo "[bluefin-cli] ERROR: unsupported architecture: ${arch}" >&2
        exit 1
        ;;
esac

# Download and install.
url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${version}_${os}_${arch}.tar.gz"
archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
checksum_url="https://github.com/${REPO}/releases/download/${tag}/checksums.txt"
echo "[bluefin-cli] Downloading ${BINARY} ${tag} (${os}/${arch})..."
echo "[bluefin-cli] URL: ${url}"

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

curl -fsSL "${url}" -o "${tmp}/bluefin-cli.tar.gz"
curl -fsSL "${checksum_url}" -o "${tmp}/checksums.txt" || {
    echo "[bluefin-cli] ERROR: could not download checksums for ${archive}" >&2
    exit 1
}

expected_sha=$(awk -v archive="${archive}" '$2 == archive { print $1; exit }' "${tmp}/checksums.txt")
if [ "${#expected_sha}" -ne 64 ]; then
    echo "[bluefin-cli] ERROR: checksums.txt has no valid SHA-256 entry for ${archive}" >&2
    exit 1
fi
case "${expected_sha}" in
    *[!0-9A-Fa-f]*)
        echo "[bluefin-cli] ERROR: checksums.txt has no valid SHA-256 entry for ${archive}" >&2
        exit 1
        ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
    actual_sha=$(sha256sum "${tmp}/bluefin-cli.tar.gz" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    actual_sha=$(shasum -a 256 "${tmp}/bluefin-cli.tar.gz" | awk '{print $1}')
else
    echo "[bluefin-cli] ERROR: sha256sum or shasum is required to verify ${archive}" >&2
    exit 1
fi

if [ "${actual_sha}" != "${expected_sha}" ]; then
    echo "[bluefin-cli] ERROR: checksum verification failed for ${archive}" >&2
    exit 1
fi

tar -xzf "${tmp}/bluefin-cli.tar.gz" -C "${tmp}"

if [ ! -f "${tmp}/${BINARY}" ]; then
    echo "[bluefin-cli] ERROR: ${BINARY} not found in archive" >&2
    exit 1
fi

mkdir -p "${BIN_DIR}"
install -m 0755 "${tmp}/${BINARY}" "${BIN_DIR}/${BINARY}"

echo "[bluefin-cli] Installed ${BINARY} ${tag} to ${BIN_DIR}/${BINARY}"
echo "[bluefin-cli] Run 'bluefin-cli' to get started."
