#!/usr/bin/env bash
# Audits the Rust dependencies of the IronRDP client Relay ships: the WebAssembly client (ironrdp-web), not the
# whole IronRDP workspace, whose lockfile also holds native clients and servers (DNS, zstd, zlib build scripts)
# that never reach us. An advisory fails the build when its crate is in the client's wasm dependency tree and is
# not in desktop/rdp-client/audit-allow.txt, which says why each accepted one is accepted.
# Needs cargo-audit and the source from `make rdp-client`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT/desktop/.ironrdp-src"
ALLOW="$ROOT/desktop/rdp-client/audit-allow.txt"
PIN="$ROOT/desktop/rdp-client/ironrdp.pin"

command -v cargo-audit >/dev/null 2>&1 || { echo "cargo-audit not found (cargo install --version 0.22.2 cargo-audit --locked)" >&2; exit 1; }
[ -f "$SRC/Cargo.lock" ] || { echo "$SRC is missing; run make rdp-client first" >&2; exit 1; }
export RUSTUP_TOOLCHAIN="$(sed -n 's/^rust=//p' "$PIN")"

REPORT="$(mktemp)"
trap 'rm -f "$REPORT"' EXIT
cargo audit --file "$SRC/Cargo.lock" --json > "$REPORT" || true   # exits 1 when it finds vulnerabilities

python3 - "$REPORT" "$SRC" "$ALLOW" <<'PY'
import json, subprocess, sys

report, src, allow_file = sys.argv[1:4]
found = json.load(open(report))
allowed = {}
for line in open(allow_file):
    line = line.strip()
    if line and not line.startswith("#"):
        advisory, _, reason = line.partition(" ")
        allowed[advisory] = reason.strip()

def shipped(crate):
    """Is the crate in the wasm client's dependency tree?"""
    r = subprocess.run(["cargo", "tree", "-p", "ironrdp-web", "--target", "wasm32-unknown-unknown", "--offline", "-i", crate],
                       cwd=src, capture_output=True, text=True)
    return r.returncode == 0 and r.stdout.strip() != ""

failed = False
for v in found.get("vulnerabilities", {}).get("list", []):
    advisory, crate, version = v["advisory"]["id"], v["package"]["name"], v["package"]["version"]
    title = v["advisory"]["title"]
    if not shipped(crate):
        print(f"  skip  {advisory} {crate} {version}: not in the WebAssembly client ({title})")
    elif advisory in allowed:
        print(f"  ok    {advisory} {crate} {version}: accepted: {allowed[advisory][:100]}")
    else:
        print(f"  FAIL  {advisory} {crate} {version}: in the WebAssembly client and not accepted ({title})")
        failed = True
warnings = found.get("warnings", {})
print("  note  %d warnings (unmaintained or unsound crates) in the workspace; not blocking" % sum(len(x) for x in warnings.values()))
sys.exit(1 if failed else 0)
PY
