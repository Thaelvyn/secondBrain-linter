#!/usr/bin/env bash
# sblint installer - fetches the release binary for the current platform.
set -euo pipefail

VERSION=""
if [[ "${1:-}" == "--version" ]]; then
  if [[ $# -lt 2 ]]; then
    echo "install.sh: --version requires a value (e.g. v0.1.0)" >&2
    exit 1
  fi
  VERSION="$2"
fi

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64|Darwin-arm64*)   BIN_NAME="sblint-darwin-arm64" ;;
  Linux-x86_64|Linux-amd64)     BIN_NAME="sblint-linux-amd64" ;;
  *) echo "install.sh: unsupported platform $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac

DEST_DIR="${HOME}/scripts/bin"
DEST="${DEST_DIR}/sblint"
mkdir -p "${DEST_DIR}"

if [[ -n "${VERSION}" ]]; then
  URL="https://github.com/Thaelvyn/secondBrain-linter/releases/download/${VERSION}/${BIN_NAME}"
else
  URL="https://github.com/Thaelvyn/secondBrain-linter/releases/latest/download/${BIN_NAME}"
fi

TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT

echo "install.sh: downloading ${URL}"
curl -fSL "${URL}" -o "${TMP}"
chmod +x "${TMP}"
mv -f "${TMP}" "${DEST}"

echo "install.sh: installed ${DEST}"
"${DEST}" --version
