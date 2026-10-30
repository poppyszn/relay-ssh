#!/usr/bin/env bash
# Build IronRDP WebAssembly module and place in desktop/frontend/src/wasm/
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WASM_DIR="${ROOT_DIR}/desktop/rdp-wasm"
OUT_DIR="${ROOT_DIR}/desktop/frontend/src/wasm"

echo "Building IronRDP WebAssembly..."
if ! command -v wasm-pack >/dev/null 2>&1; then
    echo "wasm-pack not found. Please install wasm-pack (version 0.13.1) or run within CI." >&2
    exit 1
fi

mkdir -p "${OUT_DIR}"
wasm-pack build "${WASM_DIR}" --target web --out-dir "${OUT_DIR}" --out-name ironrdp
echo "IronRDP WebAssembly built successfully in ${OUT_DIR}"
