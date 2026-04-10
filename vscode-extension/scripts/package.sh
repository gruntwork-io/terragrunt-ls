#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

pushd "$SCRIPT_DIR/.." >/dev/null

BINARY_NAME="terragrunt-ls"
if [ "${GOOS:-}" = "windows" ]; then
  BINARY_NAME="terragrunt-ls.exe"
fi

GOOS="${GOOS:-}" GOARCH="${GOARCH:-}" go build -o "out/${BINARY_NAME}" ..
