#!/usr/bin/env bash
# Mechanically enforces the v2 invariants. Run by `make check` and by CI.
# These are not style preferences. Each one prevents a specific failure of v1.
set -uo pipefail

fail=0
bad()  { printf '  FAIL  %s\n' "$1"; fail=1; }
good() { printf '  ok    %s\n' "$1"; }

echo "guardrails:"

# The Relay desktop app lives in desktop/ with its own module, dependencies and
# guardrails (desktop/scripts/guardrails.sh). The CLI budgets below cover the CLI
# only, so every scan excludes desktop/.

# 1. Banned imports: TUI frameworks, terminal emulators, SSH-protocol and PTY
#    libraries. v1 died because these were allowed in.
BANNED='charmbracelet|vt10x|tcell|gocui|termbox|golang.org/x/crypto/ssh|golang.org/x/term|creack/pty|gliderlabs/ssh'
hits=$(grep -rInE --exclude-dir='.claude' --exclude-dir='desktop' "\"[a-z0-9./-]*($BANNED)" --include='*.go' . || true)
if [ -n "$hits" ]; then
  echo "$hits"
  bad "banned import (TUI / terminal-emulator / SSH-protocol / PTY library)"
else
  good "no TUI, SSH-protocol, or PTY imports"
fi

# 2. Exactly two direct dependencies.
ndeps=$(go mod edit -json | python3 -c \
  'import json,sys; print(sum(1 for r in json.load(sys.stdin).get("Require") or [] if not r.get("Indirect")))')
if [ "$ndeps" -le 2 ]; then
  good "direct dependencies: $ndeps of 2"
else
  bad "direct dependencies: $ndeps, budget is 2"
fi

# 3. No .go file over 400 lines.
big=$(find . -name '*.go' -not -path './.git/*' -not -path './.claude/*' -not -path './desktop/*' -exec awk 'END{if(NR>400) print FILENAME" ("NR" lines)"}' {} \;)
if [ -n "$big" ]; then
  echo "$big"
  bad "file over the 400-line cap"
else
  good "all files within the 400-line cap"
fi

# 4. Total non-test Go LOC under 2900.
loc=$(find . -name '*.go' -not -name '*_test.go' -not -path './.git/*' -not -path './.claude/*' -not -path './desktop/*' -exec cat {} + | wc -l | tr -d ' ')
if [ "$loc" -le 2900 ]; then
  good "non-test Go lines: $loc of 2900"
else
  bad "non-test Go lines: $loc, budget is 2900"
fi

# 5. syscall.Exec appears exactly once, in internal/launch. More than one call
#    site means something other than the launcher is running processes.
nexec=$(grep -rIn --exclude-dir='.claude' --exclude-dir='desktop' 'syscall\.Exec' --include='*.go' . | grep -v '_test.go' | wc -l | tr -d ' ')
where=$(grep -rIl --exclude-dir='.claude' --exclude-dir='desktop' 'syscall\.Exec' --include='*.go' . | grep -v '_test.go' || true)
if [ "$nexec" = "1" ] && [ "$where" = "./internal/launch/launch.go" ]; then
  good "syscall.Exec confined to internal/launch"
else
  bad "syscall.Exec count=$nexec location=$where (expected 1 in ./internal/launch/launch.go)"
fi

[ "$fail" = "0" ] || { echo; echo "guardrails failed — see docs/superpowers/specs/2026-09-11-sssh-v2-design.md"; exit 1; }
echo "guardrails passed"
