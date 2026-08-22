#!/usr/bin/env bash
set -euo pipefail

REPO="${DANGO_REPO:-gsimone/dango-tui}"
INSTALL_DIR="${DANGO_INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="dango"

os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Linux) os_id=linux ;;
  Darwin) os_id=darwin ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT) os_id=windows ;;
  *)
    echo "install.sh: unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64 | amd64) arch_id=x64 ;;
  arm64 | aarch64) arch_id=arm64 ;;
  *)
    echo "install.sh: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="dango-${os_id}-${arch_id}"
if [ "$os_id" = windows ]; then
  asset="${asset}.exe"
fi

url="https://github.com/${REPO}/releases/latest/download/${asset}"

mkdir -p "$INSTALL_DIR"
tmp="$(mktemp "${TMPDIR:-/tmp}/dango.XXXXXX")"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT

echo "Downloading ${asset} from the latest ${REPO} release..."
if ! curl -fsSL --retry 3 --retry-delay 1 -o "$tmp" "$url"; then
  echo "install.sh: no release binary for ${os_id}-${arch_id} (${asset})." >&2
  echo "See https://github.com/${REPO}/releases/latest for published assets." >&2
  exit 1
fi

if [ ! -s "$tmp" ]; then
  echo "install.sh: downloaded file is empty" >&2
  exit 1
fi

# GitHub sometimes serves an HTML 404 page without failing the HTTP status.
if head -c 15 "$tmp" | grep -q '<!DOCTYPE html>'; then
  echo "install.sh: no release binary for ${os_id}-${arch_id} (${asset})." >&2
  echo "See https://github.com/${REPO}/releases/latest for published assets." >&2
  exit 1
fi

dest="${INSTALL_DIR}/${BIN_NAME}"
install -m 0755 "$tmp" "$dest"

echo "Installed ${dest}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo "Run: ${BIN_NAME}"
    ;;
  *)
    echo "Add ${INSTALL_DIR} to PATH, then run: ${BIN_NAME}"
    ;;
esac
