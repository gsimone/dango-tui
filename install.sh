#!/usr/bin/env bash
# Install dango from the rolling `nightly` prerelease.
# Safe to: curl -fsSL https://raw.githubusercontent.com/gsimone/dango-tui/main/install.sh | bash
# No version prompt. Not 0.1.0. linux/amd64 and darwin/arm64 only.
set -euo pipefail

REPO="gsimone/dango-tui"
INSTALL_DIR="${DANGO_INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="dango"

os="$(uname -s)"
arch="$(uname -m)"

case "${os}-${arch}" in
  Linux-x86_64 | Linux-amd64)
    asset="dango-linux-amd64"
    ;;
  Darwin-arm64 | Darwin-aarch64)
    asset="dango-darwin-arm64"
    ;;
  *)
    echo "install.sh: unsupported platform ${os}/${arch}." >&2
    echo "Need linux/amd64 or darwin/arm64." >&2
    exit 1
    ;;
esac

url="https://github.com/${REPO}/releases/download/nightly/${asset}"

mkdir -p "${INSTALL_DIR}"
tmp="$(mktemp "${TMPDIR:-/tmp}/dango.XXXXXX")"
cleanup() { rm -f "${tmp}"; }
trap cleanup EXIT

echo "Downloading ${asset} from ${REPO} nightly..."
if ! curl -fsSL --retry 3 --retry-delay 1 -o "${tmp}" "${url}"; then
  echo "install.sh: no nightly binary for ${asset}." >&2
  echo "See https://github.com/${REPO}/releases/tag/nightly" >&2
  exit 1
fi

if [ ! -s "${tmp}" ]; then
  echo "install.sh: downloaded file is empty" >&2
  exit 1
fi

# GitHub can serve an HTML 404 page without failing the HTTP status.
if head -c 15 "${tmp}" | grep -q '<!DOCTYPE html>'; then
  echo "install.sh: no nightly binary for ${asset}." >&2
  echo "See https://github.com/${REPO}/releases/tag/nightly" >&2
  exit 1
fi

dest="${INSTALL_DIR}/${BIN_NAME}"
install -m 0755 "${tmp}" "${dest}"

echo "Installed ${dest}"
wc -c "${dest}"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo "Run: ${BIN_NAME}"
    ;;
  *)
    echo "Add ${INSTALL_DIR} to PATH, then run: ${BIN_NAME}"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
