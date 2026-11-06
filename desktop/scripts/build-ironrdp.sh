#!/usr/bin/env bash
# Builds IronRDP's browser client (a WebAssembly module plus its web component and RDP backend, all
# official Devolutions code) at the commit pinned in desktop/rdp-client/ironrdp.pin, and copies the
# result to desktop/frontend/vendor/ironrdp/ (not committed). Needs git, node, rustup and wasm-pack.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PIN="$ROOT/desktop/rdp-client/ironrdp.pin"
SRC="$ROOT/desktop/.ironrdp-src"
OUT="$ROOT/desktop/frontend/vendor/ironrdp"

pinned() { sed -n "s/^$1=//p" "$PIN"; }
REF="$(pinned ref)"; RUST="$(pinned rust)"
[ -n "$REF" ] && [ -n "$RUST" ] || { echo "$PIN is missing ref or rust" >&2; exit 1; }
case "$REF" in *[!0-9a-f]* | "") echo "ref in $PIN must be a full commit hash" >&2; exit 1 ;; esac
[ "${#REF}" = 40 ] || { echo "ref in $PIN must be a full 40-character commit hash" >&2; exit 1; }

for tool in git node npm rustup wasm-pack; do
  command -v "$tool" >/dev/null 2>&1 || { echo "$tool not found. Install it first (macOS: brew install rustup wasm-pack, then rustup install $RUST)." >&2; exit 1; }
done

[ -d "$SRC/.git" ] || git init -q "$SRC"
git -C "$SRC" fetch -q --depth 1 https://github.com/Devolutions/IronRDP "$REF"
git -C "$SRC" checkout -q --detach FETCH_HEAD
ACTUAL="$(git -C "$SRC" rev-parse HEAD)"
[ "$ACTUAL" = "$REF" ] || { echo "fetched $ACTUAL, not the pinned $REF. Refusing to build." >&2; exit 1; }

echo "Building IronRDP $REF with Rust $RUST ..."
rustup toolchain install "$RUST" --profile minimal --target wasm32-unknown-unknown >/dev/null
export RUSTUP_TOOLCHAIN="$RUST"
(
  cd "$SRC/crates/ironrdp-web"
  # The flags IronRDP's own `cargo xtask web build` uses for a release build.
  RUSTFLAGS='-Ctarget-feature=+simd128,+bulk-memory --cfg getrandom_backend="wasm_js"' wasm-pack build --target web --release
  # Vite cannot handle `new URL(..., import.meta.url)`; IronRDP's xtask rewrites it to a ?url import, so do the same.
  js=pkg/ironrdp_web.js
  if ! grep -q "wasmUrl" "$js"; then
    { printf "import wasmUrl from './ironrdp_web_bg.wasm?url';\n\n"; sed "s#new URL('ironrdp_web_bg.wasm', import.meta.url)#wasmUrl#" "$js"; } > "$js.tmp"
    mv "$js.tmp" "$js"
  fi
  if grep -q "new URL('ironrdp_web_bg.wasm'" "$js"; then echo "could not rewrite the wasm URL in $js" >&2; exit 1; fi
)

(cd "$SRC/web-client/iron-remote-desktop" && npm ci --ignore-scripts --no-audit --no-fund && npm run build)
(cd "$SRC/web-client/iron-remote-desktop-rdp" && npm ci --ignore-scripts --no-audit --no-fund && npm run build-alone)

rm -rf "$OUT" && mkdir -p "$OUT"
cp "$SRC/web-client/iron-remote-desktop/dist/iron-remote-desktop.js" "$SRC/web-client/iron-remote-desktop/dist/index.d.ts" "$OUT/"
mv "$OUT/index.d.ts" "$OUT/iron-remote-desktop.d.ts"
cp "$SRC/web-client/iron-remote-desktop-rdp/dist/iron-remote-desktop-rdp.js" "$OUT/"
cp "$SRC/web-client/iron-remote-desktop-rdp/dist/index.d.ts" "$OUT/iron-remote-desktop-rdp.d.ts"
cp "$SRC/LICENSE-MIT" "$SRC/LICENSE-APACHE" "$OUT/"
printf '%s\n' "$REF" > "$OUT/VERSION"
echo "IronRDP client built into $OUT"
