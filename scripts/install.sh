#!/usr/bin/env bash
# Installs the qdivzero CLI from the latest GitHub release.
# Usage: curl -fsSL https://raw.githubusercontent.com/QDivZero/qdivzero-cli/main/scripts/install.sh | sh
set -euo pipefail

REPO="QDivZero/qdivzero-cli"
INSTALL_DIR="${QDIV0_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)      echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

echo "==> resolving latest release"
API="https://api.github.com/repos/$REPO/releases/latest"
VERSION="$(curl -fsSL "$API" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')"
if [ -z "$VERSION" ]; then
  echo "could not resolve latest release" >&2
  exit 1
fi
echo "==> latest release: $VERSION"

BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
ASSET="qdivzero_${VERSION#v}_${OS}_${ARCH}.tar.gz"

echo "==> downloading $ASSET"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$BASE_URL/$ASSET" -o "$TMP/$ASSET"
curl -fsSL "$BASE_URL/checksums.txt" -o "$TMP/checksums.txt"

echo "==> verifying checksum"
(cd "$TMP" && grep " $ASSET$" checksums.txt | sha256sum -c -)

echo "==> extracting"
tar -xzf "$TMP/$ASSET" -C "$TMP"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP/qdivzero" "$INSTALL_DIR/qdivzero"

echo "==> installed to $INSTALL_DIR/qdivzero"
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "add $INSTALL_DIR to your PATH, then run:"
  echo "  qdivzero configure"
fi
