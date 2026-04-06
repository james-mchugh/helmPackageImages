#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="${HELM_PLUGIN_DIR}"
VERSION="${HELM_PLUGIN_VERSION:-0.2.0}"
REPO="james-mchugh/helmPackageImages"
BIN_DIR="${PLUGIN_DIR}/bin"
mkdir -p "${BIN_DIR}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64)         ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

TARBALL="helm-package-images_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${TARBALL}"

echo "Installing helm-package-images v${VERSION} for ${OS}/${ARCH}..."

if curl -fsSL "${URL}" -o /tmp/helm-package-images.tar.gz 2>/dev/null; then
  tar -xzf /tmp/helm-package-images.tar.gz -C "${BIN_DIR}" helm-package-images
  chmod +x "${BIN_DIR}/helm-package-images"
  rm -f /tmp/helm-package-images.tar.gz
  echo "Installed to ${BIN_DIR}/helm-package-images"
else
  echo "Download failed. Attempting to build from source (requires Go)..."
  if ! command -v go &>/dev/null; then
    echo "Error: 'go' not found. Install Go from https://go.dev or download a pre-built release from:" >&2
    echo "  https://github.com/${REPO}/releases" >&2
    exit 1
  fi
  cd "${PLUGIN_DIR}"
  go build -o "${BIN_DIR}/helm-package-images" .
  echo "Built and installed to ${BIN_DIR}/helm-package-images"
fi
