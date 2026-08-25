#!/usr/bin/env bash
# Install dango and dango-describe from the rolling `nightly` prerelease.
# Safe to: curl -fsSL https://raw.githubusercontent.com/gsimone/dango-tui/main/install.sh | bash
# No version prompt. Not 0.1.0. linux/amd64 and darwin/arm64 only.
set -euo pipefail

REPO="gsimone/dango-tui"
INSTALL_DIR="${DANGO_INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="dango"
DESCRIBE_NAME="dango-describe"
# Test hook. Product default is the nightly tag.
NIGHTLY_BASE="${DANGO_NIGHTLY_BASE:-https://github.com/${REPO}/releases/download/nightly}"

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

mkdir -p "${INSTALL_DIR}"
tmp_bin="$(mktemp "${TMPDIR:-/tmp}/dango.XXXXXX")"
tmp_desc="$(mktemp "${TMPDIR:-/tmp}/dango-describe.XXXXXX")"
cleanup() { rm -f "${tmp_bin}" "${tmp_desc}"; }
trap cleanup EXIT

download_nightly() {
  local name="$1"
  local dest="$2"
  local url="${NIGHTLY_BASE}/${name}"

  echo "Downloading ${name} from ${REPO} nightly..."
  if ! curl -fsSL --retry 3 --retry-delay 1 -o "${dest}" "${url}"; then
    echo "install.sh: no nightly binary for ${name}." >&2
    echo "See https://github.com/${REPO}/releases/tag/nightly" >&2
    exit 1
  fi
  if [ ! -s "${dest}" ]; then
    echo "install.sh: downloaded file is empty" >&2
    exit 1
  fi
  # GitHub can serve an HTML 404 page without failing the HTTP status.
  if head -c 15 "${dest}" | grep -q '<!DOCTYPE html>'; then
    echo "install.sh: no nightly binary for ${name}." >&2
    echo "See https://github.com/${REPO}/releases/tag/nightly" >&2
    exit 1
  fi
}

# Fetch both first. A missing describe asset must not leave only dango installed.
download_nightly "${asset}" "${tmp_bin}"
download_nightly "${DESCRIBE_NAME}" "${tmp_desc}"

install -m 0755 "${tmp_bin}" "${INSTALL_DIR}/${BIN_NAME}"
install -m 0755 "${tmp_desc}" "${INSTALL_DIR}/${DESCRIBE_NAME}"

echo "Installed ${INSTALL_DIR}/${BIN_NAME}"
echo "Installed ${INSTALL_DIR}/${DESCRIBE_NAME}"
wc -c "${INSTALL_DIR}/${BIN_NAME}" "${INSTALL_DIR}/${DESCRIBE_NAME}"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo "Run: ${BIN_NAME}"
    ;;
  *)
    echo "Add ${INSTALL_DIR} to PATH, then run: ${BIN_NAME}"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
