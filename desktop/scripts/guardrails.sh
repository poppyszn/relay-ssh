#!/usr/bin/env bash
# Guardrails for the Relay desktop module. The CLI's own budgets live in
# ../scripts/guardrails.sh and deliberately do not cover desktop/.
set -uo pipefail
cd "$(dirname "$0")/.."
fail=0
bad()  { printf '  FAIL  %s\n' "$1"; fail=1; }
good() { printf '  ok    %s\n' "$1"; }
echo "desktop guardrails:"

# Go must never interpret terminal output: no VT parsers, cell grids or TUI
# frameworks. This is what killed v1. xterm.js in the webview does the rendering.
BANNED='vt10x|charmbracelet|tcell|gocui|termbox|hinshun/vt10x|ActiveState/vt10x'
hits=$(grep -rInE "\"[a-z0-9./-]*($BANNED)" --include='*.go' --exclude-dir=node_modules --exclude-dir=build . || true)
if [ -n "$hits" ]; then echo "$hits"; bad "terminal-emulation or TUI import in Go"; else good "no terminal emulation in Go"; fi

# Terminal bytes are shipped with flow control: the pump must wait on inFlight.
if grep -q 'highWater' session.go && grep -q '"ack"' session.go; then good "flow control present (highWater + ack)"; else bad "flow control missing from session.go"; fi

# The socket must stay on loopback and be token- and Origin-guarded.
if grep -q '127.0.0.1:0' terminal.go && grep -q 'allowedOrigins' terminal.go && grep -q 'Token' terminal.go; then
  good "terminal socket is loopback + token + origin guarded"
else bad "terminal socket guard missing"; fi

[ "$fail" = "0" ] || { echo "desktop guardrails failed"; exit 1; }
echo "desktop guardrails passed"
