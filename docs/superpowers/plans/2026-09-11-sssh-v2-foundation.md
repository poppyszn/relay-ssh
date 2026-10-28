# super-ssh v2 Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `sssh` — a connection vault that generates OpenSSH config, launches the real `ssh` binary via `exec`, and backs everything up into one encrypted file.

**Architecture:** `connections.toml` is the source of truth. Every mutation regenerates `~/.ssh/sssh.d/config`, which the user's `~/.ssh/config` pulls in with a single `Include` line inside marked comments. Connecting means `syscall.Exec` of `ssh <profile-name>` — the generated config already carries the key, jump host, tunnels, and auth settings, so there are no flags to assemble and no Go code left running once the session starts.

**Tech Stack:** Go 1.27, `github.com/BurntSushi/toml` v1.6.0, `filippo.io/age` v1.3.2. Nothing else. `archive/tar`, `compress/gzip`, `os/exec`, `syscall`, and `flag` come from the standard library.

**Reference spec:** `docs/superpowers/specs/2026-09-11-sssh-v2-design.md`

## Global Constraints

Every task's requirements implicitly include this section. Values are exact — do not substitute.

- **Module path:** `github.com/poppyszn/super-ssh-v2`
- **Binary name:** `sssh`
- **Go version:** `go 1.27.0` in `go.mod`
- **Direct dependencies: exactly two.** `github.com/BurntSushi/toml v1.6.0` and `filippo.io/age v1.3.2`. Adding a third fails CI.
- **Banned imports (CI-enforced):** anything matching `charmbracelet`, `vt10x`, `tcell`, `gocui`, `termbox`, `golang.org/x/crypto/ssh`, `golang.org/x/term`, `creack/pty`, `gliderlabs/ssh`. These are TUI frameworks, SSH-protocol implementations, and PTY libraries — the three things that made v1 unusable.
- **File size cap:** 400 lines per `.go` file, tests included. CI-enforced.
- **Total non-test Go LOC cap:** 2500. CI-enforced.
- **`syscall.Exec` appears exactly once in the repo**, in `internal/launch/launch.go`. CI-enforced.
- **Platforms:** macOS and Linux only. Windows is out of scope; `syscall.Exec` does not exist there.
- **Tests are hermetic.** Every test that touches the filesystem builds its own root with `t.TempDir()` and constructs `paths.New(dir)`. No test reads or writes the real `$HOME`. No test opens a network connection. No test runs `ssh`.
- **Every file write that can corrupt user state is atomic:** write `<target>.tmp-<random>` in the same directory, `fsync`, then `os.Rename`. This applies to `connections.toml`, `~/.ssh/sssh.d/config`, and `~/.ssh/config`.
- **Permissions:** directories `0700`, private keys `0600`, `connections.toml` `0600`, generated config `0600`.
- **Commit style:** Conventional Commits (`feat:`, `fix:`, `test:`, `docs:`, `chore:`). Commit at the end of every task.
- **Error text:** all user-facing errors print to stderr as `sssh: <what failed>: <cause>` and exit non-zero.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `cmd/sssh/main.go` | Argument dispatch only. No business logic. |
| `cmd/sssh/cmd_*.go` | One file per subcommand; each parses its own flags and calls into `internal/`. |
| `internal/paths/paths.go` | Derives every filesystem location from a single root. The only place a path is spelled out. |
| `internal/config/types.go` | `Profile`, `Tunnel`, `Store` types and their TOML tags. |
| `internal/config/validate.go` | All validation rules. |
| `internal/config/store.go` | `Load` / `Save` with atomic write. |
| `internal/sshconfig/generate.go` | `Store` → ssh config text. Pure function, no I/O. |
| `internal/sshconfig/include.go` | The marked `Include` block in `~/.ssh/config`. |
| `internal/keys/keys.go` | Adopting key material into the sandbox, permission enforcement. |
| `internal/launch/launch.go` | argv construction and the single `syscall.Exec`. |
| `internal/pick/pick.go` | fzf detection and the numbered fallback. |
| `internal/importer/importer.go` | Parsing an existing `~/.ssh/config`. |
| `internal/backup/archive.go` | tar + gzip + age. |
| `internal/backup/restore.go` | Extraction, sandbox safety, `known_hosts` merge. |
| `internal/doctor/doctor.go` | Health checks. |
| `scripts/guardrails.sh` | Mechanically enforces the budgets above. |
| `CLAUDE.md` | The rules future sessions must follow. |

---

### Task 1: Scaffold, guardrails, and CI

This task exists to make every later task's constraints machine-checked. The guardrail script is not documentation — it fails the build.

**Files:**
- Create: `go.mod`, `.gitignore`, `Makefile`, `CLAUDE.md`, `scripts/guardrails.sh`, `.github/workflows/ci.yml`, `cmd/sssh/main.go`, `cmd/sssh/main_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `make check` (the verification gate every later task runs), and a compiling `sssh` binary that responds to `version`.

- [ ] **Step 1: Initialise the module and pin both dependencies**

```bash
cd /Users/precious/Code/personal/super-ssh-v2
go mod init github.com/poppyszn/super-ssh-v2
go get github.com/BurntSushi/toml@v1.6.0
go get filippo.io/age@v1.3.2
```

Confirm `go.mod` contains `go 1.27.0`. If `go mod init` wrote a bare `go 1.27`, edit it to `go 1.27.0`.

- [ ] **Step 2: Write `.gitignore`**

```gitignore
/sssh
/dist/
*.tmp-*
*.sssh
.DS_Store
```

- [ ] **Step 3: Write `scripts/guardrails.sh`**

```bash
#!/usr/bin/env bash
# Mechanically enforces the v2 invariants. Run by `make check` and by CI.
# These are not style preferences. Each one prevents a specific failure of v1.
set -uo pipefail

fail=0
bad()  { printf '  FAIL  %s\n' "$1"; fail=1; }
good() { printf '  ok    %s\n' "$1"; }

echo "guardrails:"

# 1. Banned imports: TUI frameworks, terminal emulators, SSH-protocol and PTY
#    libraries. v1 died because these were allowed in.
BANNED='charmbracelet|vt10x|tcell|gocui|termbox|golang.org/x/crypto/ssh|golang.org/x/term|creack/pty|gliderlabs/ssh'
hits=$(grep -rInE "\"[a-z0-9./-]*($BANNED)" --include='*.go' . || true)
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
big=$(find . -name '*.go' -not -path './.git/*' -exec awk 'END{if(NR>400) print FILENAME" ("NR" lines)"}' {} \;)
if [ -n "$big" ]; then
  echo "$big"
  bad "file over the 400-line cap"
else
  good "all files within the 400-line cap"
fi

# 4. Total non-test Go LOC under 2500.
loc=$(find . -name '*.go' -not -name '*_test.go' -not -path './.git/*' -exec cat {} + | wc -l | tr -d ' ')
if [ "$loc" -le 2500 ]; then
  good "non-test Go lines: $loc of 2500"
else
  bad "non-test Go lines: $loc, budget is 2500"
fi

# 5. syscall.Exec appears exactly once, in internal/launch. More than one call
#    site means something other than the launcher is running processes.
nexec=$(grep -rIn 'syscall\.Exec' --include='*.go' . | grep -v '_test.go' | wc -l | tr -d ' ')
where=$(grep -rIl 'syscall\.Exec' --include='*.go' . | grep -v '_test.go' || true)
if [ "$nexec" = "1" ] && [ "$where" = "./internal/launch/launch.go" ]; then
  good "syscall.Exec confined to internal/launch"
else
  bad "syscall.Exec count=$nexec location=$where (expected 1 in ./internal/launch/launch.go)"
fi

[ "$fail" = "0" ] || { echo; echo "guardrails failed — see docs/superpowers/specs/2026-09-11-sssh-v2-design.md"; exit 1; }
echo "guardrails passed"
```

Then: `chmod +x scripts/guardrails.sh`

- [ ] **Step 4: Write the `Makefile`**

Note the literal tab characters required by make.

```makefile
BINARY := sssh
PKG    := ./cmd/sssh

.PHONY: help build install test race vet fmt guardrails check clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the sssh binary
	go build -o $(BINARY) $(PKG)

install: ## Install sssh into $$GOPATH/bin
	go install $(PKG)

test: ## Run the test suite
	go test ./...

race: ## Run the test suite with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Check formatting
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

guardrails: ## Enforce the architectural budgets
	@./scripts/guardrails.sh

check: fmt vet race guardrails ## Everything CI runs. This is the gate before any "done".

clean: ## Remove build output
	rm -f $(BINARY)

.DEFAULT_GOAL := help
```

- [ ] **Step 5: Write `cmd/sssh/main.go`**

```go
// Command sssh manages SSH connection profiles and launches the real ssh
// binary. It is never running while a session is open: connecting replaces
// this process via exec.
package main

import (
	"fmt"
	"os"
)

// version is the tool version, overridden at release time with
// -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "sssh:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command given (try `sssh help`)")
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("sssh " + version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}
	return fmt.Errorf("unknown command %q (try `sssh help`)", args[0])
}

const usage = `sssh — SSH connection manager

Usage:
  sssh                    pick a connection and connect
  sssh <name>             connect to a saved profile
  sssh ls [--tree]        list profiles
  sssh add                add a profile
  sssh edit <name>        edit a profile
  sssh rm <name>          remove a profile
  sssh cp <name>:/p ./l   copy files with scp
  sssh import             adopt hosts from ~/.ssh/config
  sssh backup [-o FILE]   write an encrypted archive
  sssh restore FILE       restore from an encrypted archive
  sssh doctor             check the installation
  sssh sync               regenerate ~/.ssh/sssh.d/config
  sssh version            print the version
`
```

- [ ] **Step 6: Write the failing test `cmd/sssh/main_test.go`**

```go
package main

import "testing"

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"wibble"})
	if err == nil {
		t.Fatal("expected an error for an unknown command, got nil")
	}
	want := `unknown command "wibble" (try ` + "`sssh help`)"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunNoArgs(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("expected an error when no command is given, got nil")
	}
}

func TestRunVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		if err := run([]string{arg}); err != nil {
			t.Fatalf("run(%q) = %v, want nil", arg, err)
		}
	}
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		if err := run([]string{arg}); err != nil {
			t.Fatalf("run(%q) = %v, want nil", arg, err)
		}
	}
}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./cmd/sssh/ -v`
Expected: all four PASS. (The implementation was written in Step 5 because a scaffold task has nothing to red-green against; TDD proper begins in Task 2.)

- [ ] **Step 8: Write `.github/workflows/ci.yml`**

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:

jobs:
  check:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27.0'
          cache: true
      - run: make check
```

- [ ] **Step 9: Write `CLAUDE.md`**

This is the guardrail document. It is read by every future session.

````markdown
# super-ssh v2 — working agreement

Read this before changing anything. It encodes why v1 was abandoned.

## What this tool is

A connection vault that generates OpenSSH config and then gets out of the way.
Three jobs, in order of importance:

1. One encrypted file holds profiles and keys, so a new machine is a restore away.
2. Connections are organised and visible instead of buried in `ssh_config`.
3. Jump hosts, tunnels, `scp`, and agent forwarding are saved profile properties,
   not flags you have to remember.

It is not a terminal, a multiplexer, or an SSH client.

## The three invariants

Violating one of these is a defect, not a trade-off.

1. **sssh is never running while you are connected.** It builds an argv and
   `exec`s `ssh`, replacing its own process image. No child process, no pipes,
   no PTY, no bytes flowing through Go.
2. **sssh owns `~/.ssh/sssh.d/` and one marked block in `~/.ssh/config`.**
   Every write outside that sandbox is a bug. The single exception is
   `known_hosts`, which is merged and never overwritten.
3. **The source of truth is a file a human can repair with `vim`.**

## Why v1 failed

v1 vendored forks of `bubbletea` and `vt10x` and emulated a terminal in-process.
Every byte of remote output went through a VT parser, a cell grid, a style
engine, and a full-screen diff. `cat` of a large file ran that loop thousands of
times a second. Copying text was slow, long output stalled sessions, the machine
ran hot. Roughly 12,000 of its 29,000 lines existed to serve a goal the user
never had.

Every symptom traced to one cause: sssh was in the data path. Invariant 1 makes
that structurally impossible. Do not weaken it.

## Before adding any feature, answer these

- **Does it require sssh to be running while a session is open?**
  If yes, the answer is no. This rules out tabs, panes, splits, scrollback,
  text selection, mouse support, session lists, and connection multiplexing —
  permanently, not "not yet".
- **Can OpenSSH already do this through `ssh_config`?**
  If yes, it is a line in `generate.go`, not a subsystem. Jump hosts and port
  forwarding cost v1 about 1,400 lines; here they are two `fmt.Fprintf` calls.
- **Does it need a new dependency?**
  The budget is two. Spend one only if the alternative is implementing crypto
  or a parser for a format you do not control.
- **Does it store a secret?**
  It should not. Passwords, passphrases, and 2FA codes are prompted for by
  `ssh` itself. There is no keychain, no unlock screen, no encrypted field.

## Budgets, enforced by `scripts/guardrails.sh` in CI

| Budget | Limit |
| --- | --- |
| Direct dependencies | 2 |
| Lines per `.go` file | 400 |
| Non-test Go lines, whole repo | 2500 |
| `syscall.Exec` call sites | 1, in `internal/launch` |
| TUI / SSH-protocol / PTY imports | 0 |

Raising a budget requires changing the spec first and saying why in the commit
message. Silently bumping a number in `guardrails.sh` defeats the entire point
of this file.

## How to work

- **TDD.** Write the failing test, watch it fail, write the smallest code that
  passes, watch it pass, commit. A test you never saw fail proves nothing.
- **Spec before code** for anything that adds a command or changes the data
  model. Specs live in `docs/superpowers/specs/`, plans in
  `docs/superpowers/plans/`.
- **Hermetic tests.** Every filesystem test builds its own root with
  `t.TempDir()` and passes it to `paths.New`. Never touch the real `$HOME`.
  Never open a network connection. Never invoke `ssh`.
- **Atomic writes.** Anything that could leave the user locked out of their own
  hosts — `connections.toml`, the generated config, `~/.ssh/config` — is
  written to a temp file in the same directory and renamed.
- **Conventional commits**, committed per task.

## The gate before claiming anything is done

```bash
make check
```

`fmt`, `vet`, `test -race`, and the guardrails. Run it and read the output.
Do not report work as complete on the strength of having written it.
````

- [ ] **Step 10: Run the full gate**

Run: `make check`
Expected: `fmt`, `vet`, and `race` pass; guardrails prints five `ok` lines and `guardrails passed`.

If guardrail 5 fails with `count=0`, that is correct at this point — `internal/launch` does not exist yet. Temporarily relax check 5 to accept `count=0` by changing its condition to `if { [ "$nexec" = "1" ] && [ "$where" = "./internal/launch/launch.go" ]; } || [ "$nexec" = "0" ]; then`, and remove the `|| [ "$nexec" = "0" ]` clause in Task 8 once `launch.go` exists.

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum .gitignore Makefile CLAUDE.md scripts/ .github/ cmd/
git commit -m "chore: scaffold sssh v2 with CI-enforced architectural budgets"
```

---

### Task 2: Paths and atomic writes

Two pieces of infrastructure with no dependencies of their own. `paths` is the only place a filesystem location is spelled out, which is what makes every later test hermetic. `fsutil` is the atomic write used by everything that touches user state.

**Files:**
- Create: `internal/paths/paths.go`, `internal/paths/paths_test.go`, `internal/fsutil/fsutil.go`, `internal/fsutil/fsutil_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `paths.Paths` struct with fields `Root, ConfigDir, StoreFile, SSHDir, SSHConfig, Sandbox, GenConfig, KeysDir, KnownHosts` — all `string`.
  - `paths.New(root string) Paths`
  - `paths.Default() (Paths, error)`
  - `paths.EnsureDirs(p Paths) error`
  - `fsutil.WriteFile(path string, data []byte, mode os.FileMode) error`

- [ ] **Step 1: Write the failing test `internal/paths/paths_test.go`**

```go
package paths

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDerivesEveryLocationFromRoot(t *testing.T) {
	p := New("/home/u")
	cases := map[string]string{
		"ConfigDir":  "/home/u/.config/sssh",
		"StoreFile":  "/home/u/.config/sssh/connections.toml",
		"SSHDir":     "/home/u/.ssh",
		"SSHConfig":  "/home/u/.ssh/config",
		"Sandbox":    "/home/u/.ssh/sssh.d",
		"GenConfig":  "/home/u/.ssh/sssh.d/config",
		"KeysDir":    "/home/u/.ssh/sssh.d/keys",
		"KnownHosts": "/home/u/.ssh/known_hosts",
	}
	got := map[string]string{
		"ConfigDir": p.ConfigDir, "StoreFile": p.StoreFile, "SSHDir": p.SSHDir,
		"SSHConfig": p.SSHConfig, "Sandbox": p.Sandbox, "GenConfig": p.GenConfig,
		"KeysDir": p.KeysDir, "KnownHosts": p.KnownHosts,
	}
	for field, want := range cases {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
}

func TestEnsureDirsCreatesEverythingAt0700(t *testing.T) {
	p := New(t.TempDir())
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, dir := range []string{p.ConfigDir, p.SSHDir, p.Sandbox, p.KeysDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
		if perm := info.Mode().Perm(); perm != fs.FileMode(0o700) {
			t.Errorf("%s mode = %o, want 700", dir, perm)
		}
	}
}

func TestEnsureDirsIsIdempotent(t *testing.T) {
	p := New(t.TempDir())
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("first EnsureDirs: %v", err)
	}
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("second EnsureDirs: %v", err)
	}
}

func TestDefaultHonoursSSSHHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SSSH_HOME", dir)
	p, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if p.Root != dir {
		t.Errorf("Root = %q, want %q", p.Root, dir)
	}
	if p.StoreFile != filepath.Join(dir, ".config", "sssh", "connections.toml") {
		t.Errorf("StoreFile = %q", p.StoreFile)
	}
}

func TestDefaultFallsBackToHomeDir(t *testing.T) {
	t.Setenv("SSSH_HOME", "")
	p, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if p.Root == "" {
		t.Error("Root is empty")
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/paths/ -v`
Expected: FAIL — the package does not exist yet (`no Go files in .../internal/paths`).

- [ ] **Step 3: Write `internal/paths/paths.go`**

```go
// Package paths derives every filesystem location sssh touches from a single
// root directory. Nothing else in the codebase spells out a path, which is what
// lets tests run against t.TempDir() and never see the real home directory.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths is the complete set of locations sssh reads or writes.
//
// Everything from Sandbox down is owned by sssh and may be rewritten freely.
// SSHConfig is shared with the user: only the marked Include block inside it
// belongs to sssh. KnownHosts is shared too, and is only ever merged.
type Paths struct {
	Root       string // the home directory this tree hangs off
	ConfigDir  string // <root>/.config/sssh
	StoreFile  string // <root>/.config/sssh/connections.toml -- the source of truth
	SSHDir     string // <root>/.ssh
	SSHConfig  string // <root>/.ssh/config       -- shared; only the marked block is ours
	Sandbox    string // <root>/.ssh/sssh.d       -- entirely ours
	GenConfig  string // <root>/.ssh/sssh.d/config
	KeysDir    string // <root>/.ssh/sssh.d/keys
	KnownHosts string // <root>/.ssh/known_hosts  -- shared; merged, never overwritten
}

// New builds the path set for a given home directory.
//
// The config directory is $HOME/.config/sssh on every platform. This
// deliberately does not use os.UserConfigDir, which on macOS returns
// ~/Library/Application Support -- a location that is awkward to back up by
// hand and surprising to anyone who expects dotfile conventions.
func New(root string) Paths {
	cfg := filepath.Join(root, ".config", "sssh")
	ssh := filepath.Join(root, ".ssh")
	sandbox := filepath.Join(ssh, "sssh.d")
	return Paths{
		Root:       root,
		ConfigDir:  cfg,
		StoreFile:  filepath.Join(cfg, "connections.toml"),
		SSHDir:     ssh,
		SSHConfig:  filepath.Join(ssh, "config"),
		Sandbox:    sandbox,
		GenConfig:  filepath.Join(sandbox, "config"),
		KeysDir:    filepath.Join(sandbox, "keys"),
		KnownHosts: filepath.Join(ssh, "known_hosts"),
	}
}

// Default uses $SSSH_HOME when set, otherwise the user's home directory.
// SSSH_HOME exists so the tool can be exercised end to end against a throwaway
// tree without risking the real ~/.ssh.
func Default() (Paths, error) {
	if root := os.Getenv("SSSH_HOME"); root != "" {
		return New(root), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("locating home directory: %w", err)
	}
	return New(home), nil
}

// EnsureDirs creates every directory sssh writes into, at 0700. It is safe to
// call repeatedly.
func EnsureDirs(p Paths) error {
	for _, dir := range []string{p.ConfigDir, p.SSHDir, p.Sandbox, p.KeysDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("setting permissions on %s: %w", dir, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the paths tests**

Run: `go test ./internal/paths/ -v`
Expected: all five PASS.

- [ ] **Step 5: Write the failing test `internal/fsutil/fsutil_test.go`**

```go
package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileCreatesWithMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.toml")
	if err := WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", data, "hello")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestWriteFileOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.toml")
	if err := os.WriteFile(path, []byte("old content that is long"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("content = %q, want %q", data, "new")
	}
}

// The temp file must be created in the target's own directory, not in /tmp,
// so that the final rename is atomic rather than a cross-device copy.
func TestWriteFileLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(filepath.Join(dir, "out.toml"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want 1", len(entries))
	}
}

func TestWriteFileFailsOnMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "out.toml")
	if err := WriteFile(path, []byte("x"), 0o600); err == nil {
		t.Fatal("expected an error writing into a missing directory, got nil")
	}
}
```

- [ ] **Step 6: Run it and confirm it fails**

Run: `go test ./internal/fsutil/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 7: Write `internal/fsutil/fsutil.go`**

```go
// Package fsutil provides the atomic file write used for every piece of state
// whose partial write would lock the user out of their own hosts.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data to path atomically.
//
// It creates a temp file in the target's own directory -- not /tmp, so the
// final step is a same-filesystem rename rather than a copy -- sets the
// requested mode, fsyncs, and renames over the target. A crash at any point
// leaves either the old file or the new one, never a truncated mixture.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op once the rename has succeeded

	if err := f.Chmod(mode); err != nil {
		f.Close()
		return fmt.Errorf("setting permissions on %s: %w", tmp, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("syncing %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}
```

- [ ] **Step 8: Run both packages**

Run: `go test ./internal/... -v`
Expected: all tests PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/paths internal/fsutil
git commit -m "feat: add path derivation and atomic file writes"
```

---

### Task 3: Profile types and validation

All validation lives here so there is exactly one definition of a valid profile. Note that the name rules are stricter than ssh requires — that is deliberate and explained in the code comments.

**Files:**
- Create: `internal/config/types.go`, `internal/config/validate.go`, `internal/config/validate_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `config.AuthType` with constants `AuthAgent` (`"agent"`), `AuthKey` (`"key"`), `AuthPassword` (`"password"`)
  - `config.TunnelType` with constants `TunnelLocal` (`"local"`), `TunnelRemote` (`"remote"`), `TunnelDynamic` (`"dynamic"`)
  - `config.Tunnel` struct: `Type TunnelType`, `Listen string`, `Target string`
  - `config.Profile` struct: `Name, Host, User string`, `Port int`, `Auth AuthType`, `Key, Folder string`, `Tags []string`, `Jump string`, `ForwardAgent bool`, `Tunnels []Tunnel`
  - `config.Store` struct: `Version int`, `Profiles []Profile`
  - `config.Version` constant = `1`
  - `(Profile) Validate() error`, `(Tunnel) Validate() error`, `(Store) Validate() error`
  - `(Store) Find(name string) (Profile, bool)`, `(Store) Names() []string`

- [ ] **Step 1: Write `internal/config/types.go`**

```go
// Package config defines the connection store: the TOML file that is the single
// source of truth for every profile sssh knows about.
package config

// Version is the schema version written into connections.toml. Load refuses a
// file with a different version rather than guessing at its meaning.
const Version = 1

// AuthType selects which authentication directives get generated.
//
// No secret is ever stored. A profile records only which method it uses; ssh
// prompts for passwords, key passphrases, and 2FA codes itself.
type AuthType string

const (
	// AuthAgent relies on the running ssh-agent. Generates no directives.
	AuthAgent AuthType = "agent"
	// AuthKey uses a specific private key file.
	AuthKey AuthType = "key"
	// AuthPassword asks ssh to prefer interactive password authentication.
	AuthPassword AuthType = "password"
)

// TunnelType selects which port-forwarding directive gets generated.
type TunnelType string

const (
	TunnelLocal   TunnelType = "local"   // LocalForward
	TunnelRemote  TunnelType = "remote"  // RemoteForward
	TunnelDynamic TunnelType = "dynamic" // DynamicForward (SOCKS proxy)
)

// Tunnel becomes one forwarding line in the generated config. Target is unused
// for dynamic tunnels, which only need something to listen on.
type Tunnel struct {
	Type   TunnelType `toml:"type"`
	Listen string     `toml:"listen"`
	Target string     `toml:"target,omitempty"`
}

// Profile is one saved connection.
//
// Name doubles as the ssh Host alias and as the argument to `sssh <name>`,
// which is why Validate constrains it so tightly.
//
// Key is either a bare filename, meaning a key adopted into
// ~/.ssh/sssh.d/keys/, or an absolute or ~-prefixed path, meaning a key linked
// in place and therefore not covered by backups.
//
// Jump names another profile, never a free-text host, so a jump host gets the
// same key and port handling as any other connection.
type Profile struct {
	Name         string   `toml:"name"`
	Host         string   `toml:"host"`
	User         string   `toml:"user"`
	Port         int      `toml:"port"`
	Auth         AuthType `toml:"auth"`
	Key          string   `toml:"key,omitempty"`
	Folder       string   `toml:"folder,omitempty"`
	Tags         []string `toml:"tags,omitempty"`
	Jump         string   `toml:"jump,omitempty"`
	ForwardAgent bool     `toml:"forward_agent,omitempty"`
	Tunnels      []Tunnel `toml:"tunnel,omitempty"`
}

// Store is the whole of connections.toml.
type Store struct {
	Version  int       `toml:"version"`
	Profiles []Profile `toml:"profile"`
}

// Find returns the profile with the given name.
func (s Store) Find(name string) (Profile, bool) {
	for _, p := range s.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

// Names returns every profile name in the order the profiles are stored, which
// Save keeps sorted.
func (s Store) Names() []string {
	out := make([]string, 0, len(s.Profiles))
	for _, p := range s.Profiles {
		out = append(out, p.Name)
	}
	return out
}
```

- [ ] **Step 2: Write the failing test `internal/config/validate_test.go`**

```go
package config

import "testing"

// valid returns a profile that passes validation, so each test can mutate one
// field and assert on that field alone.
func valid() Profile {
	return Profile{
		Name: "prod-web", Host: "10.0.0.5", User: "deploy",
		Port: 22, Auth: AuthAgent,
	}
}

func TestProfileValidateAcceptsAGoodProfile(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestProfileValidateRejectsBadFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Profile)
	}{
		{"empty name", func(p *Profile) { p.Name = "" }},
		{"name with a space", func(p *Profile) { p.Name = "prod web" }},
		{"name with a star", func(p *Profile) { p.Name = "web*" }},
		{"name with a question mark", func(p *Profile) { p.Name = "web?" }},
		{"name with a bang", func(p *Profile) { p.Name = "!web" }},
		{"name with a hash", func(p *Profile) { p.Name = "web#1" }},
		{"reserved name ls", func(p *Profile) { p.Name = "ls" }},
		{"reserved name backup", func(p *Profile) { p.Name = "backup" }},
		{"empty host", func(p *Profile) { p.Host = "" }},
		{"empty user", func(p *Profile) { p.User = "" }},
		{"port zero", func(p *Profile) { p.Port = 0 }},
		{"port negative", func(p *Profile) { p.Port = -1 }},
		{"port too large", func(p *Profile) { p.Port = 65536 }},
		{"unknown auth", func(p *Profile) { p.Auth = "magic" }},
		{"empty auth", func(p *Profile) { p.Auth = "" }},
		{"key auth without a key", func(p *Profile) { p.Auth = AuthKey; p.Key = "" }},
		{"agent auth with a key", func(p *Profile) { p.Auth = AuthAgent; p.Key = "id_ed25519" }},
		{"folder with a leading slash", func(p *Profile) { p.Folder = "/work" }},
		{"folder with a trailing slash", func(p *Profile) { p.Folder = "work/" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid()
			tt.mutate(&p)
			if err := p.Validate(); err == nil {
				t.Fatalf("Validate() = nil, want an error for %s", tt.name)
			}
		})
	}
}

func TestProfileValidateAcceptsGoodNames(t *testing.T) {
	for _, name := range []string{"web", "prod-web", "web.example.com", "db_1", "A1"} {
		p := valid()
		p.Name = name
		if err := p.Validate(); err != nil {
			t.Errorf("name %q rejected: %v", name, err)
		}
	}
}

func TestTunnelValidate(t *testing.T) {
	tests := []struct {
		name    string
		tunnel  Tunnel
		wantErr bool
	}{
		{"good local", Tunnel{TunnelLocal, "5432", "db:5432"}, false},
		{"good remote", Tunnel{TunnelRemote, "8080", "localhost:80"}, false},
		{"good dynamic", Tunnel{TunnelDynamic, "1080", ""}, false},
		{"local without target", Tunnel{TunnelLocal, "5432", ""}, true},
		{"remote without target", Tunnel{TunnelRemote, "8080", ""}, true},
		{"dynamic with target", Tunnel{TunnelDynamic, "1080", "db:5432"}, true},
		{"no listen", Tunnel{TunnelLocal, "", "db:5432"}, true},
		{"unknown type", Tunnel{"sideways", "1", "a:1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tunnel.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreValidateRejectsDuplicateNames(t *testing.T) {
	a, b := valid(), valid()
	s := Store{Version: Version, Profiles: []Profile{a, b}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for duplicate names, got nil")
	}
}

func TestStoreValidateRejectsUnknownJumpTarget(t *testing.T) {
	p := valid()
	p.Jump = "ghost"
	s := Store{Version: Version, Profiles: []Profile{p}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for an unknown jump target, got nil")
	}
}

func TestStoreValidateAcceptsAJumpChain(t *testing.T) {
	a, b, c := valid(), valid(), valid()
	a.Name, b.Name, c.Name = "a", "b", "c"
	a.Jump, b.Jump = "b", "c"
	s := Store{Version: Version, Profiles: []Profile{a, b, c}}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestStoreValidateRejectsJumpCycles(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		a := valid()
		a.Name, a.Jump = "a", "a"
		s := Store{Version: Version, Profiles: []Profile{a}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a self-referencing jump, got nil")
		}
	})
	t.Run("two hop", func(t *testing.T) {
		a, b := valid(), valid()
		a.Name, b.Name = "a", "b"
		a.Jump, b.Jump = "b", "a"
		s := Store{Version: Version, Profiles: []Profile{a, b}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a two-hop jump cycle, got nil")
		}
	})
	t.Run("three hop", func(t *testing.T) {
		a, b, c := valid(), valid(), valid()
		a.Name, b.Name, c.Name = "a", "b", "c"
		a.Jump, b.Jump, c.Jump = "b", "c", "a"
		s := Store{Version: Version, Profiles: []Profile{a, b, c}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a three-hop jump cycle, got nil")
		}
	})
}

func TestStoreValidateRejectsWrongVersion(t *testing.T) {
	s := Store{Version: 99, Profiles: []Profile{valid()}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for an unsupported version, got nil")
	}
}

func TestStoreFindAndNames(t *testing.T) {
	a, b := valid(), valid()
	a.Name, b.Name = "alpha", "beta"
	s := Store{Version: Version, Profiles: []Profile{a, b}}

	got, ok := s.Find("beta")
	if !ok {
		t.Fatal("Find(beta) not found")
	}
	if got.Name != "beta" {
		t.Errorf("Find(beta).Name = %q", got.Name)
	}
	if _, ok := s.Find("ghost"); ok {
		t.Error("Find(ghost) = found, want not found")
	}

	names := s.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names() = %v, want [alpha beta]", names)
	}
}
```

- [ ] **Step 3: Run it and confirm it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `Validate` is undefined (`types.go` exists but `validate.go` does not).

- [ ] **Step 4: Write `internal/config/validate.go`**

```go
package config

import (
	"fmt"
	"strings"
)

// reservedNames are the subcommands. A profile may not take one of these,
// because `sssh <name>` dispatches on the same argument position: a profile
// called "backup" would be unreachable, and would shadow the backup command.
var reservedNames = map[string]bool{
	"ls": true, "add": true, "edit": true, "rm": true, "cp": true,
	"import": true, "backup": true, "restore": true, "doctor": true,
	"sync": true, "version": true, "help": true,
}

// validNameChar allows letters, digits, '.', '_' and '-'.
//
// This is stricter than ssh itself requires, on purpose. ssh treats '*', '?'
// and '!' as pattern metacharacters, so a profile named "web*" would generate a
// Host block silently matching hosts the user never configured -- including,
// potentially, every host they have. '#' would start a comment mid-line. A
// space would split the alias in two. Rejecting the whole class is safer than
// enumerating the dangerous ones.
func validNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '.', r == '_', r == '-':
		return true
	}
	return false
}

// Validate checks a single profile in isolation. Cross-profile rules -- unique
// names, jump targets resolving, jump chains terminating -- live on Store.
func (p Profile) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if reservedNames[p.Name] {
		return fmt.Errorf("name %q is reserved for the sssh subcommand of the same name", p.Name)
	}
	for _, r := range p.Name {
		if !validNameChar(r) {
			return fmt.Errorf("profile %q: name contains %q; only letters, digits, '.', '_' and '-' are allowed", p.Name, string(r))
		}
	}
	if p.Host == "" {
		return fmt.Errorf("profile %q: host is required", p.Name)
	}
	if p.User == "" {
		return fmt.Errorf("profile %q: user is required", p.Name)
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("profile %q: port must be between 1 and 65535, got %d", p.Name, p.Port)
	}
	switch p.Auth {
	case AuthAgent, AuthPassword:
		if p.Key != "" {
			return fmt.Errorf("profile %q: key is set but auth is %q; use auth = \"key\" or clear the key", p.Name, p.Auth)
		}
	case AuthKey:
		if p.Key == "" {
			return fmt.Errorf("profile %q: key is required when auth is %q", p.Name, AuthKey)
		}
	default:
		return fmt.Errorf("profile %q: unknown auth %q (want \"agent\", \"key\" or \"password\")", p.Name, p.Auth)
	}
	if strings.HasPrefix(p.Folder, "/") || strings.HasSuffix(p.Folder, "/") {
		return fmt.Errorf("profile %q: folder %q must not begin or end with '/'", p.Name, p.Folder)
	}
	for i, t := range p.Tunnels {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("profile %q, tunnel %d: %w", p.Name, i+1, err)
		}
	}
	return nil
}

// Validate checks one forwarding rule.
func (t Tunnel) Validate() error {
	switch t.Type {
	case TunnelLocal, TunnelRemote:
		if t.Listen == "" {
			return fmt.Errorf("listen is required")
		}
		if t.Target == "" {
			return fmt.Errorf("target is required for a %s tunnel", t.Type)
		}
	case TunnelDynamic:
		if t.Listen == "" {
			return fmt.Errorf("listen is required")
		}
		if t.Target != "" {
			return fmt.Errorf("a dynamic tunnel must not set target (it is a SOCKS proxy, not a fixed route)")
		}
	default:
		return fmt.Errorf("unknown tunnel type %q (want \"local\", \"remote\" or \"dynamic\")", t.Type)
	}
	return nil
}

// Validate checks every profile, then the rules that span profiles.
func (s Store) Validate() error {
	if s.Version != Version {
		return fmt.Errorf("unsupported store version %d (this sssh understands version %d)", s.Version, Version)
	}
	seen := make(map[string]bool, len(s.Profiles))
	for _, p := range s.Profiles {
		if err := p.Validate(); err != nil {
			return err
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate profile name %q", p.Name)
		}
		seen[p.Name] = true
	}
	for _, p := range s.Profiles {
		if p.Jump != "" && !seen[p.Jump] {
			return fmt.Errorf("profile %q: jump target %q is not a saved profile", p.Name, p.Jump)
		}
	}
	return s.checkJumpCycles()
}

// checkJumpCycles walks each profile's jump chain. Without this, a cycle makes
// ssh recurse until it exhausts file descriptors, with an error that points
// nowhere near the real cause.
func (s Store) checkJumpCycles() error {
	jump := make(map[string]string, len(s.Profiles))
	for _, p := range s.Profiles {
		jump[p.Name] = p.Jump
	}
	for _, p := range s.Profiles {
		visited := map[string]bool{p.Name: true}
		for cur := p.Jump; cur != ""; cur = jump[cur] {
			if visited[cur] {
				return fmt.Errorf("profile %q: jump chain forms a cycle through %q", p.Name, cur)
			}
			visited[cur] = true
		}
	}
	return nil
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/config/ -v`
Expected: every test PASSES, including all nineteen `TestProfileValidateRejectsBadFields` subtests.

- [ ] **Step 6: Commit**

```bash
git add internal/config
git commit -m "feat: add profile types and validation"
```

---

### Task 4: Loading and saving the store

**Files:**
- Create: `internal/config/store.go`, `internal/config/store_test.go`

**Interfaces:**
- Consumes: `config.Store` and `Validate` (Task 3), `fsutil.WriteFile` (Task 2).
- Produces: `config.Load(path string) (Store, error)`, `config.Save(path string, s Store) error`

- [ ] **Step 1: Write the failing test `internal/config/store_test.go`**

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileYieldsEmptyStore(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "connections.toml"))
	if err != nil {
		t.Fatalf("Load: %v, want nil (a missing file is a first run, not an error)", err)
	}
	if s.Version != Version {
		t.Errorf("Version = %d, want %d", s.Version, Version)
	}
	if len(s.Profiles) != 0 {
		t.Errorf("Profiles = %d, want 0", len(s.Profiles))
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	want := Store{Version: Version, Profiles: []Profile{
		{
			Name: "prod-web", Host: "10.0.0.5", User: "deploy", Port: 2222,
			Auth: AuthKey, Key: "prod-web", Folder: "work/prod",
			Tags: []string{"web", "prod"}, Jump: "bastion", ForwardAgent: true,
			Tunnels: []Tunnel{{TunnelLocal, "5432", "db.internal:5432"}},
		},
		{Name: "bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: AuthAgent},
	}}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Profiles) != 2 {
		t.Fatalf("loaded %d profiles, want 2", len(got.Profiles))
	}
	p, ok := got.Find("prod-web")
	if !ok {
		t.Fatal("prod-web not found after round trip")
	}
	if p.Host != "10.0.0.5" || p.User != "deploy" || p.Port != 2222 {
		t.Errorf("host/user/port = %s/%s/%d", p.Host, p.User, p.Port)
	}
	if p.Auth != AuthKey || p.Key != "prod-web" {
		t.Errorf("auth/key = %s/%s", p.Auth, p.Key)
	}
	if p.Folder != "work/prod" || p.Jump != "bastion" || !p.ForwardAgent {
		t.Errorf("folder/jump/forwardagent = %s/%s/%v", p.Folder, p.Jump, p.ForwardAgent)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "web" {
		t.Errorf("tags = %v", p.Tags)
	}
	if len(p.Tunnels) != 1 || p.Tunnels[0].Type != TunnelLocal || p.Tunnels[0].Target != "db.internal:5432" {
		t.Errorf("tunnels = %+v", p.Tunnels)
	}
}

func TestSaveSortsProfilesByName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	s := Store{Version: Version, Profiles: []Profile{
		{Name: "zulu", Host: "z", User: "u", Port: 22, Auth: AuthAgent},
		{Name: "alpha", Host: "a", User: "u", Port: 22, Auth: AuthAgent},
	}}
	if err := Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if names := got.Names(); names[0] != "alpha" || names[1] != "zulu" {
		t.Errorf("Names() = %v, want [alpha zulu]", names)
	}
}

func TestSaveWritesAt0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	if err := Save(path, Store{Version: Version}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestSaveRejectsAnInvalidStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	s := Store{Version: Version, Profiles: []Profile{{Name: "no-host", User: "u", Port: 22, Auth: AuthAgent}}}
	if err := Save(path, s); err == nil {
		t.Fatal("expected Save to reject a profile with no host, got nil")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Save wrote a file despite rejecting the store")
	}
}

func TestLoadRejectsMalformedTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	if err := os.WriteFile(path, []byte("this is not = = toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for malformed TOML, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the file", err)
	}
}

func TestLoadRejectsAnInvalidStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	body := "version = 1\n\n[[profile]]\n  name = \"a\"\n  host = \"h\"\n  user = \"u\"\n  port = 22\n  auth = \"key\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for key auth with no key, got nil")
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/config/ -run 'TestLoad|TestSave' -v`
Expected: FAIL — `Load` and `Save` are undefined.

- [ ] **Step 3: Write `internal/config/store.go`**

```go
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/poppyszn/super-ssh-v2/internal/fsutil"
)

// Load reads connections.toml.
//
// A missing file yields an empty store rather than an error, so a first run
// needs no bootstrap step. Anything else -- unreadable, unparseable, or
// semantically invalid -- is reported with the path in the message, because the
// user is expected to be able to open this file and fix it by hand.
func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Store{Version: Version}, nil
	}
	if err != nil {
		return Store{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var s Store
	if _, err := toml.Decode(string(data), &s); err != nil {
		return Store{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return Store{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Save validates, sorts, and writes the store atomically at 0600.
//
// Sorting keeps the file's diff stable, so a private git repo of
// connections.toml shows real changes rather than reordering noise. Validation
// happens before any bytes are written: an invalid store must never reach disk,
// because Load would then refuse to read it back.
func Save(path string, s Store) error {
	s.Version = Version
	sorted := make([]Profile, len(s.Profiles))
	copy(sorted, s.Profiles)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	s.Profiles = sorted

	if err := s.Validate(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(s); err != nil {
		return fmt.Errorf("encoding the connection store: %w", err)
	}
	if err := fsutil.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/config/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "feat: load and save the connection store atomically"
```

---

### Task 5: Generating ssh config

This is where the design pays off. Jump hosts and port forwarding — roughly 1,400 lines of engine in v1 — are `fmt.Fprintf` calls here.

**Files:**
- Create: `internal/sshconfig/generate.go`, `internal/sshconfig/generate_test.go`

**Interfaces:**
- Consumes: `config.Store`, `config.Profile`, the `Auth*` and `Tunnel*` constants (Task 3); `paths.Paths` and `fsutil.WriteFile` (Task 2).
- Produces:
  - `sshconfig.Generate(s config.Store) string` — pure, no I/O
  - `sshconfig.KeyRef(key string) string`
  - `sshconfig.Write(p paths.Paths, s config.Store) error`

**Directive order is fixed** and must not vary: `HostName`, `User`, `Port`, `IdentityFile`, `IdentitiesOnly`, `PreferredAuthentications`, `ProxyJump`, `ForwardAgent`, then tunnels in stored order. Indentation is exactly four spaces. There is **no timestamp** in the header — output must be byte-identical for identical input, because `sssh doctor` detects drift by regenerating and comparing.

- [ ] **Step 1: Write the failing test `internal/sshconfig/generate_test.go`**

```go
package sshconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func fullStore() config.Store {
	return config.Store{Version: config.Version, Profiles: []config.Profile{
		{
			Name: "prod-web", Host: "10.0.0.5", User: "deploy", Port: 2222,
			Auth: config.AuthKey, Key: "prod-web", Jump: "bastion", ForwardAgent: true,
			Tunnels: []config.Tunnel{{Type: config.TunnelLocal, Listen: "5432", Target: "db.internal:5432"}},
		},
		{Name: "bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
	}}
}

// The golden output. Every byte matters: doctor compares against a fresh
// Generate to detect hand edits, so any nondeterminism would report false drift.
const golden = `# Managed by sssh. Do not edit this file.
# Generated from connections.toml -- run ` + "`sssh sync`" + ` to regenerate.

Host bastion
    HostName b.example.com
    User admin
    Port 22

Host prod-web
    HostName 10.0.0.5
    User deploy
    Port 2222
    IdentityFile ~/.ssh/sssh.d/keys/prod-web
    IdentitiesOnly yes
    ProxyJump bastion
    ForwardAgent yes
    LocalForward 5432 db.internal:5432
`

func TestGenerateMatchesGolden(t *testing.T) {
	got := Generate(fullStore())
	if got != golden {
		t.Errorf("Generate() mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, golden)
	}
}

// Profiles are emitted sorted by name regardless of stored order, so the file
// never churns.
func TestGenerateIsDeterministic(t *testing.T) {
	s := fullStore()
	first := Generate(s)
	s.Profiles[0], s.Profiles[1] = s.Profiles[1], s.Profiles[0]
	if second := Generate(s); first != second {
		t.Error("Generate() output depends on profile order")
	}
}

func TestGenerateEmptyStoreIsJustTheHeader(t *testing.T) {
	got := Generate(config.Store{Version: config.Version})
	if !strings.HasPrefix(got, "# Managed by sssh.") {
		t.Errorf("missing header: %q", got)
	}
	if strings.Contains(got, "Host ") {
		t.Errorf("empty store produced a Host block: %q", got)
	}
}

func TestGenerateAuthVariants(t *testing.T) {
	tests := []struct {
		name       string
		profile    config.Profile
		wantLines  []string
		unwanted   []string
	}{
		{
			name:      "agent emits no auth directives",
			profile:   config.Profile{Name: "a", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
			wantLines: []string{"    HostName h"},
			unwanted:  []string{"IdentityFile", "IdentitiesOnly", "PreferredAuthentications"},
		},
		{
			name:      "key emits IdentityFile and IdentitiesOnly",
			profile:   config.Profile{Name: "a", Host: "h", User: "u", Port: 22, Auth: config.AuthKey, Key: "k"},
			wantLines: []string{"    IdentityFile ~/.ssh/sssh.d/keys/k", "    IdentitiesOnly yes"},
			unwanted:  []string{"PreferredAuthentications"},
		},
		{
			name:      "password prefers interactive methods",
			profile:   config.Profile{Name: "a", Host: "h", User: "u", Port: 22, Auth: config.AuthPassword},
			wantLines: []string{"    PreferredAuthentications password,keyboard-interactive"},
			unwanted:  []string{"IdentityFile"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Generate(config.Store{Version: config.Version, Profiles: []config.Profile{tt.profile}})
			for _, want := range tt.wantLines {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in:\n%s", want, got)
				}
			}
			for _, bad := range tt.unwanted {
				if strings.Contains(got, bad) {
					t.Errorf("unexpected %q in:\n%s", bad, got)
				}
			}
		})
	}
}

func TestGenerateTunnelVariants(t *testing.T) {
	tests := []struct {
		tunnel config.Tunnel
		want   string
	}{
		{config.Tunnel{Type: config.TunnelLocal, Listen: "5432", Target: "db:5432"}, "    LocalForward 5432 db:5432"},
		{config.Tunnel{Type: config.TunnelRemote, Listen: "8080", Target: "localhost:80"}, "    RemoteForward 8080 localhost:80"},
		{config.Tunnel{Type: config.TunnelDynamic, Listen: "1080"}, "    DynamicForward 1080"},
	}
	for _, tt := range tests {
		t.Run(string(tt.tunnel.Type), func(t *testing.T) {
			p := config.Profile{Name: "a", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent,
				Tunnels: []config.Tunnel{tt.tunnel}}
			got := Generate(config.Store{Version: config.Version, Profiles: []config.Profile{p}})
			if !strings.Contains(got, tt.want) {
				t.Errorf("missing %q in:\n%s", tt.want, got)
			}
		})
	}
}

func TestGenerateEmitsEveryTunnelInOrder(t *testing.T) {
	p := config.Profile{Name: "a", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent,
		Tunnels: []config.Tunnel{
			{Type: config.TunnelLocal, Listen: "1", Target: "x:1"},
			{Type: config.TunnelLocal, Listen: "2", Target: "x:2"},
		}}
	got := Generate(config.Store{Version: config.Version, Profiles: []config.Profile{p}})
	i, j := strings.Index(got, "LocalForward 1 x:1"), strings.Index(got, "LocalForward 2 x:2")
	if i < 0 || j < 0 {
		t.Fatalf("missing a tunnel line:\n%s", got)
	}
	if i > j {
		t.Error("tunnels emitted out of order")
	}
}

func TestKeyRef(t *testing.T) {
	tests := map[string]string{
		"prod-web":            "~/.ssh/sssh.d/keys/prod-web",
		"/etc/ssh/keys/ci":    "/etc/ssh/keys/ci",
		"~/.ssh/id_ed25519":   "~/.ssh/id_ed25519",
	}
	for in, want := range tests {
		if got := KeyRef(in); got != want {
			t.Errorf("KeyRef(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteCreatesTheConfigAt0600(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, fullStore()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(p.GenConfig)
	if err != nil {
		t.Fatalf("reading generated config: %v", err)
	}
	if string(data) != golden {
		t.Errorf("written content does not match Generate()")
	}
	info, err := os.Stat(p.GenConfig)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/sshconfig/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/sshconfig/generate.go`**

```go
// Package sshconfig turns the connection store into OpenSSH configuration.
//
// This is the whole reason v2 is small. Jump hosts, port forwarding, agent
// forwarding, and key selection are all features OpenSSH already implements
// correctly; here they are string formatting. v1 reimplemented them over the
// wire in about 1,400 lines.
package sshconfig

import (
	"fmt"
	"sort"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/fsutil"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// header carries no timestamp on purpose: identical input must produce
// byte-identical output so `sssh doctor` can detect hand edits by regenerating
// and comparing.
const header = "# Managed by sssh. Do not edit this file.\n" +
	"# Generated from connections.toml -- run `sssh sync` to regenerate.\n"

// KeyRef renders a profile's key as an IdentityFile value.
//
// An adopted key is stored as a bare filename and rendered with a tilde, so the
// generated config survives being restored onto a machine whose home directory
// has a different path. A linked key is already absolute or tilde-prefixed and
// passes through unchanged.
func KeyRef(key string) string {
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "~") {
		return key
	}
	return "~/.ssh/sssh.d/keys/" + key
}

// Generate renders the entire store as ssh config text. It performs no I/O.
func Generate(s config.Store) string {
	profiles := make([]config.Profile, len(s.Profiles))
	copy(profiles, s.Profiles)
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })

	var b strings.Builder
	b.WriteString(header)
	for _, p := range profiles {
		fmt.Fprintf(&b, "\nHost %s\n", p.Name)
		fmt.Fprintf(&b, "    HostName %s\n", p.Host)
		fmt.Fprintf(&b, "    User %s\n", p.User)
		fmt.Fprintf(&b, "    Port %d\n", p.Port)

		switch p.Auth {
		case config.AuthKey:
			fmt.Fprintf(&b, "    IdentityFile %s\n", KeyRef(p.Key))
			// IdentitiesOnly stops ssh offering every key in the agent first,
			// which on a host with many loaded keys can exhaust the server's
			// authentication attempt limit before the right key is tried.
			b.WriteString("    IdentitiesOnly yes\n")
		case config.AuthPassword:
			b.WriteString("    PreferredAuthentications password,keyboard-interactive\n")
		case config.AuthAgent:
			// Nothing: ssh finds the agent through SSH_AUTH_SOCK by default.
		}

		if p.Jump != "" {
			b.WriteString("    ProxyJump " + p.Jump + "\n")
		}
		if p.ForwardAgent {
			b.WriteString("    ForwardAgent yes\n")
		}
		for _, t := range p.Tunnels {
			switch t.Type {
			case config.TunnelLocal:
				fmt.Fprintf(&b, "    LocalForward %s %s\n", t.Listen, t.Target)
			case config.TunnelRemote:
				fmt.Fprintf(&b, "    RemoteForward %s %s\n", t.Listen, t.Target)
			case config.TunnelDynamic:
				fmt.Fprintf(&b, "    DynamicForward %s\n", t.Listen)
			}
		}
	}
	return b.String()
}

// Write regenerates the sandbox config file atomically.
func Write(p paths.Paths, s config.Store) error {
	if err := fsutil.WriteFile(p.GenConfig, []byte(Generate(s)), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", p.GenConfig, err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/sshconfig/ -v`
Expected: all PASS. If the golden test fails, diff the reported blocks character by character — spacing and blank lines are part of the contract.

- [ ] **Step 5: Commit**

```bash
git add internal/sshconfig
git commit -m "feat: generate ssh config from the connection store"
```

---

### Task 6: The managed Include block

**Files:**
- Create: `internal/sshconfig/include.go`, `internal/sshconfig/include_test.go`

**Interfaces:**
- Consumes: `paths.Paths`, `fsutil.WriteFile` (Task 2).
- Produces:
  - Constants `BeginMarker`, `EndMarker`, `IncludeLine`; variable `Block`
  - `HasBlock(content string) bool`
  - `AddBlock(content string) string`
  - `RemoveBlock(content string) string`
  - `BlockIsFirst(content string) bool`
  - `EnsureInclude(p paths.Paths) error`

**Placement is load-bearing.** ssh config is first-match-wins: the first value seen for a keyword wins, and a `Host *` block earlier in the file would resolve settings before the `Include` was ever reached. The block therefore goes at the very top.

- [ ] **Step 1: Write the failing test `internal/sshconfig/include_test.go`**

```go
package sshconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func TestAddBlockToEmptyContent(t *testing.T) {
	got := AddBlock("")
	want := "# >>> sssh managed block >>>\nInclude ~/.ssh/sssh.d/config\n# <<< sssh managed block <<<\n"
	if got != want {
		t.Errorf("AddBlock(\"\") = %q, want %q", got, want)
	}
}

// The block must precede the user's own content, because ssh config is
// first-match-wins and a Host * block above the Include would win.
func TestAddBlockGoesAtTheTop(t *testing.T) {
	existing := "Host myserver\n    HostName example.com\n"
	got := AddBlock(existing)
	if !strings.HasPrefix(got, BeginMarker) {
		t.Errorf("block is not at the top:\n%s", got)
	}
	if !strings.Contains(got, existing) {
		t.Errorf("user content was not preserved:\n%s", got)
	}
	if strings.Index(got, IncludeLine) > strings.Index(got, "Host myserver") {
		t.Error("Include appears after the user's Host block")
	}
}

func TestAddBlockIsIdempotent(t *testing.T) {
	once := AddBlock("Host a\n    HostName a.example.com\n")
	if twice := AddBlock(once); twice != once {
		t.Errorf("AddBlock is not idempotent:\n%s", twice)
	}
}

func TestHasBlock(t *testing.T) {
	if HasBlock("Host a\n") {
		t.Error("HasBlock reported true for content with no block")
	}
	if !HasBlock(AddBlock("Host a\n")) {
		t.Error("HasBlock reported false for content with a block")
	}
}

func TestRemoveBlockRestoresOriginalContent(t *testing.T) {
	original := "Host myserver\n    HostName example.com\n"
	if got := RemoveBlock(AddBlock(original)); got != original {
		t.Errorf("RemoveBlock(AddBlock(x)) = %q, want %q", got, original)
	}
}

func TestRemoveBlockLeavesUnmanagedContentAlone(t *testing.T) {
	original := "Host myserver\n    HostName example.com\n"
	if got := RemoveBlock(original); got != original {
		t.Errorf("RemoveBlock changed unmanaged content: %q", got)
	}
}

func TestBlockIsFirst(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"block at the top", AddBlock("Host a\n    HostName a\n"), true},
		{"block after a Host", "Host a\n    HostName a\n" + Block, false},
		{"block after a Match", "Match user bob\n    User bob\n" + Block, false},
		{"host keyword lowercase", "host a\n" + Block, false},
		{"no block at all", "Host a\n", false},
		{"block after a comment is still first", "# my notes\n" + Block + "\nHost a\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BlockIsFirst(tt.content); got != tt.want {
				t.Errorf("BlockIsFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureIncludeCreatesAMissingConfigAt0600(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("EnsureInclude: %v", err)
	}
	data, err := os.ReadFile(p.SSHConfig)
	if err != nil {
		t.Fatalf("reading ssh config: %v", err)
	}
	if !HasBlock(string(data)) {
		t.Errorf("block missing from a freshly created config:\n%s", data)
	}
	info, _ := os.Stat(p.SSHConfig)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestEnsureIncludePreservesExistingContentAndMode(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	existing := "Host myserver\n    HostName example.com\n    User me\n"
	if err := os.WriteFile(p.SSHConfig, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("EnsureInclude: %v", err)
	}
	data, _ := os.ReadFile(p.SSHConfig)
	if !strings.Contains(string(data), existing) {
		t.Errorf("user content was lost:\n%s", data)
	}
	if !BlockIsFirst(string(data)) {
		t.Errorf("block is not first:\n%s", data)
	}
	info, _ := os.Stat(p.SSHConfig)
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mode = %o, want the original 644 to be preserved", perm)
	}
}

func TestEnsureIncludeIsIdempotent(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(p.SSHConfig)
	if err := EnsureInclude(p); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(p.SSHConfig)
	if string(first) != string(second) {
		t.Errorf("second EnsureInclude changed the file:\n%s", second)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/sshconfig/ -run 'Block|Include' -v`
Expected: FAIL — `AddBlock` and friends are undefined.

- [ ] **Step 3: Write `internal/sshconfig/include.go`**

```go
package sshconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/fsutil"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// The markers delimit the only part of ~/.ssh/config that sssh owns. Every
// byte outside them belongs to the user and is never modified.
const (
	BeginMarker = "# >>> sssh managed block >>>"
	EndMarker   = "# <<< sssh managed block <<<"
	IncludeLine = "Include ~/.ssh/sssh.d/config"
)

// Block is what gets inserted into the user's ssh config. The Include path uses
// a tilde rather than an absolute path so the same line works on any machine
// the config is restored onto.
var Block = BeginMarker + "\n" + IncludeLine + "\n" + EndMarker + "\n"

// HasBlock reports whether the managed block is present.
func HasBlock(content string) bool {
	return strings.Contains(content, BeginMarker) && strings.Contains(content, EndMarker)
}

// AddBlock returns content with the managed block prepended.
//
// Top placement is mandatory. ssh config is first-match-wins: the first value
// seen for a keyword is the one used. An Include placed below a "Host *" block
// would be read too late to affect anything that block already set, and the
// failure would be silent.
func AddBlock(content string) string {
	if HasBlock(content) {
		return content
	}
	if content == "" {
		return Block
	}
	return Block + "\n" + content
}

// RemoveBlock strips the managed block and leaves everything else byte-identical.
func RemoveBlock(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inside := false
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case BeginMarker:
			inside = true
			continue
		case EndMarker:
			inside = false
			continue
		}
		if inside {
			continue
		}
		out = append(out, line)
	}
	// Drop the single blank line AddBlock inserted between the block and the
	// user's content, so remove(add(x)) == x.
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	return strings.Join(out, "\n")
}

// BlockIsFirst reports whether the managed block appears before any Host or
// Match directive. `sssh doctor` uses this to catch a config where the block was
// moved down, which disables it without any visible error.
func BlockIsFirst(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == BeginMarker {
			return true
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "host ") || strings.HasPrefix(lower, "match ") {
			return false
		}
	}
	return false
}

// EnsureInclude adds the managed block to the user's ssh config if it is not
// already there, creating the file if the machine has none. An existing file
// keeps its permissions: sssh has no business tightening or loosening a mode
// the user chose.
func EnsureInclude(p paths.Paths) error {
	data, err := os.ReadFile(p.SSHConfig)
	if errors.Is(err, fs.ErrNotExist) {
		if err := fsutil.WriteFile(p.SSHConfig, []byte(Block), 0o600); err != nil {
			return fmt.Errorf("creating %s: %w", p.SSHConfig, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", p.SSHConfig, err)
	}
	if HasBlock(string(data)) {
		return nil
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(p.SSHConfig); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := fsutil.WriteFile(p.SSHConfig, []byte(AddBlock(string(data))), mode); err != nil {
		return fmt.Errorf("updating %s: %w", p.SSHConfig, err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/sshconfig/ -v`
Expected: all PASS, including the golden test from Task 5.

- [ ] **Step 5: Commit**

```bash
git add internal/sshconfig
git commit -m "feat: manage the Include block in the user's ssh config"
```

---

### Task 7: Adopting keys

**Files:**
- Create: `internal/keys/keys.go`, `internal/keys/keys_test.go`

**Interfaces:**
- Consumes: nothing beyond stdlib.
- Produces:
  - `keys.ValidatePrivateKey(content []byte) error`
  - `keys.AdoptContent(keysDir, name string, content []byte) (string, error)` — returns the basename to store in `Profile.Key`
  - `keys.AdoptFile(keysDir, name, src, home string) (string, error)`
  - `keys.ExpandTilde(path, home string) string`

- [ ] **Step 1: Write the failing test `internal/keys/keys_test.go`**

```go
package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A syntactically plausible private key. Content is never parsed as crypto --
// only checked for the PEM header -- so this does not need to be a real key.
const fakeKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
-----END OPENSSH PRIVATE KEY-----
`

func TestValidatePrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"openssh private key", fakeKey, false},
		{"rsa private key", "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----\n", false},
		{"empty", "", true},
		{"whitespace only", "   \n\t\n", true},
		{"ed25519 public key", "ssh-ed25519 AAAAC3Nza me@host\n", true},
		{"rsa public key", "ssh-rsa AAAAB3Nza me@host\n", true},
		{"ecdsa public key", "ecdsa-sha2-nistp256 AAAAE2V me@host\n", true},
		{"random text", "hello world\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrivateKey([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePrivateKey() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePrivateKeyNamesThePublicKeyMistake(t *testing.T) {
	err := ValidatePrivateKey([]byte("ssh-ed25519 AAAAC3Nza me@host\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "public key") {
		t.Errorf("error %q should tell the user they supplied a public key", err)
	}
}

func TestAdoptContentWritesAt0600(t *testing.T) {
	dir := t.TempDir()
	name, err := AdoptContent(dir, "prod-web", []byte(fakeKey))
	if err != nil {
		t.Fatalf("AdoptContent: %v", err)
	}
	if name != "prod-web" {
		t.Errorf("returned name = %q, want %q", name, "prod-web")
	}
	dest := filepath.Join(dir, "prod-web")
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != fakeKey {
		t.Errorf("content mismatch")
	}
}

// OpenSSH rejects a key file whose final line has no newline.
func TestAdoptContentAppendsATrailingNewline(t *testing.T) {
	dir := t.TempDir()
	if _, err := AdoptContent(dir, "k", []byte(strings.TrimSuffix(fakeKey, "\n"))); err != nil {
		t.Fatalf("AdoptContent: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "k"))
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("stored key has no trailing newline")
	}
}

// Re-adopting identical bytes must succeed, so `sssh import` can run twice.
func TestAdoptContentIsIdempotentForIdenticalKeys(t *testing.T) {
	dir := t.TempDir()
	if _, err := AdoptContent(dir, "k", []byte(fakeKey)); err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptContent(dir, "k", []byte(fakeKey)); err != nil {
		t.Fatalf("re-adopting an identical key failed: %v", err)
	}
}

// Adopting different bytes under a name in use must fail rather than silently
// destroy a key the user may still need.
func TestAdoptContentRefusesToOverwriteADifferentKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := AdoptContent(dir, "k", []byte(fakeKey)); err != nil {
		t.Fatal(err)
	}
	other := strings.Replace(fakeKey, "b3Blbn", "ZZZZZZ", 1)
	if _, err := AdoptContent(dir, "k", []byte(other)); err == nil {
		t.Fatal("expected an error when adopting a different key under an existing name")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "k"))
	if string(data) != fakeKey {
		t.Error("the original key was modified despite the error")
	}
}

func TestAdoptContentRejectsAPublicKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := AdoptContent(dir, "k", []byte("ssh-rsa AAAA me@host\n")); err == nil {
		t.Fatal("expected an error adopting a public key")
	}
	if _, err := os.Stat(filepath.Join(dir, "k")); err == nil {
		t.Error("a file was written despite the error")
	}
}

func TestAdoptContentCreatesTheKeysDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	if _, err := AdoptContent(dir, "k", []byte(fakeKey)); err != nil {
		t.Fatalf("AdoptContent: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("keys directory not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("keys directory mode = %o, want 700", perm)
	}
}

func TestAdoptFileReadsFromDisk(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(home, "id_test")
	if err := os.WriteFile(src, []byte(fakeKey), 0o600); err != nil {
		t.Fatal(err)
	}
	keysDir := filepath.Join(t.TempDir(), "keys")
	if _, err := AdoptFile(keysDir, "mykey", src, home); err != nil {
		t.Fatalf("AdoptFile: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(keysDir, "mykey"))
	if err != nil {
		t.Fatalf("adopted key not found: %v", err)
	}
	if string(data) != fakeKey {
		t.Error("content mismatch")
	}
}

func TestAdoptFileExpandsTilde(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "id_test"), []byte(fakeKey), 0o600); err != nil {
		t.Fatal(err)
	}
	keysDir := filepath.Join(t.TempDir(), "keys")
	if _, err := AdoptFile(keysDir, "mykey", "~/id_test", home); err != nil {
		t.Fatalf("AdoptFile with a tilde path: %v", err)
	}
}

func TestAdoptFileReportsAMissingSource(t *testing.T) {
	if _, err := AdoptFile(t.TempDir(), "k", "/nonexistent/key", "/home/u"); err == nil {
		t.Fatal("expected an error for a missing source file")
	}
}

func TestExpandTilde(t *testing.T) {
	tests := map[string]string{
		"~":            "/home/u",
		"~/.ssh/id":    "/home/u/.ssh/id",
		"/etc/key":     "/etc/key",
		"relative/key": "relative/key",
		"~notme/key":   "~notme/key",
	}
	for in, want := range tests {
		if got := ExpandTilde(in, "/home/u"); got != want {
			t.Errorf("ExpandTilde(%q) = %q, want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/keys/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/keys/keys.go`**

```go
// Package keys adopts private key material into the sssh sandbox, so that one
// encrypted backup contains everything needed to connect from a new machine.
package keys

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// publicKeyPrefixes are how an OpenSSH public key begins. Supplying the .pub
// file by mistake is the single most common error here, and it is worth
// catching with a message that says so.
var publicKeyPrefixes = []string{"ssh-rsa ", "ssh-ed25519 ", "ssh-dss ", "ecdsa-sha2-", "sk-ssh-", "sk-ecdsa-"}

// ValidatePrivateKey performs a cheap structural check.
//
// It deliberately does not parse the key: doing so would mean importing an SSH
// library, which the guardrails forbid, and would fail on encrypted keys whose
// passphrase sssh does not have. Catching the obvious mistakes at add time,
// when the user still has context, is worth more than cryptographic certainty.
func ValidatePrivateKey(content []byte) error {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return errors.New("the key is empty")
	}
	for _, prefix := range publicKeyPrefixes {
		if bytes.HasPrefix(trimmed, []byte(prefix)) {
			return errors.New("this is a public key; sssh needs the matching private key (the same filename without the .pub extension)")
		}
	}
	if !bytes.Contains(trimmed, []byte("PRIVATE KEY")) {
		return errors.New(`this does not look like a private key: no "PRIVATE KEY" header found`)
	}
	return nil
}

// AdoptContent stores key material in keysDir at 0600 and returns the basename
// to record in the profile.
//
// Adopting identical bytes again succeeds, so `sssh import` is safe to re-run.
// Adopting different bytes under a name already in use fails, because silently
// replacing a key the user may still need is not recoverable.
func AdoptContent(keysDir, name string, content []byte) (string, error) {
	if err := ValidatePrivateKey(content); err != nil {
		return "", fmt.Errorf("key %q: %w", name, err)
	}
	if !bytes.HasSuffix(content, []byte("\n")) {
		content = append(content, '\n') // OpenSSH rejects a key with no final newline
	}

	dest := filepath.Join(keysDir, name)
	switch existing, err := os.ReadFile(dest); {
	case err == nil && bytes.Equal(existing, content):
		return name, os.Chmod(dest, 0o600)
	case err == nil:
		return "", fmt.Errorf("a different key is already stored as %q; remove %s or use another profile name", name, dest)
	case !errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("reading %s: %w", dest, err)
	}

	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return "", fmt.Errorf("creating %s: %w", keysDir, err)
	}
	if err := os.WriteFile(dest, content, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return name, nil
}

// AdoptFile reads a key from src and adopts it under name. home resolves a
// leading tilde; it is passed in rather than looked up so tests stay hermetic.
func AdoptFile(keysDir, name, src, home string) (string, error) {
	content, err := os.ReadFile(ExpandTilde(src, home))
	if err != nil {
		return "", fmt.Errorf("reading key %s: %w", src, err)
	}
	return AdoptContent(keysDir, name, content)
}

// ExpandTilde resolves a leading "~" or "~/" against home. "~user" forms are
// left alone: sssh does not resolve other users' home directories.
func ExpandTilde(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/keys/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/keys
git commit -m "feat: adopt private keys into the sssh sandbox"
```

---

### Task 8: Launching ssh

The smallest package in the repo, and the most important. Because the generated config already carries every setting, the argv is just `ssh <name>`.

**Files:**
- Create: `internal/launch/launch.go`, `internal/launch/launch_test.go`
- Modify: `scripts/guardrails.sh` — remove the temporary `|| [ "$nexec" = "0" ]` clause added in Task 1 Step 10.

**Interfaces:**
- Consumes: nothing beyond stdlib.
- Produces:
  - `launch.Args(name string, extra []string) []string`
  - `launch.SCPArgs(operands []string) []string`
  - `launch.Resolve(argv []string) (string, error)`
  - `launch.Exec(argv []string) error` — does not return on success

**Tests must never call `Exec`.** It replaces the test binary's process image and the run would report nothing. Test `Args`, `SCPArgs`, and `Resolve`; `Exec` is three lines over a verified `Resolve`.

- [ ] **Step 1: Write the failing test `internal/launch/launch_test.go`**

```go
package launch

import (
	"reflect"
	"testing"
)

// The argv is deliberately minimal. Every setting -- key, port, user, jump
// host, tunnels, auth preference -- is already in the generated ssh config
// under this Host alias, so there is nothing to assemble here.
func TestArgs(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		extra   []string
		want    []string
	}{
		{"bare connect", "prod-web", nil, []string{"ssh", "prod-web"}},
		{"remote command", "prod-web", []string{"uptime"}, []string{"ssh", "prod-web", "uptime"}},
		{"command with args", "prod-web", []string{"tail", "-f", "/var/log/x"}, []string{"ssh", "prod-web", "tail", "-f", "/var/log/x"}},
		{"empty extra slice", "web", []string{}, []string{"ssh", "web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Args(tt.profile, tt.extra); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Args() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Args must not alias or mutate the caller's slice.
func TestArgsDoesNotMutateExtra(t *testing.T) {
	extra := []string{"uptime"}
	Args("web", extra)
	if len(extra) != 1 || extra[0] != "uptime" {
		t.Errorf("extra was mutated: %v", extra)
	}
}

// Profile names are ssh Host aliases, so they work as scp targets unchanged.
func TestSCPArgs(t *testing.T) {
	got := SCPArgs([]string{"prod-web:/var/log/app.log", "./"})
	want := []string{"scp", "prod-web:/var/log/app.log", "./"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SCPArgs() = %v, want %v", got, want)
	}
}

func TestResolveFindsAnExistingBinary(t *testing.T) {
	// sh is present on every platform this tool supports.
	path, err := Resolve([]string{"sh"})
	if err != nil {
		t.Fatalf("Resolve(sh) = %v, want nil", err)
	}
	if path == "" {
		t.Error("Resolve returned an empty path")
	}
}

func TestResolveReportsAMissingBinary(t *testing.T) {
	_, err := Resolve([]string{"definitely-not-a-real-binary-xyzzy"})
	if err == nil {
		t.Fatal("expected an error for a binary that is not in PATH")
	}
}

func TestResolveRejectsEmptyArgv(t *testing.T) {
	if _, err := Resolve(nil); err == nil {
		t.Fatal("expected an error for an empty argv")
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/launch/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/launch/launch.go`**

```go
// Package launch builds the command line for ssh and scp and then replaces this
// process with it.
//
// This package is the reason sssh is fast. v1 spoke the SSH protocol in Go and
// piped every byte of remote output through a terminal emulator; that is why
// copying text was slow, why a long-running command stalled the session, and
// why the machine ran hot. Here, sssh exec's the real ssh binary and ceases to
// exist. Nothing it does can affect the session, because it is no longer
// running.
//
// The guardrail script enforces that syscall.Exec appears only in this file.
package launch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Args builds the argv for connecting to a saved profile.
//
// It is this short because the generated ssh config already carries the key,
// port, user, jump host, tunnels, and authentication preference under this Host
// alias. There are no flags to assemble. Anything in extra is passed through, so
// `sssh web uptime` runs a remote command.
func Args(name string, extra []string) []string {
	argv := make([]string, 0, len(extra)+2)
	argv = append(argv, "ssh", name)
	return append(argv, extra...)
}

// SCPArgs builds the argv for a copy. Profile names need no rewriting: they are
// ssh Host aliases, so scp resolves them through the same generated config.
func SCPArgs(operands []string) []string {
	argv := make([]string, 0, len(operands)+1)
	argv = append(argv, "scp")
	return append(argv, operands...)
}

// Resolve locates argv[0] in PATH.
func Resolve(argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("no command given")
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return "", fmt.Errorf("%s was not found in PATH: %w", argv[0], err)
	}
	return path, nil
}

// Exec replaces this process with the given command.
//
// On success it does not return: the process image is gone and ssh owns the
// terminal directly. On failure it reports the full command line, so a problem
// can be reproduced by hand.
func Exec(argv []string) error {
	bin, err := Resolve(argv)
	if err != nil {
		return err
	}
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		return fmt.Errorf("running %s: %w", strings.Join(argv, " "), err)
	}
	return nil // unreachable: syscall.Exec either replaces the process or errors
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/launch/ -v`
Expected: all PASS.

- [ ] **Step 5: Restore the strict guardrail**

In `scripts/guardrails.sh`, check 5 must now be exactly:

```bash
if [ "$nexec" = "1" ] && [ "$where" = "./internal/launch/launch.go" ]; then
```

Remove the `|| [ "$nexec" = "0" ]` allowance added in Task 1.

- [ ] **Step 6: Run the full gate**

Run: `make check`
Expected: all tests pass and guardrails prints five `ok` lines, including `syscall.Exec confined to internal/launch`.

- [ ] **Step 7: Commit**

```bash
git add internal/launch scripts/guardrails.sh
git commit -m "feat: build ssh argv and exec the real ssh binary"
```

---

### Task 9: CLI plumbing and `sssh ls`

**Files:**
- Create: `cmd/sssh/app.go`, `cmd/sssh/cmd_ls.go`, `cmd/sssh/cmd_ls_test.go`, `cmd/sssh/app_test.go`
- Modify: `cmd/sssh/main.go` (replace `run` with the dispatching version below), `cmd/sssh/main_test.go` (replace `TestRunUnknownCommand`, which no longer holds — an unrecognised first argument is now treated as a profile name)

**Interfaces:**
- Consumes: `paths.Default`, `paths.EnsureDirs`, `config.Load`, `config.Save`, `sshconfig.Write`, `sshconfig.EnsureInclude`.
- Produces:
  - `type app struct { paths paths.Paths; store config.Store; stdout, stderr io.Writer; stdin io.Reader }`
  - `newApp(p paths.Paths) (*app, error)`
  - `(*app) save() error` — persists the store **and** regenerates the config
  - `(*app) sync() error`
  - `withApp(fn func(*app) error) error`
  - `cmdLs(a *app, args []string) error`
  - `renderList(profiles []config.Profile) string`, `renderTree(profiles []config.Profile) string`

- [ ] **Step 1: Write `cmd/sssh/app.go`**

```go
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
)

// app is the shared context every command needs. Streams are fields rather
// than package-level os.Stdout references so command output can be captured in
// tests without touching global state.
type app struct {
	paths  paths.Paths
	store  config.Store
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func newApp(p paths.Paths) (*app, error) {
	if err := paths.EnsureDirs(p); err != nil {
		return nil, err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	return &app{paths: p, store: store, stdout: os.Stdout, stderr: os.Stderr, stdin: os.Stdin}, nil
}

// save persists the store and regenerates everything downstream of it.
//
// Every mutating command ends here. Regeneration is not a separate step the
// user can forget, which is why there is no drift to reconcile and why `sssh
// sync` exists only as a repair tool.
func (a *app) save() error {
	if err := config.Save(a.paths.StoreFile, a.store); err != nil {
		return err
	}
	return a.sync()
}

// sync rewrites the generated config and makes sure the user's ssh config
// includes it.
func (a *app) sync() error {
	if err := sshconfig.Write(a.paths, a.store); err != nil {
		return err
	}
	return sshconfig.EnsureInclude(a.paths)
}

// withApp resolves paths, loads the store, and runs fn.
func withApp(fn func(*app) error) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	a, err := newApp(p)
	if err != nil {
		return err
	}
	return fn(a)
}

// errNoProfile is the message used everywhere a name fails to resolve, so the
// user is always pointed at the same next step.
func errNoProfile(name string) error {
	return fmt.Errorf("no profile named %q (run `sssh ls` to see what is saved)", name)
}
```

- [ ] **Step 2: Replace `run` in `cmd/sssh/main.go`**

Keep `main`, `version`, and `usage` exactly as they are. Replace the whole `run` function with:

```go
func run(args []string) error {
	if len(args) == 0 {
		return withApp(cmdPickAndConnect)
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("sssh " + version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	case "ls":
		return withApp(func(a *app) error { return cmdLs(a, args[1:]) })
	case "sync":
		return withApp(cmdSync)
	}
	if strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("unknown flag %q (try `sssh help`)", args[0])
	}
	return withApp(func(a *app) error { return cmdConnect(a, args[0], args[1:]) })
}
```

Add `"strings"` to the import block. `cmdPickAndConnect`, `cmdSync`, and `cmdConnect` arrive in Tasks 11–13; until then, add these three temporary stubs at the bottom of `main.go` so the package compiles, and delete each one as its real implementation lands:

```go
// Temporary stubs. Each is replaced by its real implementation:
// cmdSync in Task 11, cmdConnect in Task 12, cmdPickAndConnect in Task 13.
func cmdSync(a *app) error            { return a.sync() }
func cmdConnect(a *app, name string, extra []string) error {
	if _, ok := a.store.Find(name); !ok {
		return errNoProfile(name)
	}
	return fmt.Errorf("connect is not implemented yet")
}
func cmdPickAndConnect(a *app) error { return fmt.Errorf("the picker is not implemented yet") }
```

- [ ] **Step 3: Replace `TestRunUnknownCommand` in `cmd/sssh/main_test.go`**

Delete `TestRunUnknownCommand` — an unrecognised first argument is now a profile name, not an error about commands. Replace it with:

```go
func TestRunUnknownNameIsReportedAsAMissingProfile(t *testing.T) {
	t.Setenv("SSSH_HOME", t.TempDir())
	err := run([]string{"wibble"})
	if err == nil {
		t.Fatal("expected an error for a name that is not a saved profile")
	}
	if !strings.Contains(err.Error(), "no profile named") {
		t.Errorf("error = %q, want it to mention a missing profile", err)
	}
}

func TestRunRejectsUnknownFlags(t *testing.T) {
	t.Setenv("SSSH_HOME", t.TempDir())
	if err := run([]string{"--wibble"}); err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
}
```

Add `"strings"` to that file's imports.

- [ ] **Step 4: Write the failing test `cmd/sssh/cmd_ls_test.go`**

```go
package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func testProfiles() []config.Profile {
	return []config.Profile{
		{Name: "bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
		{Name: "prod-web", Host: "10.0.0.5", User: "deploy", Port: 2222, Auth: config.AuthKey,
			Key: "prod-web", Folder: "work/prod", Tags: []string{"web", "prod"}},
		{Name: "staging", Host: "s.example.com", User: "deploy", Port: 22, Auth: config.AuthAgent,
			Folder: "work/staging"},
	}
}

// newTestApp builds an app rooted at a temp directory with the given profiles
// already saved, and with output captured.
func newTestApp(t *testing.T, profiles []config.Profile) (*app, *bytes.Buffer) {
	t.Helper()
	p := paths.New(t.TempDir())
	a, err := newApp(p)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}
	a.store.Profiles = profiles
	if len(profiles) > 0 {
		if err := a.save(); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	out := &bytes.Buffer{}
	a.stdout = out
	return a, out
}

func TestRenderListShowsEveryProfile(t *testing.T) {
	got := renderList(testProfiles())
	for _, want := range []string{"bastion", "prod-web", "staging", "10.0.0.5", "2222", "deploy", "work/prod", "web,prod"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderList output missing %q:\n%s", want, got)
		}
	}
}

func TestRenderListHasAHeaderRow(t *testing.T) {
	got := renderList(testProfiles())
	first := strings.SplitN(got, "\n", 2)[0]
	for _, col := range []string{"NAME", "HOST", "USER", "FOLDER", "TAGS"} {
		if !strings.Contains(first, col) {
			t.Errorf("header row %q missing column %q", first, col)
		}
	}
}

func TestRenderTreeGroupsByFolder(t *testing.T) {
	got := renderTree(testProfiles())
	prod := strings.Index(got, "work/prod")
	staging := strings.Index(got, "work/staging")
	if prod < 0 || staging < 0 {
		t.Fatalf("folders missing from tree output:\n%s", got)
	}
	if prod > staging {
		t.Error("folders are not in sorted order")
	}
	if !strings.Contains(got, "(no folder)") {
		t.Errorf("profiles with no folder are not grouped:\n%s", got)
	}
}

func TestCmdLsOnAnEmptyStoreExplainsWhatToDo(t *testing.T) {
	a, out := newTestApp(t, nil)
	if err := cmdLs(a, nil); err != nil {
		t.Fatalf("cmdLs: %v", err)
	}
	if !strings.Contains(out.String(), "sssh add") {
		t.Errorf("empty-store output should point at `sssh add`, got:\n%s", out.String())
	}
}

func TestCmdLsWritesToTheAppStdout(t *testing.T) {
	a, out := newTestApp(t, testProfiles())
	if err := cmdLs(a, nil); err != nil {
		t.Fatalf("cmdLs: %v", err)
	}
	if !strings.Contains(out.String(), "prod-web") {
		t.Errorf("output missing a profile:\n%s", out.String())
	}
}

func TestCmdLsTreeFlag(t *testing.T) {
	a, out := newTestApp(t, testProfiles())
	if err := cmdLs(a, []string{"--tree"}); err != nil {
		t.Fatalf("cmdLs --tree: %v", err)
	}
	if !strings.Contains(out.String(), "work/prod") {
		t.Errorf("--tree output missing folders:\n%s", out.String())
	}
}

func TestAppSaveRegeneratesTheConfigAndIncludeBlock(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	data, err := readFileString(a.paths.GenConfig)
	if err != nil {
		t.Fatalf("generated config not written: %v", err)
	}
	if !strings.Contains(data, "Host prod-web") {
		t.Errorf("generated config missing a Host block:\n%s", data)
	}
	sshCfg, err := readFileString(a.paths.SSHConfig)
	if err != nil {
		t.Fatalf("ssh config not written: %v", err)
	}
	if !strings.Contains(sshCfg, "Include ~/.ssh/sssh.d/config") {
		t.Errorf("Include line missing:\n%s", sshCfg)
	}
}
```

Add this helper to `cmd/sssh/app_test.go`:

```go
package main

import "os"

func readFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}
```

- [ ] **Step 5: Run it and confirm it fails**

Run: `go test ./cmd/sssh/ -v`
Expected: FAIL — `cmdLs`, `renderList`, and `renderTree` are undefined.

- [ ] **Step 6: Write `cmd/sssh/cmd_ls.go`**

```go
package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func cmdLs(a *app, args []string) error {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	tree := fs.Bool("tree", false, "group profiles by folder")
	folder := fs.String("folder", "", "show only profiles in this folder")
	tag := fs.String("tag", "", "show only profiles with this tag")
	if err := fs.Parse(args); err != nil {
		return err
	}

	profiles := filterProfiles(a.store.Profiles, *folder, *tag)
	if len(profiles) == 0 {
		if len(a.store.Profiles) == 0 {
			fmt.Fprintln(a.stdout, "No connections saved yet. Add one with `sssh add`,")
			fmt.Fprintln(a.stdout, "or adopt your existing hosts with `sssh import`.")
			return nil
		}
		fmt.Fprintln(a.stdout, "No profiles match that filter.")
		return nil
	}

	if *tree {
		fmt.Fprint(a.stdout, renderTree(profiles))
		return nil
	}
	fmt.Fprint(a.stdout, renderList(profiles))
	return nil
}

func filterProfiles(profiles []config.Profile, folder, tag string) []config.Profile {
	out := make([]config.Profile, 0, len(profiles))
	for _, p := range profiles {
		if folder != "" && p.Folder != folder {
			continue
		}
		if tag != "" && !hasTag(p, tag) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func hasTag(p config.Profile, tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// renderList produces an aligned table. tabwriter is stdlib, so this costs no
// dependency budget.
func renderList(profiles []config.Profile) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOST\tUSER\tAUTH\tFOLDER\tTAGS")
	for _, p := range sortedByName(profiles) {
		fmt.Fprintf(w, "%s\t%s:%d\t%s\t%s\t%s\t%s\n",
			p.Name, p.Host, p.Port, p.User, p.Auth, p.Folder, strings.Join(p.Tags, ","))
	}
	w.Flush()
	return b.String()
}

// renderTree groups by folder. Profiles with no folder are gathered under
// "(no folder)" and shown last, so the named groups read first.
func renderTree(profiles []config.Profile) string {
	groups := map[string][]config.Profile{}
	for _, p := range profiles {
		key := p.Folder
		if key == "" {
			key = "(no folder)"
		}
		groups[key] = append(groups[key], p)
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if (names[i] == "(no folder)") != (names[j] == "(no folder)") {
			return names[j] == "(no folder)"
		}
		return names[i] < names[j]
	})

	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "%s\n", name)
		for _, p := range sortedByName(groups[name]) {
			fmt.Fprintf(&b, "  %-20s %s@%s:%d\n", p.Name, p.User, p.Host, p.Port)
		}
	}
	return b.String()
}

func sortedByName(profiles []config.Profile) []config.Profile {
	out := make([]config.Profile, len(profiles))
	copy(out, profiles)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./cmd/sssh/ -v`
Expected: all PASS.

- [ ] **Step 8: Run the full gate and commit**

Run: `make check`

```bash
git add cmd/sssh
git commit -m "feat: add CLI plumbing and the ls command"
```

---

### Task 10: `sssh add`

**Files:**
- Create: `cmd/sssh/cmd_add.go`, `cmd/sssh/cmd_add_test.go`
- Modify: `cmd/sssh/main.go` — add `case "add": return withApp(func(a *app) error { return cmdAdd(a, args[1:]) })` to the switch

**Interfaces:**
- Consumes: `keys.AdoptFile`, `keys.AdoptContent`, `config.Profile`, `(*app).save`.
- Produces: `cmdAdd(a *app, args []string) error`, `promptFor(r *bufio.Reader, w io.Writer, label, def string) (string, error)`

**Two modes, and the rule between them is explicit:** if `--name` is supplied, `add` is fully non-interactive and every other value comes from flags or defaults. If `--name` is absent, every field is prompted for. There is no partial mode — a half-filled flag set does not start a prompt for the rest, because that makes scripted use unpredictable.

- [ ] **Step 1: Write the failing test `cmd/sssh/cmd_add_test.go`**

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

const testKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
-----END OPENSSH PRIVATE KEY-----
`

func TestCmdAddNonInteractiveWithFlags(t *testing.T) {
	a, _ := newTestApp(t, nil)
	err := cmdAdd(a, []string{
		"--name", "prod-web", "--host", "10.0.0.5", "--user", "deploy",
		"--port", "2222", "--auth", "agent", "--folder", "work/prod",
		"--tags", "web,prod", "--forward-agent",
	})
	if err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	p, ok := a.store.Find("prod-web")
	if !ok {
		t.Fatal("profile was not added to the store")
	}
	if p.Host != "10.0.0.5" || p.User != "deploy" || p.Port != 2222 {
		t.Errorf("host/user/port = %s/%s/%d", p.Host, p.User, p.Port)
	}
	if p.Folder != "work/prod" || !p.ForwardAgent {
		t.Errorf("folder/forwardAgent = %s/%v", p.Folder, p.ForwardAgent)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "web" || p.Tags[1] != "prod" {
		t.Errorf("tags = %v, want [web prod]", p.Tags)
	}
}

func TestCmdAddDefaultsPortTo22AndAuthToAgent(t *testing.T) {
	a, _ := newTestApp(t, nil)
	if err := cmdAdd(a, []string{"--name", "x", "--host", "h", "--user", "u"}); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	p, _ := a.store.Find("x")
	if p.Port != 22 {
		t.Errorf("Port = %d, want the default 22", p.Port)
	}
	if p.Auth != config.AuthAgent {
		t.Errorf("Auth = %q, want the default %q", p.Auth, config.AuthAgent)
	}
}

func TestCmdAddPersistsAndRegenerates(t *testing.T) {
	a, _ := newTestApp(t, nil)
	if err := cmdAdd(a, []string{"--name", "x", "--host", "h", "--user", "u"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.Load(a.paths.StoreFile)
	if err != nil {
		t.Fatalf("reloading the store: %v", err)
	}
	if _, ok := reloaded.Find("x"); !ok {
		t.Error("the profile was not written to connections.toml")
	}
	generated, err := readFileString(a.paths.GenConfig)
	if err != nil {
		t.Fatalf("generated config: %v", err)
	}
	if !strings.Contains(generated, "Host x") {
		t.Errorf("ssh config was not regenerated:\n%s", generated)
	}
}

func TestCmdAddAdoptsAKeyFromAPath(t *testing.T) {
	a, _ := newTestApp(t, nil)
	src := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(src, []byte(testKey), 0o600); err != nil {
		t.Fatal(err)
	}
	err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", src})
	if err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	p, _ := a.store.Find("k")
	if p.Key != "k" {
		t.Errorf("Key = %q, want the adopted basename %q", p.Key, "k")
	}
	adopted := filepath.Join(a.paths.KeysDir, "k")
	data, err := os.ReadFile(adopted)
	if err != nil {
		t.Fatalf("key was not adopted into the sandbox: %v", err)
	}
	if string(data) != testKey {
		t.Error("adopted key content mismatch")
	}
}

func TestCmdAddLinkKeepsTheKeyInPlace(t *testing.T) {
	a, _ := newTestApp(t, nil)
	src := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(src, []byte(testKey), 0o600); err != nil {
		t.Fatal(err)
	}
	err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", src, "--link"})
	if err != nil {
		t.Fatalf("cmdAdd --link: %v", err)
	}
	p, _ := a.store.Find("k")
	if p.Key != src {
		t.Errorf("Key = %q, want the original path %q", p.Key, src)
	}
	if _, err := os.Stat(filepath.Join(a.paths.KeysDir, "k")); err == nil {
		t.Error("--link should not copy the key into the sandbox")
	}
}

func TestCmdAddReadsAPastedKeyFromStdin(t *testing.T) {
	a, _ := newTestApp(t, nil)
	a.stdin = strings.NewReader(testKey)
	err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", "-"})
	if err != nil {
		t.Fatalf("cmdAdd with a pasted key: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(a.paths.KeysDir, "k"))
	if err != nil {
		t.Fatalf("pasted key was not stored: %v", err)
	}
	if string(data) != testKey {
		t.Error("pasted key content mismatch")
	}
}

func TestCmdAddRejectsDuplicateNames(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	err := cmdAdd(a, []string{"--name", "prod-web", "--host", "h", "--user", "u"})
	if err == nil {
		t.Fatal("expected an error adding a profile whose name already exists")
	}
	if !strings.Contains(err.Error(), "already") {
		t.Errorf("error %q should say the name is already taken", err)
	}
}

func TestCmdAddRejectsAnInvalidProfileWithoutWriting(t *testing.T) {
	a, _ := newTestApp(t, nil)
	if err := cmdAdd(a, []string{"--name", "bad*name", "--host", "h", "--user", "u"}); err == nil {
		t.Fatal("expected an error for a name containing a pattern character")
	}
	if len(a.store.Profiles) != 0 {
		t.Error("the invalid profile was added to the store anyway")
	}
}

func TestCmdAddRejectsAnUnknownJumpTarget(t *testing.T) {
	a, _ := newTestApp(t, nil)
	err := cmdAdd(a, []string{"--name", "x", "--host", "h", "--user", "u", "--jump", "ghost"})
	if err == nil {
		t.Fatal("expected an error for a jump target that does not exist")
	}
}

func TestCmdAddInteractivePrompts(t *testing.T) {
	a, out := newTestApp(t, nil)
	// name, host, user, port (blank -> 22), auth (blank -> agent),
	// folder, tags, jump, forward agent
	a.stdin = strings.NewReader("web\nexample.com\nme\n\n\nwork\nprod\n\nn\n")
	if err := cmdAdd(a, nil); err != nil {
		t.Fatalf("interactive cmdAdd: %v", err)
	}
	p, ok := a.store.Find("web")
	if !ok {
		t.Fatalf("profile not created; prompts were:\n%s", out.String())
	}
	if p.Host != "example.com" || p.User != "me" {
		t.Errorf("host/user = %s/%s", p.Host, p.User)
	}
	if p.Port != 22 {
		t.Errorf("Port = %d, want 22 from the blank default", p.Port)
	}
	if p.Auth != config.AuthAgent {
		t.Errorf("Auth = %q, want agent from the blank default", p.Auth)
	}
	if p.Folder != "work" {
		t.Errorf("Folder = %q", p.Folder)
	}
	if p.ForwardAgent {
		t.Error("ForwardAgent should be false after answering n")
	}
}

func TestPromptForUsesTheDefaultOnBlankInput(t *testing.T) {
	out := &bytes.Buffer{}
	got, err := promptFor(newReader("\n"), out, "Port", "22")
	if err != nil {
		t.Fatalf("promptFor: %v", err)
	}
	if got != "22" {
		t.Errorf("promptFor = %q, want the default %q", got, "22")
	}
	if !strings.Contains(out.String(), "22") {
		t.Errorf("prompt %q should show the default", out.String())
	}
}

func TestPromptForTrimsInput(t *testing.T) {
	got, err := promptFor(newReader("  value  \n"), &bytes.Buffer{}, "Label", "")
	if err != nil {
		t.Fatalf("promptFor: %v", err)
	}
	if got != "value" {
		t.Errorf("promptFor = %q, want %q", got, "value")
	}
}
```

Add this helper to `cmd/sssh/app_test.go`:

```go
func newReader(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }
```

with `"bufio"` and `"strings"` added to that file's imports.

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./cmd/sssh/ -run TestCmdAdd -v`
Expected: FAIL — `cmdAdd` is undefined.

- [ ] **Step 3: Write `cmd/sssh/cmd_add.go`**

```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/keys"
)

// cmdAdd has exactly two modes, and never a mixture.
//
// With --name, it is fully non-interactive: every other value comes from a flag
// or its documented default. Without --name, every field is prompted for. A
// half-filled flag set does not start prompting for the remainder, because that
// would make the command's behaviour in a script depend on which flags happened
// to be supplied.
func cmdAdd(a *app, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	var (
		name    = fs.String("name", "", "profile name; also the ssh Host alias")
		host    = fs.String("host", "", "hostname or IP address")
		user    = fs.String("user", "", "login user")
		port    = fs.Int("port", 22, "port")
		auth    = fs.String("auth", "agent", "agent, key, or password")
		key     = fs.String("key", "", "path to a private key, or - to read it from stdin")
		link    = fs.Bool("link", false, "reference the key where it is instead of copying it into the sandbox (excludes it from backups)")
		folder  = fs.String("folder", "", "folder path, e.g. work/prod")
		tags    = fs.String("tags", "", "comma-separated tags")
		jump    = fs.String("jump", "", "name of the profile to connect through")
		forward = fs.Bool("forward-agent", false, "forward the ssh agent to this host")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	p := config.Profile{
		Name: *name, Host: *host, User: *user, Port: *port,
		Auth: config.AuthType(*auth), Folder: *folder, Jump: *jump, ForwardAgent: *forward,
	}
	keySource := *key

	if *name == "" {
		var err error
		if p, keySource, err = promptProfile(a); err != nil {
			return err
		}
	} else if *tags != "" {
		p.Tags = splitTags(*tags)
	}

	if _, exists := a.store.Find(p.Name); exists {
		return fmt.Errorf("a profile named %q already exists; use `sssh edit %s` to change it", p.Name, p.Name)
	}

	if p.Auth == config.AuthKey {
		stored, err := resolveKey(a, p.Name, keySource, *link)
		if err != nil {
			return err
		}
		p.Key = stored
	}

	// Validate against the store as it would be, so an unknown jump target or a
	// bad name is caught before anything is written.
	candidate := a.store
	candidate.Profiles = append(append([]config.Profile{}, a.store.Profiles...), p)
	if err := candidate.Validate(); err != nil {
		return err
	}

	a.store = candidate
	if err := a.save(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Added %s (%s@%s:%d). Connect with `sssh %s`.\n", p.Name, p.User, p.Host, p.Port, p.Name)
	return nil
}

// resolveKey turns the --key value into what gets stored in Profile.Key: a bare
// basename for an adopted key, or the original path for a linked one.
func resolveKey(a *app, name, source string, link bool) (string, error) {
	if source == "" {
		return "", fmt.Errorf("auth is %q but no key was given; pass --key PATH, or --key - to paste one", config.AuthKey)
	}
	if link {
		if source == "-" {
			return "", fmt.Errorf("--link cannot be combined with --key - (there is no file to link to)")
		}
		return source, nil
	}
	if source == "-" {
		content, err := io.ReadAll(a.stdin)
		if err != nil {
			return "", fmt.Errorf("reading the key from stdin: %w", err)
		}
		return keys.AdoptContent(a.paths.KeysDir, name, content)
	}
	return keys.AdoptFile(a.paths.KeysDir, name, source, a.paths.Root)
}

// promptProfile asks for each field in turn. Blank answers take the default
// shown in the prompt.
func promptProfile(a *app) (config.Profile, string, error) {
	r := bufio.NewReader(a.stdin)
	var p config.Profile
	var err error

	if p.Name, err = promptFor(r, a.stdout, "Name", ""); err != nil {
		return p, "", err
	}
	if p.Host, err = promptFor(r, a.stdout, "Host", ""); err != nil {
		return p, "", err
	}
	if p.User, err = promptFor(r, a.stdout, "User", ""); err != nil {
		return p, "", err
	}
	portStr, err := promptFor(r, a.stdout, "Port", "22")
	if err != nil {
		return p, "", err
	}
	if p.Port, err = strconv.Atoi(portStr); err != nil {
		return p, "", fmt.Errorf("port %q is not a number", portStr)
	}
	authStr, err := promptFor(r, a.stdout, "Auth (agent/key/password)", "agent")
	if err != nil {
		return p, "", err
	}
	p.Auth = config.AuthType(authStr)

	keySource := ""
	if p.Auth == config.AuthKey {
		if keySource, err = promptFor(r, a.stdout, "Key file path, or - to paste the key", ""); err != nil {
			return p, "", err
		}
		if keySource == "-" {
			fmt.Fprintln(a.stdout, "Paste the private key, then press Ctrl-D:")
		}
	}
	if p.Folder, err = promptFor(r, a.stdout, "Folder (optional)", ""); err != nil {
		return p, "", err
	}
	tagStr, err := promptFor(r, a.stdout, "Tags, comma separated (optional)", "")
	if err != nil {
		return p, "", err
	}
	p.Tags = splitTags(tagStr)
	if p.Jump, err = promptFor(r, a.stdout, "Jump through which profile? (optional)", ""); err != nil {
		return p, "", err
	}
	fwd, err := promptFor(r, a.stdout, "Forward the ssh agent? (y/N)", "n")
	if err != nil {
		return p, "", err
	}
	p.ForwardAgent = strings.EqualFold(fwd, "y") || strings.EqualFold(fwd, "yes")

	return p, keySource, nil
}

// promptFor writes a label and reads one line. A blank answer takes def.
func promptFor(r *bufio.Reader, w io.Writer, label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(w, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("reading %s: %w", label, err)
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return def, nil
	}
	return answer, nil
}

func splitTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}
```

- [ ] **Step 4: Add the dispatch case in `cmd/sssh/main.go`**

Inside the `switch args[0]` block, after the `"ls"` case:

```go
	case "add":
		return withApp(func(a *app) error { return cmdAdd(a, args[1:]) })
```

- [ ] **Step 5: Run the tests**

Run: `go test ./cmd/sssh/ -v`
Expected: all PASS.

- [ ] **Step 6: Run the full gate and commit**

Run: `make check`

```bash
git add cmd/sssh
git commit -m "feat: add the add command with flag and interactive modes"
```

---

### Task 11: `sssh rm` and `sssh sync`

**Files:**
- Create: `cmd/sssh/cmd_rm.go`, `cmd/sssh/cmd_sync.go`, `cmd/sssh/cmd_rm_test.go`
- Modify: `cmd/sssh/main.go` — add the `"rm"` case and delete the temporary `cmdSync` stub; `cmd/sssh/app.go` — add the `confirm` helper

**Interfaces:**
- Consumes: `(*app).save`, `(*app).sync`, `promptFor` (Task 10).
- Produces: `cmdRm(a *app, args []string) error`, `cmdSync(a *app) error`, `confirm(a *app, question string) (bool, error)`

**Two safety rules, both tested:** removing a profile that another profile jumps through is refused, because it would leave a dangling `ProxyJump`. And the adopted key file is **kept** by default — `--purge-key` deletes it, and only when no other profile references it.

- [ ] **Step 1: Write the failing test `cmd/sssh/cmd_rm_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func TestCmdRmRemovesAProfile(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	a.stdin = strings.NewReader("y\n")
	if err := cmdRm(a, []string{"staging"}); err != nil {
		t.Fatalf("cmdRm: %v", err)
	}
	if _, ok := a.store.Find("staging"); ok {
		t.Error("profile is still in the store")
	}
	reloaded, err := config.Load(a.paths.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.Find("staging"); ok {
		t.Error("profile is still in connections.toml")
	}
	generated, _ := readFileString(a.paths.GenConfig)
	if strings.Contains(generated, "Host staging") {
		t.Errorf("ssh config still has the Host block:\n%s", generated)
	}
}

func TestCmdRmAbortsWhenNotConfirmed(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	a.stdin = strings.NewReader("n\n")
	if err := cmdRm(a, []string{"staging"}); err != nil {
		t.Fatalf("cmdRm: %v", err)
	}
	if _, ok := a.store.Find("staging"); !ok {
		t.Error("profile was removed despite the user declining")
	}
}

func TestCmdRmYesSkipsConfirmation(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	a.stdin = strings.NewReader("") // nothing to read; --yes must not prompt
	if err := cmdRm(a, []string{"--yes", "staging"}); err != nil {
		t.Fatalf("cmdRm --yes: %v", err)
	}
	if _, ok := a.store.Find("staging"); ok {
		t.Error("profile was not removed")
	}
}

// Removing a jump host would leave a dangling ProxyJump that only fails at
// connect time, with an error from ssh that does not name the cause.
func TestCmdRmRefusesToRemoveAJumpHostInUse(t *testing.T) {
	profiles := testProfiles()
	profiles[1].Jump = "bastion"
	a, _ := newTestApp(t, profiles)
	err := cmdRm(a, []string{"--yes", "bastion"})
	if err == nil {
		t.Fatal("expected an error removing a profile used as a jump host")
	}
	if !strings.Contains(err.Error(), "prod-web") {
		t.Errorf("error %q should name the profile that depends on it", err)
	}
}

func TestCmdRmKeepsTheKeyByDefault(t *testing.T) {
	a, _ := newTestApp(t, nil)
	a.stdin = strings.NewReader(testKey)
	if err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", "-"}); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(a.paths.KeysDir, "k")
	if err := cmdRm(a, []string{"--yes", "k"}); err != nil {
		t.Fatalf("cmdRm: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("the key was deleted without --purge-key: %v", err)
	}
}

func TestCmdRmPurgeKeyDeletesIt(t *testing.T) {
	a, _ := newTestApp(t, nil)
	a.stdin = strings.NewReader(testKey)
	if err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", "-"}); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(a.paths.KeysDir, "k")
	if err := cmdRm(a, []string{"--yes", "--purge-key", "k"}); err != nil {
		t.Fatalf("cmdRm --purge-key: %v", err)
	}
	if _, err := os.Stat(keyPath); err == nil {
		t.Error("the key still exists after --purge-key")
	}
}

func TestCmdRmPurgeKeyKeepsAKeyStillInUse(t *testing.T) {
	a, _ := newTestApp(t, nil)
	a.stdin = strings.NewReader(testKey)
	if err := cmdAdd(a, []string{"--name", "k", "--host", "h", "--user", "u", "--auth", "key", "--key", "-"}); err != nil {
		t.Fatal(err)
	}
	// A second profile that links the same adopted key by name.
	a.store.Profiles = append(a.store.Profiles, config.Profile{
		Name: "other", Host: "h2", User: "u", Port: 22, Auth: config.AuthKey, Key: "k",
	})
	if err := a.save(); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(a.paths.KeysDir, "k")
	if err := cmdRm(a, []string{"--yes", "--purge-key", "k"}); err != nil {
		t.Fatalf("cmdRm: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Error("a key still referenced by another profile was deleted")
	}
}

func TestCmdRmUnknownProfile(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdRm(a, []string{"--yes", "ghost"}); err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
}

func TestCmdRmRequiresExactlyOneName(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdRm(a, []string{"--yes"}); err == nil {
		t.Fatal("expected an error when no name is given")
	}
	if err := cmdRm(a, []string{"--yes", "a", "b"}); err == nil {
		t.Fatal("expected an error when two names are given")
	}
}

func TestCmdSyncRegeneratesFromTheStore(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := os.Remove(a.paths.GenConfig); err != nil {
		t.Fatal(err)
	}
	if err := cmdSync(a); err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	generated, err := readFileString(a.paths.GenConfig)
	if err != nil {
		t.Fatalf("config was not regenerated: %v", err)
	}
	if !strings.Contains(generated, "Host prod-web") {
		t.Errorf("regenerated config is missing a Host block:\n%s", generated)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./cmd/sssh/ -run 'TestCmdRm|TestCmdSync' -v`
Expected: FAIL — `cmdRm` is undefined.

- [ ] **Step 3: Write `cmd/sssh/cmd_rm.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func cmdRm(a *app, args []string) error {
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	purgeKey := fs.Bool("purge-key", false, "also delete the adopted key file, if nothing else uses it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: sssh rm [--yes] [--purge-key] <name>")
	}
	name := fs.Arg(0)

	p, ok := a.store.Find(name)
	if !ok {
		return errNoProfile(name)
	}

	// A dangling ProxyJump only fails at connect time, with an ssh error that
	// does not point at the cause. Refuse here, where the fix is obvious.
	for _, other := range a.store.Profiles {
		if other.Jump == name {
			return fmt.Errorf("%q is the jump host for %q; change or remove %q first", name, other.Name, other.Name)
		}
	}

	if !*yes {
		confirmed, err := confirm(a, fmt.Sprintf("Remove %s (%s@%s:%d)?", p.Name, p.User, p.Host, p.Port))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(a.stdout, "Nothing removed.")
			return nil
		}
	}

	remaining := make([]config.Profile, 0, len(a.store.Profiles))
	for _, other := range a.store.Profiles {
		if other.Name != name {
			remaining = append(remaining, other)
		}
	}
	a.store.Profiles = remaining
	if err := a.save(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Removed %s.\n", name)

	return handleOrphanedKey(a, p, *purgeKey)
}

// handleOrphanedKey deals with the key file a removed profile used.
//
// Deleting key material is not reversible, so the default is to keep it and say
// where it is. --purge-key deletes it, but only when it was adopted into the
// sandbox (a bare filename) and no remaining profile references it.
func handleOrphanedKey(a *app, removed config.Profile, purge bool) error {
	if removed.Auth != config.AuthKey || removed.Key == "" {
		return nil
	}
	if strings.HasPrefix(removed.Key, "/") || strings.HasPrefix(removed.Key, "~") {
		return nil // linked in place; never ours to delete
	}
	for _, other := range a.store.Profiles {
		if other.Key == removed.Key {
			fmt.Fprintf(a.stdout, "Key %s is still used by %s; keeping it.\n", removed.Key, other.Name)
			return nil
		}
	}
	keyPath := filepath.Join(a.paths.KeysDir, removed.Key)
	if !purge {
		fmt.Fprintf(a.stdout, "Key %s was left in place. Delete it with `sssh rm --purge-key`, or see `sssh doctor`.\n", keyPath)
		return nil
	}
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", keyPath, err)
	}
	fmt.Fprintf(a.stdout, "Deleted %s.\n", keyPath)
	return nil
}

```

Note that `cmd_rm.go` no longer imports `bufio` — `confirm` moves to `app.go` in the next step, because `sssh import` needs it too.

- [ ] **Step 4: Write `cmd/sssh/cmd_sync.go`**

```go
package main

import "fmt"

// cmdSync regenerates the ssh config from the store.
//
// Every mutating command does this already, so sync exists for exactly one
// case: the user hand-edited connections.toml and wants the result applied.
func cmdSync(a *app) error {
	if err := a.sync(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Wrote %s (%d profiles).\n", a.paths.GenConfig, len(a.store.Profiles))
	fmt.Fprintf(a.stdout, "%s includes it.\n", a.paths.SSHConfig)
	return nil
}
```

Add `confirm` to `cmd/sssh/app.go` — both `rm` and `import` use it, so it belongs with the shared plumbing rather than in one command's file:

```go
// confirm asks a yes/no question, defaulting to no. Every destructive command
// routes through it so the default is always the safe answer.
func confirm(a *app, question string) (bool, error) {
	answer, err := promptFor(bufio.NewReader(a.stdin), a.stdout, question+" (y/N)", "n")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}
```

Add `"bufio"` and `"strings"` to `app.go`'s imports.

- [ ] **Step 5: Wire it up in `cmd/sssh/main.go`**

Delete the temporary `cmdSync` stub, and add to the switch:

```go
	case "rm":
		return withApp(func(a *app) error { return cmdRm(a, args[1:]) })
```

- [ ] **Step 6: Run the tests, then the gate, then commit**

Run: `go test ./cmd/sssh/ -v` — all PASS.
Run: `make check`

```bash
git add cmd/sssh
git commit -m "feat: add the rm and sync commands"
```

---

### Task 12: Connecting

**Files:**
- Create: `cmd/sssh/cmd_connect.go`, `cmd/sssh/cmd_connect_test.go`
- Modify: `cmd/sssh/main.go` — delete the temporary `cmdConnect` stub

**Interfaces:**
- Consumes: `launch.Args`, `launch.Exec` (Task 8), `(*app).sync`.
- Produces: `prepareConnect(a *app, name string, extra []string) ([]string, error)`, `cmdConnect(a *app, name string, extra []string) error`

**`cmdConnect` is deliberately split.** `prepareConnect` does everything testable and returns the argv; `cmdConnect` is that plus one `launch.Exec` call. **No test may call `cmdConnect` on a valid profile** — `Exec` replaces the process image and would take the test binary with it, reporting nothing.

- [ ] **Step 1: Write the failing test `cmd/sssh/cmd_connect_test.go`**

```go
package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareConnectBuildsTheArgv(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	argv, err := prepareConnect(a, "prod-web", nil)
	if err != nil {
		t.Fatalf("prepareConnect: %v", err)
	}
	want := []string{"ssh", "prod-web"}
	if !reflect.DeepEqual(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestPrepareConnectPassesThroughARemoteCommand(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	argv, err := prepareConnect(a, "prod-web", []string{"tail", "-f", "/var/log/x"})
	if err != nil {
		t.Fatalf("prepareConnect: %v", err)
	}
	want := []string{"ssh", "prod-web", "tail", "-f", "/var/log/x"}
	if !reflect.DeepEqual(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestPrepareConnectRejectsAnUnknownProfile(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	_, err := prepareConnect(a, "ghost", nil)
	if err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
	if !strings.Contains(err.Error(), "sssh ls") {
		t.Errorf("error %q should point the user at `sssh ls`", err)
	}
}

// A hand-edited connections.toml must take effect without the user having to
// remember `sssh sync`, so connecting regenerates first.
func TestPrepareConnectRegeneratesTheConfigFirst(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := os.Remove(a.paths.GenConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareConnect(a, "prod-web", nil); err != nil {
		t.Fatalf("prepareConnect: %v", err)
	}
	generated, err := readFileString(a.paths.GenConfig)
	if err != nil {
		t.Fatalf("config was not regenerated before connecting: %v", err)
	}
	if !strings.Contains(generated, "Host prod-web") {
		t.Errorf("regenerated config is missing the Host block:\n%s", generated)
	}
}

func TestPrepareConnectEnsuresTheIncludeBlock(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := os.Remove(a.paths.SSHConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareConnect(a, "prod-web", nil); err != nil {
		t.Fatalf("prepareConnect: %v", err)
	}
	sshCfg, err := readFileString(a.paths.SSHConfig)
	if err != nil {
		t.Fatalf("ssh config was not recreated: %v", err)
	}
	if !strings.Contains(sshCfg, "Include ~/.ssh/sssh.d/config") {
		t.Errorf("Include line missing:\n%s", sshCfg)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./cmd/sssh/ -run TestPrepareConnect -v`
Expected: FAIL — `prepareConnect` is undefined.

- [ ] **Step 3: Write `cmd/sssh/cmd_connect.go`**

```go
package main

import (
	"github.com/poppyszn/super-ssh-v2/internal/launch"
)

// prepareConnect resolves the profile, brings the generated config up to date,
// and returns the argv to run.
//
// It is separated from cmdConnect so it can be tested: launch.Exec replaces the
// process image, so a test that called cmdConnect on a real profile would
// vanish mid-run and report nothing.
//
// Regenerating here means a hand-edited connections.toml takes effect on the
// next connect without the user remembering `sssh sync`.
func prepareConnect(a *app, name string, extra []string) ([]string, error) {
	if _, ok := a.store.Find(name); !ok {
		return nil, errNoProfile(name)
	}
	if err := a.sync(); err != nil {
		return nil, err
	}
	return launch.Args(name, extra), nil
}

// cmdConnect replaces this process with ssh. On success it does not return:
// from here on there is no Go code between the user and the remote host, which
// is the entire point of v2.
func cmdConnect(a *app, name string, extra []string) error {
	argv, err := prepareConnect(a, name, extra)
	if err != nil {
		return err
	}
	return launch.Exec(argv)
}
```

- [ ] **Step 4: Delete the stub**

Remove the temporary `cmdConnect` function from `cmd/sssh/main.go`. The dispatch line at the bottom of `run` already calls it.

- [ ] **Step 5: Run the tests, the gate, and commit**

Run: `go test ./... -v` — all PASS.
Run: `make check`

```bash
git add cmd/sssh
git commit -m "feat: connect by exec'ing ssh with the generated config"
```

---

### Task 13: The picker

**Files:**
- Create: `internal/pick/pick.go`, `internal/pick/pick_test.go`, `cmd/sssh/cmd_pick.go`
- Modify: `cmd/sssh/main.go` — delete the temporary `cmdPickAndConnect` stub

**Interfaces:**
- Consumes: `config.Profile`.
- Produces:
  - `pick.ErrCancelled` (a sentinel `error`)
  - `pick.Available() bool`
  - `pick.Lines(profiles []config.Profile) []string`
  - `pick.ParseSelection(line string) string`
  - `pick.ParseChoice(input string, profiles []config.Profile) (string, error)`
  - `pick.Fallback(profiles []config.Profile, stdin io.Reader, stdout io.Writer) (string, error)`
  - `pick.WithFzf(profiles []config.Profile) (string, error)`
  - `pick.Choose(profiles []config.Profile, stdin io.Reader, stdout io.Writer) (string, error)`
  - `cmdPickAndConnect(a *app) error`

**What this package must never grow:** raw terminal mode, incremental filtering, scrolling, colour, or a preview pane. fzf does all of that, is maintained by someone else, and costs zero dependency budget because it is an optional external binary rather than a Go import. The fallback is a numbered list and one `ReadString` — that is the whole feature.

**Tests must not invoke fzf.** `WithFzf` spawns a subprocess that wants a terminal; CI has none. Test `Lines`, `ParseSelection`, `ParseChoice`, and `Fallback`, which are pure or stream-driven.

- [ ] **Step 1: Write the failing test `internal/pick/pick_test.go`**

```go
package pick

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func profiles() []config.Profile {
	return []config.Profile{
		{Name: "bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
		{Name: "prod-web", Host: "10.0.0.5", User: "deploy", Port: 2222, Auth: config.AuthAgent,
			Folder: "work/prod", Tags: []string{"web"}},
	}
}

// The name must come first and be tab-separated, because ParseSelection
// recovers it by splitting on the first tab.
func TestLinesPutTheNameFirst(t *testing.T) {
	lines := Lines(profiles())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if !strings.HasPrefix(lines[0], "bastion\t") {
		t.Errorf("line %q should start with the name and a tab", lines[0])
	}
	if !strings.Contains(lines[1], "deploy@10.0.0.5:2222") {
		t.Errorf("line %q is missing the connection detail", lines[1])
	}
	if !strings.Contains(lines[1], "work/prod") {
		t.Errorf("line %q is missing the folder", lines[1])
	}
}

func TestLinesRoundTripThroughParseSelection(t *testing.T) {
	for i, line := range Lines(profiles()) {
		want := profiles()[i].Name
		if got := ParseSelection(line); got != want {
			t.Errorf("ParseSelection(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestParseSelectionHandlesTrailingNewline(t *testing.T) {
	if got := ParseSelection("prod-web\tdeploy@h:22\t-\t\n"); got != "prod-web" {
		t.Errorf("ParseSelection = %q, want %q", got, "prod-web")
	}
}

func TestParseChoice(t *testing.T) {
	ps := profiles()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"by number", "1", "bastion", false},
		{"by second number", "2", "prod-web", false},
		{"by name", "prod-web", "prod-web", false},
		{"with surrounding space", "  2  ", "prod-web", false},
		{"number too low", "0", "", true},
		{"number too high", "3", "", true},
		{"unknown name", "ghost", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChoice(tt.input, ps)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseChoice(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseChoice(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseChoiceBlankInputCancels(t *testing.T) {
	_, err := ParseChoice("", profiles())
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("blank input gave %v, want ErrCancelled", err)
	}
}

func TestFallbackPrintsANumberedListAndReadsAChoice(t *testing.T) {
	out := &bytes.Buffer{}
	got, err := Fallback(profiles(), strings.NewReader("2\n"), out)
	if err != nil {
		t.Fatalf("Fallback: %v", err)
	}
	if got != "prod-web" {
		t.Errorf("Fallback = %q, want %q", got, "prod-web")
	}
	rendered := out.String()
	for _, want := range []string{"1", "bastion", "2", "prod-web", "admin@b.example.com:22"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("output missing %q:\n%s", want, rendered)
		}
	}
}

func TestFallbackAcceptsAName(t *testing.T) {
	got, err := Fallback(profiles(), strings.NewReader("bastion\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Fallback: %v", err)
	}
	if got != "bastion" {
		t.Errorf("Fallback = %q, want %q", got, "bastion")
	}
}

func TestFallbackCancelsOnBlankInput(t *testing.T) {
	_, err := Fallback(profiles(), strings.NewReader("\n"), &bytes.Buffer{})
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("got %v, want ErrCancelled", err)
	}
}

func TestFallbackCancelsOnClosedInput(t *testing.T) {
	_, err := Fallback(profiles(), strings.NewReader(""), &bytes.Buffer{})
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("got %v, want ErrCancelled", err)
	}
}

func TestChooseOnAnEmptyStore(t *testing.T) {
	_, err := Choose(nil, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error choosing from an empty store")
	}
	if !strings.Contains(err.Error(), "sssh add") {
		t.Errorf("error %q should point at `sssh add`", err)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/pick/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/pick/pick.go`**

```go
// Package pick chooses a connection.
//
// It delegates to fzf when fzf is installed, and otherwise prints a numbered
// list and reads one line. It must never grow raw terminal mode, incremental
// filtering, scrolling, colour, or a preview pane: fzf already does all of that
// and is maintained by somebody else. "Just a small picker" is how v1 started,
// and it ended with a vendored terminal emulator.
package pick

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

// ErrCancelled means the user backed out. Callers exit quietly: cancelling a
// picker is not a failure and deserves no error message.
var ErrCancelled = errors.New("cancelled")

// Available reports whether fzf is installed.
func Available() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// Lines renders profiles for fzf. The name comes first, tab-separated, so
// ParseSelection can recover it from whatever fzf echoes back.
func Lines(profiles []config.Profile) []string {
	out := make([]string, 0, len(profiles))
	for _, p := range profiles {
		folder := p.Folder
		if folder == "" {
			folder = "-"
		}
		out = append(out, fmt.Sprintf("%s\t%s@%s:%d\t%s\t%s",
			p.Name, p.User, p.Host, p.Port, folder, strings.Join(p.Tags, ",")))
	}
	return out
}

// ParseSelection extracts the profile name from a line fzf returned.
func ParseSelection(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(trimmed, "\t", 2)[0])
}

// ParseChoice maps a typed answer -- a 1-based number or a profile name -- onto
// a profile name.
func ParseChoice(input string, profiles []config.Profile) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", ErrCancelled
	}
	if n, err := strconv.Atoi(input); err == nil {
		if n < 1 || n > len(profiles) {
			return "", fmt.Errorf("%d is out of range (1-%d)", n, len(profiles))
		}
		return profiles[n-1].Name, nil
	}
	for _, p := range profiles {
		if p.Name == input {
			return p.Name, nil
		}
	}
	return "", fmt.Errorf("no profile named %q", input)
}

// Fallback prints a numbered list and reads one line. This is the entire
// non-fzf experience, on purpose.
func Fallback(profiles []config.Profile, stdin io.Reader, stdout io.Writer) (string, error) {
	for i, p := range profiles {
		fmt.Fprintf(stdout, "%3d  %-20s %s@%s:%d\n", i+1, p.Name, p.User, p.Host, p.Port)
	}
	fmt.Fprint(stdout, "\nConnect to (number or name, blank to cancel): ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", ErrCancelled // input closed, e.g. Ctrl-D
	}
	return ParseChoice(line, profiles)
}

// WithFzf pipes the list through fzf.
//
// The list arrives on fzf's stdin; fzf reads keystrokes from /dev/tty itself,
// so this does not need a terminal on stdin. fzf exits 1 when nothing matched
// and 130 when the user pressed Esc or Ctrl-C, and neither is a failure.
func WithFzf(profiles []config.Profile) (string, error) {
	cmd := exec.Command("fzf",
		"--delimiter=\t", "--with-nth=1,2,3,4",
		"--prompt=ssh > ", "--height=40%", "--reverse", "--no-multi")
	cmd.Stdin = strings.NewReader(strings.Join(Lines(profiles), "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrCancelled
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}
	name := ParseSelection(string(out))
	if name == "" {
		return "", ErrCancelled
	}
	return name, nil
}

// Choose uses fzf when it is installed and the numbered fallback otherwise.
func Choose(profiles []config.Profile, stdin io.Reader, stdout io.Writer) (string, error) {
	if len(profiles) == 0 {
		return "", errors.New("no connections saved yet; add one with `sssh add`, or adopt your existing hosts with `sssh import`")
	}
	if Available() {
		return WithFzf(profiles)
	}
	return Fallback(profiles, stdin, stdout)
}
```

- [ ] **Step 4: Write `cmd/sssh/cmd_pick.go`**

```go
package main

import (
	"errors"

	"github.com/poppyszn/super-ssh-v2/internal/pick"
)

// cmdPickAndConnect is what a bare `sssh` runs.
func cmdPickAndConnect(a *app) error {
	name, err := pick.Choose(a.store.Profiles, a.stdin, a.stdout)
	if errors.Is(err, pick.ErrCancelled) {
		return nil // backing out is not a failure
	}
	if err != nil {
		return err
	}
	return cmdConnect(a, name, nil)
}
```

Then delete the temporary `cmdPickAndConnect` stub from `cmd/sssh/main.go`.

- [ ] **Step 5: Run the tests, the gate, and commit**

Run: `go test ./... -v` — all PASS.
Run: `make check`

```bash
git add internal/pick cmd/sssh
git commit -m "feat: choose a connection through fzf with a numbered fallback"
```

---

### Task 14: `sssh import`

**Files:**
- Create: `internal/importer/importer.go`, `internal/importer/importer_test.go`, `cmd/sssh/cmd_import.go`, `cmd/sssh/cmd_import_test.go`
- Modify: `cmd/sssh/main.go` — add the `"import"` case

**Interfaces:**
- Consumes: `sshconfig.RemoveBlock`, `keys.AdoptFile`, `config.Profile`, `(*app).save`.
- Produces:
  - `type importer.Found struct { Name, Host, User string; Port int; IdentityFile, ProxyJump string; ForwardAgent bool }`
  - `importer.Parse(content string) (found []Found, skipped []string)`
  - `cmdImport(a *app, args []string) error`

**Two rules stated so they are not guessed at:** a `Host` line naming a pattern (`*`, `?`, `!`) is skipped and reported, because it is a defaults block rather than a connection. A `Host` line naming several aliases contributes one profile, using the **first** alias.

- [ ] **Step 1: Write the failing test `internal/importer/importer_test.go`**

```go
package importer

import "testing"

const sample = `# my ssh config
Host *
    ServerAliveInterval 60

Host bastion
    HostName bastion.example.com
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_bastion

Host prod-web prod-web-alt
    HostName 10.0.0.5
    User deploy
    ProxyJump bastion
    ForwardAgent yes

Host db?
    HostName db.example.com

Match user root
    HostName nowhere.example.com

Host lowercase-keys
    hostname h.example.com
    user u
    port 2200
`

func TestParseFindsConcreteHosts(t *testing.T) {
	found, _ := Parse(sample)
	byName := map[string]Found{}
	for _, f := range found {
		byName[f.Name] = f
	}

	b, ok := byName["bastion"]
	if !ok {
		t.Fatalf("bastion not found; got %v", byName)
	}
	if b.Host != "bastion.example.com" || b.User != "admin" || b.Port != 2222 {
		t.Errorf("bastion = %+v", b)
	}
	if b.IdentityFile != "~/.ssh/id_bastion" {
		t.Errorf("bastion IdentityFile = %q", b.IdentityFile)
	}

	p, ok := byName["prod-web"]
	if !ok {
		t.Fatal("prod-web not found")
	}
	if p.Port != 22 {
		t.Errorf("prod-web Port = %d, want the default 22", p.Port)
	}
	if p.ProxyJump != "bastion" || !p.ForwardAgent {
		t.Errorf("prod-web = %+v", p)
	}

	// Keywords are case-insensitive in ssh config.
	l, ok := byName["lowercase-keys"]
	if !ok {
		t.Fatal("lowercase-keys not found")
	}
	if l.Host != "h.example.com" || l.User != "u" || l.Port != 2200 {
		t.Errorf("lowercase-keys = %+v", l)
	}
}

// A pattern Host is a defaults block, not a connection.
func TestParseSkipsPatternHostsAndReportsThem(t *testing.T) {
	found, skipped := Parse(sample)
	for _, f := range found {
		if f.Name == "*" || f.Name == "db?" {
			t.Errorf("pattern host %q was imported", f.Name)
		}
	}
	if len(skipped) != 2 {
		t.Errorf("skipped = %v, want two entries (* and db?)", skipped)
	}
}

func TestParseSkipsMatchBlocks(t *testing.T) {
	found, _ := Parse(sample)
	for _, f := range found {
		if f.Host == "nowhere.example.com" {
			t.Error("a Match block was imported as a host")
		}
	}
}

// A multi-alias Host line contributes one profile, named by its first alias.
func TestParseUsesTheFirstAliasOfAMultiAliasHost(t *testing.T) {
	found, _ := Parse(sample)
	for _, f := range found {
		if f.Name == "prod-web-alt" {
			t.Error("the second alias became its own profile")
		}
	}
}

func TestParseAcceptsEqualsSeparators(t *testing.T) {
	found, _ := Parse("Host eq\n    HostName=h.example.com\n    User=u\n    Port=2022\n")
	if len(found) != 1 {
		t.Fatalf("got %d hosts, want 1", len(found))
	}
	if found[0].Host != "h.example.com" || found[0].User != "u" || found[0].Port != 2022 {
		t.Errorf("found = %+v", found[0])
	}
}

func TestParseDefaultsHostNameToTheAlias(t *testing.T) {
	found, _ := Parse("Host myserver\n    User u\n")
	if len(found) != 1 {
		t.Fatalf("got %d hosts, want 1", len(found))
	}
	if found[0].Host != "myserver" {
		t.Errorf("Host = %q, want the alias when HostName is absent", found[0].Host)
	}
}

func TestParseEmptyInput(t *testing.T) {
	found, skipped := Parse("")
	if len(found) != 0 || len(skipped) != 0 {
		t.Errorf("found = %v, skipped = %v, want both empty", found, skipped)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/importer/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/importer/importer.go`**

```go
// Package importer reads an existing ~/.ssh/config so its hosts can be adopted
// into sssh. This is both the migration path off a hand-maintained config and
// how "one file holds everything" becomes true: what you adopt is what gets
// backed up.
package importer

import (
	"strconv"
	"strings"
)

// Found is one concrete Host entry.
type Found struct {
	Name         string
	Host         string
	User         string
	Port         int
	IdentityFile string
	ProxyJump    string
	ForwardAgent bool
}

// Parse returns the concrete hosts in an ssh config, plus the names of entries
// that were skipped.
//
// Skipped entries are pattern hosts -- anything containing *, ? or ! -- which
// are defaults blocks rather than connections, and would generate a Host block
// matching hosts the user never configured. Match blocks are skipped for the
// same reason. A Host line naming several aliases contributes one Found, using
// the first alias.
func Parse(content string) (found []Found, skipped []string) {
	var current *Found
	flush := func() {
		if current != nil {
			found = append(found, *current)
			current = nil
		}
	}

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keyword, value := splitDirective(line)

		switch strings.ToLower(keyword) {
		case "host":
			flush()
			aliases := strings.Fields(value)
			if len(aliases) == 0 {
				continue
			}
			if hasPattern(aliases[0]) {
				skipped = append(skipped, aliases[0])
				continue
			}
			current = &Found{Name: aliases[0], Host: aliases[0], Port: 22}
		case "match":
			flush() // a Match block configures conditions, not a connection
		case "hostname":
			if current != nil {
				current.Host = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				if n, err := strconv.Atoi(value); err == nil {
					current.Port = n
				}
			}
		case "identityfile":
			if current != nil {
				current.IdentityFile = value
			}
		case "proxyjump":
			if current != nil {
				current.ProxyJump = value
			}
		case "forwardagent":
			if current != nil {
				current.ForwardAgent = strings.EqualFold(value, "yes")
			}
		}
	}
	flush()
	return found, skipped
}

// splitDirective handles both "Keyword value" and "Keyword=value", which ssh
// accepts interchangeably.
func splitDirective(line string) (keyword, value string) {
	if i := strings.IndexAny(line, " \t="); i >= 0 {
		keyword = line[:i]
		value = strings.TrimSpace(strings.TrimLeft(line[i:], " \t="))
		return keyword, value
	}
	return line, ""
}

func hasPattern(alias string) bool {
	return strings.ContainsAny(alias, "*?!")
}
```

- [ ] **Step 4: Write the failing test `cmd/sssh/cmd_import_test.go`**

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
)

// writeUserSSHConfig puts a hand-written config in place for import to read.
func writeUserSSHConfig(t *testing.T, a *app, body string) {
	t.Helper()
	if err := os.WriteFile(a.paths.SSHConfig, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCmdImportDryRunChangesNothing(t *testing.T) {
	a, out := newTestApp(t, nil)
	writeUserSSHConfig(t, a, "Host myserver\n    HostName example.com\n    User me\n")
	if err := cmdImport(a, []string{"--dry-run"}); err != nil {
		t.Fatalf("cmdImport --dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "myserver") {
		t.Errorf("dry run should list what it found:\n%s", out.String())
	}
	if len(a.store.Profiles) != 0 {
		t.Error("--dry-run added profiles to the store")
	}
}

func TestCmdImportAllAdoptsHosts(t *testing.T) {
	a, _ := newTestApp(t, nil)
	writeUserSSHConfig(t, a, "Host myserver\n    HostName example.com\n    User me\n    Port 2222\n")
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatalf("cmdImport --all: %v", err)
	}
	p, ok := a.store.Find("myserver")
	if !ok {
		t.Fatal("host was not imported")
	}
	if p.Host != "example.com" || p.User != "me" || p.Port != 2222 {
		t.Errorf("imported profile = %+v", p)
	}
}

func TestCmdImportAdoptsTheReferencedKey(t *testing.T) {
	a, _ := newTestApp(t, nil)
	keyPath := filepath.Join(a.paths.Root, "mykey")
	if err := os.WriteFile(keyPath, []byte(testKey), 0o600); err != nil {
		t.Fatal(err)
	}
	writeUserSSHConfig(t, a, "Host k\n    HostName h\n    User u\n    IdentityFile ~/mykey\n")
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	p, _ := a.store.Find("k")
	if p.Auth != "key" || p.Key != "k" {
		t.Errorf("profile = %+v, want auth=key and the adopted basename", p)
	}
	if _, err := os.Stat(filepath.Join(a.paths.KeysDir, "k")); err != nil {
		t.Errorf("key was not copied into the sandbox: %v", err)
	}
}

// A referenced key that does not exist must not abort the whole import.
func TestCmdImportFallsBackToAgentWhenTheKeyIsMissing(t *testing.T) {
	a, out := newTestApp(t, nil)
	writeUserSSHConfig(t, a, "Host k\n    HostName h\n    User u\n    IdentityFile ~/gone\n")
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	p, ok := a.store.Find("k")
	if !ok {
		t.Fatal("host was not imported")
	}
	if p.Auth != "agent" {
		t.Errorf("Auth = %q, want a fallback to agent", p.Auth)
	}
	if !strings.Contains(out.String(), "gone") {
		t.Errorf("the missing key should be reported:\n%s", out.String())
	}
}

// Re-running import must be safe.
func TestCmdImportSkipsProfilesAlreadyManaged(t *testing.T) {
	a, out := newTestApp(t, nil)
	writeUserSSHConfig(t, a, "Host myserver\n    HostName example.com\n    User me\n")
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatalf("second import: %v", err)
	}
	if len(a.store.Profiles) != 1 {
		t.Errorf("got %d profiles after two imports, want 1", len(a.store.Profiles))
	}
}

// sssh's own managed block must not be read back as user content: the Include
// line inside it is not a host, and the block is present in every config sssh
// has ever touched.
func TestCmdImportIgnoresTheManagedBlock(t *testing.T) {
	a, _ := newTestApp(t, nil)
	writeUserSSHConfig(t, a, sshconfig.Block+"\nHost myserver\n    HostName example.com\n    User me\n")
	if err := cmdImport(a, []string{"--all"}); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	if len(a.store.Profiles) != 1 {
		t.Fatalf("got %d profiles, want just myserver: %+v", len(a.store.Profiles), a.store.Profiles)
	}
	if _, ok := a.store.Find("myserver"); !ok {
		t.Error("the real host was not imported")
	}
	for _, p := range a.store.Profiles {
		if strings.Contains(p.Name, "sssh.d") || p.Name == "Include" {
			t.Errorf("part of the managed block was imported as a profile: %+v", p)
		}
	}
}

func TestCmdImportWithNoUserConfig(t *testing.T) {
	a, out := newTestApp(t, nil)
	if err := os.Remove(a.paths.SSHConfig); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := cmdImport(a, []string{"--dry-run"}); err != nil {
		t.Fatalf("cmdImport with no config should not error: %v", err)
	}
	if !strings.Contains(out.String(), "nothing to import") {
		t.Errorf("output should say there is nothing to import:\n%s", out.String())
	}
}
```

- [ ] **Step 5: Write `cmd/sssh/cmd_import.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/importer"
	"github.com/poppyszn/super-ssh-v2/internal/keys"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
)

func cmdImport(a *app, args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	all := fs.Bool("all", false, "adopt every host without asking")
	dryRun := fs.Bool("dry-run", false, "list what would be imported and stop")
	if err := fs.Parse(args); err != nil {
		return err
	}

	raw, err := os.ReadFile(a.paths.SSHConfig)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", a.paths.SSHConfig, err)
	}
	// Strip sssh's own block so its Include line is never read back as user
	// content.
	found, skipped := importer.Parse(sshconfig.RemoveBlock(string(raw)))

	candidates := make([]importer.Found, 0, len(found))
	for _, f := range found {
		if _, exists := a.store.Find(f.Name); exists {
			continue
		}
		candidates = append(candidates, f)
	}

	for _, name := range skipped {
		fmt.Fprintf(a.stdout, "Skipping %q: it is a pattern, not a single host.\n", name)
	}
	if len(candidates) == 0 {
		fmt.Fprintln(a.stdout, "Found nothing to import.")
		return nil
	}

	fmt.Fprintf(a.stdout, "Found %d host(s) to import:\n", len(candidates))
	for _, f := range candidates {
		fmt.Fprintf(a.stdout, "  %-20s %s@%s:%d\n", f.Name, f.User, f.Host, f.Port)
	}
	if *dryRun {
		return nil
	}
	if !*all {
		confirmed, err := confirm(a, fmt.Sprintf("Import these %d host(s)?", len(candidates)))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(a.stdout, "Nothing imported.")
			return nil
		}
	}

	imported := 0
	for _, f := range candidates {
		p := config.Profile{
			Name: f.Name, Host: f.Host, User: f.User, Port: f.Port,
			Auth: config.AuthAgent, Jump: f.ProxyJump, ForwardAgent: f.ForwardAgent,
		}
		if p.User == "" {
			fmt.Fprintf(a.stdout, "Skipping %s: no User in the config and none to guess.\n", f.Name)
			continue
		}
		// A jump target outside the store would fail validation. Drop it and
		// say so rather than refusing the whole import.
		if p.Jump != "" {
			if _, ok := a.store.Find(p.Jump); !ok && !containsName(candidates, p.Jump) {
				fmt.Fprintf(a.stdout, "%s: dropping ProxyJump %q, which is not a saved profile.\n", f.Name, p.Jump)
				p.Jump = ""
			}
		}
		if f.IdentityFile != "" {
			stored, err := keys.AdoptFile(a.paths.KeysDir, f.Name, f.IdentityFile, a.paths.Root)
			if err != nil {
				fmt.Fprintf(a.stdout, "%s: could not adopt %s (%v); falling back to agent auth.\n", f.Name, f.IdentityFile, err)
			} else {
				p.Auth = config.AuthKey
				p.Key = stored
			}
		}
		a.store.Profiles = append(a.store.Profiles, p)
		imported++
	}

	if imported == 0 {
		fmt.Fprintln(a.stdout, "Nothing imported.")
		return nil
	}
	if err := a.save(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Imported %d host(s). Run `sssh ls` to see them.\n", imported)
	return nil
}

func containsName(candidates []importer.Found, name string) bool {
	for _, c := range candidates {
		if c.Name == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 6: Add the dispatch case, run everything, commit**

In `cmd/sssh/main.go`:

```go
	case "import":
		return withApp(func(a *app) error { return cmdImport(a, args[1:]) })
```

Run: `go test ./... -v` — all PASS. Run: `make check`

```bash
git add internal/importer cmd/sssh
git commit -m "feat: import hosts from an existing ssh config"
```

---

### Task 15: Backup and restore

**Files:**
- Create: `internal/backup/archive.go`, `internal/backup/restore.go`, `internal/backup/backup_test.go`, `cmd/sssh/cmd_backup.go`
- Modify: `cmd/sssh/main.go` — add the `"backup"` and `"restore"` cases

**Interfaces:**
- Consumes: `paths.Paths`, `paths.EnsureDirs`.
- Produces:
  - `backup.Create(p paths.Paths, out, passphrase string) error`
  - `backup.Restore(in string, p paths.Paths, passphrase string, force bool) error`
  - `cmdBackup(a *app, args []string) error`, `cmdRestore(a *app, args []string) error`

**Archive layout, fixed:**

```
sssh-backup/MANIFEST.toml     version, created_at, tool version
sssh-backup/connections.toml  -> p.StoreFile
sssh-backup/ssh.d/...         -> p.Sandbox/... (config and keys, modes preserved)
sssh-backup/known_hosts       -> merged into p.KnownHosts, never overwritten
```

**Stream order is `file ← age ← gzip ← tar`.** Encrypt outermost so nothing readable touches disk; compress before encrypting, because encrypted bytes do not compress.

**Path-traversal defence is mandatory.** Every tar entry name must begin with `sssh-backup/`, must not contain `..`, and must not be absolute. A tar archive is attacker-controlled input the moment a user restores a file someone sent them; without this check an entry named `../../.bashrc` writes outside the sandbox. This is the Zip Slip vulnerability and it must have its own test.

- [ ] **Step 1: Write the failing test `internal/backup/backup_test.go`**

```go
package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

const pass = "correct horse battery staple"

// populated builds a realistic tree: store, generated config, a key, and
// known_hosts.
func populated(t *testing.T) paths.Paths {
	t.Helper()
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string, mode os.FileMode) {
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(p.StoreFile, "version = 1\n", 0o600)
	write(p.GenConfig, "# Managed by sssh.\n", 0o600)
	write(filepath.Join(p.KeysDir, "prod-web"), "-----BEGIN OPENSSH PRIVATE KEY-----\nx\n", 0o600)
	write(p.KnownHosts, "host1 ssh-ed25519 AAAA\n", 0o644)
	return p
}

func TestCreateThenRestoreRoundTrips(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatalf("Create: %v", err)
	}

	dst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, pass, false); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	for _, tc := range []struct{ path, want string }{
		{dst.StoreFile, "version = 1\n"},
		{dst.GenConfig, "# Managed by sssh.\n"},
		{filepath.Join(dst.KeysDir, "prod-web"), "-----BEGIN OPENSSH PRIVATE KEY-----\nx\n"},
	} {
		got, err := os.ReadFile(tc.path)
		if err != nil {
			t.Errorf("%s missing after restore: %v", tc.path, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("%s = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// Key permissions must survive the round trip: ssh refuses a key that is
// group- or world-readable.
func TestRestorePreservesKeyPermissions(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, pass, false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dst.KeysDir, "prod-web"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("restored key mode = %o, want 600", perm)
	}
}

func TestRestoreRejectsTheWrongPassphrase(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, "wrong passphrase", false); err == nil {
		t.Fatal("expected an error for the wrong passphrase")
	}
}

func TestCreateProducesAnEncryptedFile(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "age-encryption.org/") {
		t.Error("the archive does not carry an age header")
	}
	if strings.Contains(string(data), "PRIVATE KEY") {
		t.Error("key material is readable in the archive")
	}
}

// Restoring over an existing sandbox must not silently destroy it.
func TestRestoreRefusesANonEmptySandbox(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := populated(t) // already has content
	err := Restore(archive, dst, pass, false)
	if err == nil {
		t.Fatal("expected Restore to refuse a non-empty sandbox")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error %q should mention --force", err)
	}
}

// --force moves the old sandbox aside rather than deleting it.
func TestRestoreForceBacksUpTheOldSandbox(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := populated(t)
	marker := filepath.Join(dst.KeysDir, "old-key-i-still-need")
	if err := os.WriteFile(marker, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nold\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, pass, true); err != nil {
		t.Fatalf("Restore --force: %v", err)
	}
	entries, err := os.ReadDir(dst.SSHDir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "sssh.d.bak-") {
			found = true
		}
	}
	if !found {
		t.Errorf("--force did not move the old sandbox aside; %s contains %v", dst.SSHDir, entries)
	}
}

// known_hosts is shared state: merge, never overwrite.
func TestRestoreMergesKnownHosts(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	existing := "host2 ssh-rsa BBBB\n"
	if err := os.WriteFile(dst.KnownHosts, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, pass, false); err != nil {
		t.Fatal(err)
	}
	merged, err := os.ReadFile(dst.KnownHosts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(merged), "host1") {
		t.Error("the archived known_hosts entry is missing")
	}
	if !strings.Contains(string(merged), "host2") {
		t.Error("the machine's existing known_hosts entry was discarded")
	}
}

func TestRestoreDeduplicatesKnownHosts(t *testing.T) {
	src := populated(t)
	archive := filepath.Join(t.TempDir(), "snapshot.sssh")
	if err := Create(src, archive, pass); err != nil {
		t.Fatal(err)
	}
	dst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst.KnownHosts, []byte("host1 ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Restore(archive, dst, pass, false); err != nil {
		t.Fatal(err)
	}
	merged, _ := os.ReadFile(dst.KnownHosts)
	if strings.Count(string(merged), "host1 ssh-ed25519 AAAA") != 1 {
		t.Errorf("duplicate entry not collapsed:\n%s", merged)
	}
}

// A restored archive is attacker-controlled input. An entry naming a path
// outside the sandbox must be refused, not written.
func TestSafeArchivePath(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"normal file", "sssh-backup/connections.toml", false},
		{"nested file", "sssh-backup/ssh.d/keys/prod", false},
		{"parent traversal", "sssh-backup/../../.bashrc", true},
		{"bare traversal", "../.bashrc", true},
		{"absolute path", "/etc/passwd", true},
		{"outside the root", "other/file", true},
		{"embedded traversal", "sssh-backup/a/../../../x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeArchivePath(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeArchivePath(%q) error = %v, wantErr %v", tt.entry, err, tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/backup/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/backup/archive.go`**

```go
// Package backup writes and reads the single encrypted file that makes a new
// machine a restore away.
//
// The stream is file <- age <- gzip <- tar: compress before encrypting, because
// encrypted bytes do not compress, and encrypt outermost so nothing readable
// ever reaches disk.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

const (
	archiveRoot  = "sssh-backup"
	manifestName = archiveRoot + "/MANIFEST.toml"
	storeName    = archiveRoot + "/connections.toml"
	sandboxDir   = archiveRoot + "/ssh.d"
	knownHosts   = archiveRoot + "/known_hosts"

	// scryptWorkFactor is age's cost parameter. 18 takes roughly a second on a
	// modern laptop, which is the point: it is what makes a guessable
	// passphrase expensive to attack.
	scryptWorkFactor = 18
)

// Create writes an encrypted snapshot of everything sssh owns, plus
// known_hosts.
func Create(p paths.Paths, out, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("a passphrase is required")
	}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("preparing encryption: %w", err)
	}
	recipient.SetWorkFactor(scryptWorkFactor)

	encrypted, err := age.Encrypt(f, recipient)
	if err != nil {
		return fmt.Errorf("starting encryption: %w", err)
	}
	gz := gzip.NewWriter(encrypted)
	tw := tar.NewWriter(gz)

	manifest := fmt.Sprintf("version = 1\ncreated_at = %q\ntool = \"sssh\"\n", time.Now().UTC().Format(time.RFC3339))
	if err := addBytes(tw, manifestName, []byte(manifest), 0o600); err != nil {
		return err
	}
	if err := addFile(tw, storeName, p.StoreFile); err != nil {
		return err
	}
	if err := addTree(tw, sandboxDir, p.Sandbox); err != nil {
		return err
	}
	if err := addFile(tw, knownHosts, p.KnownHosts); err != nil {
		return err
	}

	// Close in order: tar, then gzip, then age. Each flushes into the next.
	if err := tw.Close(); err != nil {
		return fmt.Errorf("finishing the archive: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("finishing compression: %w", err)
	}
	if err := encrypted.Close(); err != nil {
		return fmt.Errorf("finishing encryption: %w", err)
	}
	return nil
}

func addBytes(tw *tar.Writer, name string, data []byte, mode int64) error {
	hdr := &tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg, ModTime: time.Now()}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing the header for %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}

// addFile adds one file. A missing file is skipped rather than fatal: a fresh
// install may have no known_hosts yet.
func addFile(tw *tar.Writer, name, src string) error {
	info, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	return addBytes(tw, name, data, int64(info.Mode().Perm()))
}

// addTree adds a directory recursively, preserving file modes so restored keys
// come back at 0600.
func addTree(tw *tar.Writer, prefix, root string) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(root, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, srcPath)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := path.Join(prefix, filepath.ToSlash(rel))
		if info.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Name: name + "/", Mode: int64(info.Mode().Perm()),
				Typeflag: tar.TypeDir, ModTime: info.ModTime(),
			})
		}
		if !info.Mode().IsRegular() {
			return nil // skip symlinks and anything exotic
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", srcPath, err)
		}
		return addBytes(tw, name, data, int64(info.Mode().Perm()))
	})
}

// safeArchivePath validates a tar entry name and returns its path relative to
// the archive root.
//
// A restored archive is attacker-controlled input the moment someone restores a
// file another person sent them. Without this check, an entry named
// "sssh-backup/../../.bashrc" writes outside the sandbox. This is the Zip Slip
// vulnerability.
func safeArchivePath(name string) (string, error) {
	clean := path.Clean(name)
	if path.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("archive entry %q escapes the archive root", name)
	}
	if clean != archiveRoot && !strings.HasPrefix(clean, archiveRoot+"/") {
		return "", fmt.Errorf("archive entry %q is outside %s/", name, archiveRoot)
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(clean, archiveRoot), "/")
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("archive entry %q contains a parent reference", name)
	}
	return rel, nil
}
```

- [ ] **Step 4: Write `internal/backup/restore.go`**

```go
package backup

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// Restore reproduces an archive into p.
//
// It writes only inside the sandbox, with one deliberate exception:
// known_hosts, which lives outside it and is merged rather than overwritten so
// a restore never discards a host the machine already trusts.
func Restore(in string, p paths.Paths, passphrase string, force bool) error {
	if err := prepareSandbox(p, force); err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}

	f, err := os.Open(in)
	if err != nil {
		return fmt.Errorf("opening %s: %w", in, err)
	}
	defer f.Close()

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return fmt.Errorf("preparing decryption: %w", err)
	}
	decrypted, err := age.Decrypt(f, identity)
	if err != nil {
		return fmt.Errorf("decrypting %s (wrong passphrase, or not an sssh backup): %w", in, err)
	}
	gz, err := gzip.NewReader(decrypted)
	if err != nil {
		return fmt.Errorf("decompressing %s: %w", in, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var archivedKnownHosts []byte

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", in, err)
		}
		rel, err := safeArchivePath(hdr.Name)
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeDir || rel == "" {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("reading %s from the archive: %w", hdr.Name, err)
		}
		mode := os.FileMode(hdr.Mode).Perm()

		switch {
		case rel == "MANIFEST.toml":
			// Informational only; nothing to restore from it today.
		case rel == "connections.toml":
			if err := writeRestored(p.StoreFile, data, mode); err != nil {
				return err
			}
		case rel == "known_hosts":
			archivedKnownHosts = data
		case strings.HasPrefix(rel, "ssh.d/"):
			dest := filepath.Join(p.Sandbox, filepath.FromSlash(strings.TrimPrefix(rel, "ssh.d/")))
			if err := writeRestored(dest, data, mode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected archive entry %q", hdr.Name)
		}
	}

	if archivedKnownHosts != nil {
		if err := mergeKnownHosts(p.KnownHosts, archivedKnownHosts); err != nil {
			return err
		}
	}
	return nil
}

// prepareSandbox refuses to restore over existing content unless forced, and
// when forced moves the old sandbox aside instead of deleting it. Restore is
// the only destructive command here; losing keys to it must be hard.
func prepareSandbox(p paths.Paths, force bool) error {
	entries, err := os.ReadDir(p.Sandbox)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", p.Sandbox, err)
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("%s already has content; re-run with --force to move it aside and restore", p.Sandbox)
	}
	backupDir := p.Sandbox + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
	if err := os.Rename(p.Sandbox, backupDir); err != nil {
		return fmt.Errorf("moving %s aside: %w", p.Sandbox, err)
	}
	return nil
}

func writeRestored(dest string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, data, mode); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	// WriteFile respects umask on creation, so set the mode explicitly: ssh
	// refuses a private key that is group- or world-readable.
	return os.Chmod(dest, mode)
}

// mergeKnownHosts appends archived entries the machine does not already have,
// preserving order and dropping exact duplicates.
func mergeKnownHosts(dest string, archived []byte) error {
	existing, err := os.ReadFile(dest)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", dest, err)
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !seen[line] {
			seen[line] = true
			out = append(out, line)
		}
	}
	for _, line := range strings.Split(string(archived), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !seen[line] {
			seen[line] = true
			out = append(out, line)
		}
	}
	body := strings.Join(out, "\n") + "\n"
	if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/backup/ -v`
Expected: all PASS.

- [ ] **Step 6: Write `cmd/sssh/cmd_backup.go`**

```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"time"

	"github.com/poppyszn/super-ssh-v2/internal/backup"
	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func cmdBackup(a *app, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	out := fs.String("o", "", "output file (default: sssh-backup-YYYYMMDD.sssh in the current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	target := *out
	if target == "" {
		target = "sssh-backup-" + time.Now().Format("20060102") + ".sssh"
	}

	passphrase, err := readPassphrase(a, "Passphrase for the backup: ")
	if err != nil {
		return err
	}
	confirmPass, err := readPassphrase(a, "Confirm passphrase: ")
	if err != nil {
		return err
	}
	if passphrase != confirmPass {
		return fmt.Errorf("the passphrases do not match")
	}

	if err := backup.Create(a.paths, target, passphrase); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Wrote %s (%d profiles, keys, and known_hosts).\n", target, len(a.store.Profiles))
	fmt.Fprintln(a.stdout, "Keep the passphrase somewhere safe: without it this file cannot be read.")
	return nil
}

func cmdRestore(a *app, args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	force := fs.Bool("force", false, "move an existing sssh.d aside and restore over it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: sssh restore [--force] <file>")
	}

	passphrase, err := readPassphrase(a, "Passphrase: ")
	if err != nil {
		return err
	}
	if err := backup.Restore(fs.Arg(0), a.paths, passphrase, *force); err != nil {
		return err
	}

	// Reload and regenerate so the restored store takes effect immediately.
	store, err := config.Load(a.paths.StoreFile)
	if err != nil {
		return fmt.Errorf("the archive restored but its store will not load: %w", err)
	}
	a.store = store
	if err := a.sync(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Restored %d profiles. Run `sssh ls` to see them.\n", len(a.store.Profiles))
	return nil
}

// readPassphrase reads a line from stdin.
//
// It does not disable terminal echo, which would mean importing
// golang.org/x/term -- a banned dependency, and one whose raw-mode handling is
// exactly the class of code v2 exists to avoid. The passphrase is visible while
// typed; that is the documented trade-off.
func readPassphrase(a *app, prompt string) (string, error) {
	fmt.Fprint(a.stdout, prompt)
	line, err := bufio.NewReader(a.stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("reading the passphrase: %w", err)
	}
	return trimNewline(line), nil
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
```

**Note for the implementer:** `readPassphrase` reads a fresh `bufio.Reader` each call, which discards anything buffered. In `cmdBackup` the two calls therefore need two lines of input, which is what a terminal gives. Do not "optimise" this into a shared reader without re-checking the tests.

- [ ] **Step 7: Add the dispatch cases, run everything, commit**

```go
	case "backup":
		return withApp(func(a *app) error { return cmdBackup(a, args[1:]) })
	case "restore":
		return withApp(func(a *app) error { return cmdRestore(a, args[1:]) })
```

Run: `go test ./... -v` — all PASS. Run: `make check`

```bash
git add internal/backup cmd/sssh
git commit -m "feat: encrypted single-file backup and restore"
```

---

### Task 16: `sssh doctor`

**Files:**
- Create: `internal/doctor/doctor.go`, `internal/doctor/doctor_test.go`, `cmd/sssh/cmd_doctor.go`
- Modify: `cmd/sssh/main.go` — add the `"doctor"` case

**Interfaces:**
- Consumes: `paths.Paths`, `config.Store`, `sshconfig.Generate`, `sshconfig.HasBlock`, `sshconfig.BlockIsFirst`.
- Produces:
  - `doctor.Status` (`StatusOK`, `StatusWarn`, `StatusFail`), `doctor.Result{Name, Status, Detail string}`
  - `doctor.Run(p paths.Paths, s config.Store) []Result`
  - `doctor.HasFailure(results []Result) bool`
  - `cmdDoctor(a *app) error` — exits non-zero when any check fails

**The eight checks, exactly:**

| Name | Fails when |
| --- | --- |
| `ssh binary` | `ssh` is not in PATH |
| `generated config` | `~/.ssh/sssh.d/config` is missing |
| `config drift` | its contents differ from a fresh `Generate` (warn, not fail — a hand edit is recoverable with `sssh sync`) |
| `include block` | `~/.ssh/config` has no managed block |
| `include position` | the block sits below a `Host` or `Match` line (warn — it is silently inert there) |
| `key permissions` | any file in `keys/` is group- or world-readable |
| `missing keys` | a profile names a key that does not exist |
| `orphaned keys` | `keys/` holds a file no profile references (warn) |
| `fzf` | not installed (warn — the fallback picker still works) |

- [ ] **Step 1: Write the failing test `internal/doctor/doctor_test.go`**

```go
package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
)

func healthy(t *testing.T) (paths.Paths, config.Store) {
	t.Helper()
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	store := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "web", Host: "h", User: "u", Port: 22, Auth: config.AuthKey, Key: "web"},
	}}
	if err := os.WriteFile(filepath.Join(p.KeysDir, "web"), []byte("-----BEGIN OPENSSH PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := sshconfig.Write(p, store); err != nil {
		t.Fatal(err)
	}
	if err := sshconfig.EnsureInclude(p); err != nil {
		t.Fatal(err)
	}
	return p, store
}

func find(t *testing.T, results []Result, name string) Result {
	t.Helper()
	for _, r := range results {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no check named %q in %v", name, results)
	return Result{}
}

func TestRunOnAHealthyInstallHasNoFailures(t *testing.T) {
	p, store := healthy(t)
	results := Run(p, store)
	if HasFailure(results) {
		for _, r := range results {
			if r.Status == StatusFail {
				t.Errorf("unexpected failure: %s — %s", r.Name, r.Detail)
			}
		}
	}
}

func TestRunDetectsAMissingGeneratedConfig(t *testing.T) {
	p, store := healthy(t)
	if err := os.Remove(p.GenConfig); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "generated config").Status; got != StatusFail {
		t.Errorf("status = %q, want %q", got, StatusFail)
	}
}

func TestRunDetectsDrift(t *testing.T) {
	p, store := healthy(t)
	if err := os.WriteFile(p.GenConfig, []byte("Host tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "config drift").Status; got != StatusWarn {
		t.Errorf("status = %q, want %q", got, StatusWarn)
	}
}

func TestRunDetectsAMissingIncludeBlock(t *testing.T) {
	p, store := healthy(t)
	if err := os.WriteFile(p.SSHConfig, []byte("Host other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "include block").Status; got != StatusFail {
		t.Errorf("status = %q, want %q", got, StatusFail)
	}
}

// An Include below a Host block is silently inert, which is the worst kind of
// broken: everything looks configured and nothing works.
func TestRunDetectsAMisplacedIncludeBlock(t *testing.T) {
	p, store := healthy(t)
	body := "Host first\n    HostName a\n" + sshconfig.Block
	if err := os.WriteFile(p.SSHConfig, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "include position").Status; got != StatusWarn {
		t.Errorf("status = %q, want %q", got, StatusWarn)
	}
}

func TestRunDetectsLooseKeyPermissions(t *testing.T) {
	p, store := healthy(t)
	if err := os.Chmod(filepath.Join(p.KeysDir, "web"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "key permissions").Status; got != StatusFail {
		t.Errorf("status = %q, want %q", got, StatusFail)
	}
}

func TestRunDetectsAMissingKey(t *testing.T) {
	p, store := healthy(t)
	if err := os.Remove(filepath.Join(p.KeysDir, "web")); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "missing keys").Status; got != StatusFail {
		t.Errorf("status = %q, want %q", got, StatusFail)
	}
}

func TestRunDetectsAnOrphanedKey(t *testing.T) {
	p, store := healthy(t)
	if err := os.WriteFile(filepath.Join(p.KeysDir, "nobody-uses-me"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "orphaned keys").Status; got != StatusWarn {
		t.Errorf("status = %q, want %q", got, StatusWarn)
	}
}

// A key linked in place lives outside the sandbox and must not be reported as
// missing just because it is not in keys/.
func TestRunIgnoresLinkedKeys(t *testing.T) {
	p, _ := healthy(t)
	external := filepath.Join(p.Root, "external_key")
	if err := os.WriteFile(external, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "linked", Host: "h", User: "u", Port: 22, Auth: config.AuthKey, Key: external},
	}}
	if err := sshconfig.Write(p, store); err != nil {
		t.Fatal(err)
	}
	if got := find(t, Run(p, store), "missing keys").Status; got == StatusFail {
		t.Error("a linked key that exists was reported as missing")
	}
}

func TestHasFailure(t *testing.T) {
	if HasFailure([]Result{{Status: StatusOK}, {Status: StatusWarn}}) {
		t.Error("HasFailure reported true with no failures")
	}
	if !HasFailure([]Result{{Status: StatusOK}, {Status: StatusFail}}) {
		t.Error("HasFailure reported false despite a failure")
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./internal/doctor/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write `internal/doctor/doctor.go`**

```go
// Package doctor checks that sssh's installation is intact. It exists because
// sssh writes into ~/.ssh, a directory the user cares about and may edit by
// hand.
package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
)

type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Result struct {
	Name   string
	Status Status
	Detail string
}

func HasFailure(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail {
			return true
		}
	}
	return false
}

// Run performs every check and returns the results in a fixed order.
func Run(p paths.Paths, s config.Store) []Result {
	generated, genErr := os.ReadFile(p.GenConfig)
	sshCfg, _ := os.ReadFile(p.SSHConfig)

	results := []Result{
		checkSSHBinary(),
		checkGeneratedConfig(p, genErr),
		checkDrift(s, generated, genErr),
		checkIncludeBlock(p, string(sshCfg)),
		checkIncludePosition(string(sshCfg)),
	}
	results = append(results, checkKeys(p, s)...)
	results = append(results, checkFzf())
	return results
}

func checkSSHBinary() Result {
	if _, err := exec.LookPath("ssh"); err != nil {
		return Result{"ssh binary", StatusFail, "ssh is not in PATH; sssh cannot connect without it"}
	}
	return Result{"ssh binary", StatusOK, ""}
}

func checkGeneratedConfig(p paths.Paths, err error) Result {
	if err != nil {
		return Result{"generated config", StatusFail, fmt.Sprintf("%s is missing; run `sssh sync`", p.GenConfig)}
	}
	return Result{"generated config", StatusOK, ""}
}

// checkDrift is a warning rather than a failure: a hand edit is recoverable
// with one command, and the user may have been mid-experiment.
func checkDrift(s config.Store, generated []byte, err error) Result {
	if err != nil {
		return Result{"config drift", StatusOK, "nothing to compare"}
	}
	if string(generated) != sshconfig.Generate(s) {
		return Result{"config drift", StatusWarn, "the generated config differs from connections.toml; run `sssh sync` to rebuild it"}
	}
	return Result{"config drift", StatusOK, ""}
}

func checkIncludeBlock(p paths.Paths, sshCfg string) Result {
	if !sshconfig.HasBlock(sshCfg) {
		return Result{"include block", StatusFail, fmt.Sprintf("%s does not include sssh's config; run `sssh sync`", p.SSHConfig)}
	}
	return Result{"include block", StatusOK, ""}
}

// checkIncludePosition catches the worst failure mode available here: an
// Include below a Host block is silently ignored, so everything looks
// configured and nothing works.
func checkIncludePosition(sshCfg string) Result {
	if !sshconfig.HasBlock(sshCfg) {
		return Result{"include position", StatusOK, "no block to place"}
	}
	if !sshconfig.BlockIsFirst(sshCfg) {
		return Result{"include position", StatusWarn,
			"the sssh block sits below a Host or Match line, where ssh will ignore it; move it to the top of the file"}
	}
	return Result{"include position", StatusOK, ""}
}

func checkKeys(p paths.Paths, s config.Store) []Result {
	referenced := map[string]bool{}
	var missing, loose []string

	for _, profile := range s.Profiles {
		if profile.Auth != config.AuthKey || profile.Key == "" {
			continue
		}
		keyPath := profile.Key
		if !strings.HasPrefix(keyPath, "/") && !strings.HasPrefix(keyPath, "~") {
			referenced[profile.Key] = true
			keyPath = filepath.Join(p.KeysDir, profile.Key)
		} else if strings.HasPrefix(keyPath, "~/") {
			keyPath = filepath.Join(p.Root, keyPath[2:])
		}
		if _, err := os.Stat(keyPath); err != nil {
			missing = append(missing, fmt.Sprintf("%s -> %s", profile.Name, keyPath))
		}
	}

	var orphaned []string
	entries, err := os.ReadDir(p.KeysDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err == nil && info.Mode().Perm()&0o077 != 0 {
				loose = append(loose, fmt.Sprintf("%s is %o", e.Name(), info.Mode().Perm()))
			}
			if !referenced[e.Name()] {
				orphaned = append(orphaned, e.Name())
			}
		}
	}

	results := make([]Result, 0, 3)
	if len(loose) > 0 {
		results = append(results, Result{"key permissions", StatusFail,
			"ssh refuses keys readable by others: " + strings.Join(loose, ", ")})
	} else {
		results = append(results, Result{"key permissions", StatusOK, ""})
	}
	if len(missing) > 0 {
		results = append(results, Result{"missing keys", StatusFail, strings.Join(missing, ", ")})
	} else {
		results = append(results, Result{"missing keys", StatusOK, ""})
	}
	if len(orphaned) > 0 {
		results = append(results, Result{"orphaned keys", StatusWarn,
			"no profile uses " + strings.Join(orphaned, ", ")})
	} else {
		results = append(results, Result{"orphaned keys", StatusOK, ""})
	}
	return results
}

func checkFzf() Result {
	if _, err := exec.LookPath("fzf"); err != nil {
		return Result{"fzf", StatusWarn, "not installed; `sssh` will use the numbered picker instead"}
	}
	return Result{"fzf", StatusOK, ""}
}
```

- [ ] **Step 4: Write `cmd/sssh/cmd_doctor.go`**

```go
package main

import (
	"fmt"

	"github.com/poppyszn/super-ssh-v2/internal/doctor"
)

func cmdDoctor(a *app) error {
	results := doctor.Run(a.paths, a.store)
	for _, r := range results {
		if r.Detail == "" {
			fmt.Fprintf(a.stdout, "%-6s %s\n", r.Status, r.Name)
			continue
		}
		fmt.Fprintf(a.stdout, "%-6s %s — %s\n", r.Status, r.Name, r.Detail)
	}
	if doctor.HasFailure(results) {
		return fmt.Errorf("some checks failed")
	}
	return nil
}
```

- [ ] **Step 5: Add the dispatch case, run everything, commit**

```go
	case "doctor":
		return withApp(cmdDoctor)
```

Run: `go test ./... -v` — all PASS. Run: `make check`

```bash
git add internal/doctor cmd/sssh
git commit -m "feat: add the doctor command"
```

---

### Task 17: `sssh cp`, `sssh edit`, and the README

**Files:**
- Create: `cmd/sssh/cmd_cp.go`, `cmd/sssh/cmd_edit.go`, `cmd/sssh/cmd_cp_test.go`, `cmd/sssh/cmd_edit_test.go`, `README.md`
- Modify: `cmd/sssh/main.go` — add the `"cp"` and `"edit"` cases

**Interfaces:**
- Consumes: `launch.SCPArgs`, `launch.Exec`, `config.Profile`, `(*app).save`.
- Produces: `prepareCopy(a *app, operands []string) ([]string, error)`, `cmdCp(a *app, args []string) error`, `cmdEdit(a *app, args []string) error`

**`sssh edit` has two forms:** with a name and flags it updates named fields; with no arguments it opens `$EDITOR` on `connections.toml`, then reloads, validates, and regenerates. The second form is why the store is a human-readable file.

- [ ] **Step 1: Write `cmd/sssh/cmd_cp_test.go`**

```go
package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestPrepareCopyBuildsScpArgv(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	argv, err := prepareCopy(a, []string{"prod-web:/var/log/app.log", "./"})
	if err != nil {
		t.Fatalf("prepareCopy: %v", err)
	}
	want := []string{"scp", "prod-web:/var/log/app.log", "./"}
	if !reflect.DeepEqual(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestPrepareCopyWorksInBothDirections(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if _, err := prepareCopy(a, []string{"./local.txt", "prod-web:/tmp/"}); err != nil {
		t.Fatalf("upload: %v", err)
	}
}

func TestPrepareCopyRejectsAnUnknownProfile(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	_, err := prepareCopy(a, []string{"ghost:/tmp/x", "./"})
	if err == nil {
		t.Fatal("expected an error for an unknown profile in a remote operand")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error %q should name the unknown profile", err)
	}
}

// A Windows-style or absolute local path must not be mistaken for host:path.
func TestPrepareCopyIgnoresLocalPathsWithColons(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if _, err := prepareCopy(a, []string{"./weird:name.txt", "prod-web:/tmp/"}); err != nil {
		t.Fatalf("a local path containing a colon was treated as remote: %v", err)
	}
}

func TestPrepareCopyNeedsTwoOperands(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if _, err := prepareCopy(a, []string{"prod-web:/tmp/x"}); err == nil {
		t.Fatal("expected an error with only one operand")
	}
}
```

- [ ] **Step 2: Write `cmd/sssh/cmd_cp.go`**

```go
package main

import (
	"fmt"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/launch"
)

// prepareCopy validates the operands and returns the scp argv.
//
// Profile names need no rewriting: they are ssh Host aliases, so scp resolves
// them through the same generated config. All this does is catch a typo before
// scp reports it as an unresolvable hostname.
//
// Split from cmdCp for the same reason as prepareConnect: Exec replaces the
// process image and cannot be called from a test.
func prepareCopy(a *app, operands []string) ([]string, error) {
	if len(operands) < 2 {
		return nil, fmt.Errorf("usage: sssh cp <source> <destination>, where either may be <profile>:/path")
	}
	for _, operand := range operands {
		name, ok := remoteProfileName(operand)
		if !ok {
			continue
		}
		if _, found := a.store.Find(name); !found {
			return nil, errNoProfile(name)
		}
	}
	if err := a.sync(); err != nil {
		return nil, err
	}
	return launch.SCPArgs(operands), nil
}

// remoteProfileName reports whether an operand is host:path form.
//
// A colon only marks a remote target when it comes before the first slash;
// "./weird:name.txt" is a local file, and scp reads it the same way.
func remoteProfileName(operand string) (string, bool) {
	colon := strings.Index(operand, ":")
	if colon <= 0 {
		return "", false
	}
	if slash := strings.Index(operand, "/"); slash >= 0 && slash < colon {
		return "", false
	}
	return operand[:colon], true
}

func cmdCp(a *app, args []string) error {
	argv, err := prepareCopy(a, args)
	if err != nil {
		return err
	}
	return launch.Exec(argv)
}
```

- [ ] **Step 3: Write `cmd/sssh/cmd_edit_test.go`**

```go
package main

import (
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func TestCmdEditUpdatesNamedFields(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdEdit(a, []string{"prod-web", "--host", "10.0.0.9", "--port", "2022"}); err != nil {
		t.Fatalf("cmdEdit: %v", err)
	}
	p, _ := a.store.Find("prod-web")
	if p.Host != "10.0.0.9" || p.Port != 2022 {
		t.Errorf("host/port = %s/%d, want 10.0.0.9/2022", p.Host, p.Port)
	}
	if p.User != "deploy" {
		t.Errorf("User = %q; unspecified fields must not change", p.User)
	}
}

func TestCmdEditPersistsAndRegenerates(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdEdit(a, []string{"prod-web", "--host", "10.0.0.9"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.Load(a.paths.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := reloaded.Find("prod-web")
	if p.Host != "10.0.0.9" {
		t.Error("the change was not written to connections.toml")
	}
	generated, _ := readFileString(a.paths.GenConfig)
	if !strings.Contains(generated, "HostName 10.0.0.9") {
		t.Errorf("the ssh config was not regenerated:\n%s", generated)
	}
}

func TestCmdEditRejectsAnUnknownProfile(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdEdit(a, []string{"ghost", "--host", "x"}); err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
}

func TestCmdEditRejectsAnInvalidChange(t *testing.T) {
	a, _ := newTestApp(t, testProfiles())
	if err := cmdEdit(a, []string{"prod-web", "--port", "99999"}); err == nil {
		t.Fatal("expected an error for an out-of-range port")
	}
	p, _ := a.store.Find("prod-web")
	if p.Port != 2222 {
		t.Error("the invalid change was applied to the in-memory store")
	}
}

func TestCmdEditRejectsAJumpCycle(t *testing.T) {
	profiles := testProfiles()
	profiles[1].Jump = "bastion"
	a, _ := newTestApp(t, profiles)
	if err := cmdEdit(a, []string{"bastion", "--jump", "prod-web"}); err == nil {
		t.Fatal("expected an error for a jump cycle")
	}
}
```

- [ ] **Step 4: Write `cmd/sssh/cmd_edit.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

// cmdEdit has two forms.
//
// With a name and flags it updates only the fields given. With no arguments it
// opens $EDITOR on connections.toml and then reloads, validates, and
// regenerates -- which is the payoff for the store being a file a human can
// read.
func cmdEdit(a *app, args []string) error {
	if len(args) == 0 {
		return editInEditor(a)
	}

	name := args[0]
	idx := -1
	for i, p := range a.store.Profiles {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errNoProfile(name)
	}
	p := a.store.Profiles[idx]

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	host := fs.String("host", p.Host, "hostname or IP address")
	user := fs.String("user", p.User, "login user")
	port := fs.Int("port", p.Port, "port")
	auth := fs.String("auth", string(p.Auth), "agent, key, or password")
	folder := fs.String("folder", p.Folder, "folder path")
	tags := fs.String("tags", strings.Join(p.Tags, ","), "comma-separated tags")
	jump := fs.String("jump", p.Jump, "name of the profile to connect through")
	forward := fs.Bool("forward-agent", p.ForwardAgent, "forward the ssh agent")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	updated := p
	updated.Host, updated.User, updated.Port = *host, *user, *port
	updated.Auth = config.AuthType(*auth)
	updated.Folder, updated.Jump, updated.ForwardAgent = *folder, *jump, *forward
	updated.Tags = splitTags(*tags)

	// Validate a copy of the whole store, so cross-profile rules such as jump
	// cycles are caught before anything changes.
	candidate := a.store
	candidate.Profiles = append([]config.Profile{}, a.store.Profiles...)
	candidate.Profiles[idx] = updated
	if err := candidate.Validate(); err != nil {
		return err
	}

	a.store = candidate
	if err := a.save(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Updated %s (%s@%s:%d).\n", updated.Name, updated.User, updated.Host, updated.Port)
	return nil
}

// editInEditor opens the store in $EDITOR and applies whatever comes back.
func editInEditor(a *app) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, a.paths.StoreFile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", editor, err)
	}

	store, err := config.Load(a.paths.StoreFile)
	if err != nil {
		return fmt.Errorf("your edits were saved but will not load, so the ssh config was left alone: %w", err)
	}
	a.store = store
	if err := a.sync(); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Applied your edits (%d profiles).\n", len(a.store.Profiles))
	return nil
}
```

- [ ] **Step 5: Add the dispatch cases in `cmd/sssh/main.go`**

```go
	case "cp":
		return withApp(func(a *app) error { return cmdCp(a, args[1:]) })
	case "edit":
		return withApp(func(a *app) error { return cmdEdit(a, args[1:]) })
```

- [ ] **Step 6: Write `README.md`**

````markdown
# sssh

A connection manager for SSH that generates OpenSSH config and then gets out of
the way.

Three jobs:

1. **One encrypted file** holds your profiles *and* your keys, so a new machine
   is a restore away.
2. **See and organise** your connections instead of reading `ssh_config`.
3. **Stop remembering flags.** Jump hosts, tunnels, `scp`, and agent forwarding
   are saved properties of a profile.

It is not a terminal. When you connect, sssh `exec`s the real `ssh` binary and
ceases to exist — so output speed, copy and paste, scrollback, and tmux all
behave exactly as they always have, because it *is* a normal ssh session.

Because sssh writes into `~/.ssh/config`, everything else on your machine sees
your hosts too: `scp`, `rsync`, `git`, Ansible, and VSCode Remote all work
without knowing sssh exists.

## Install

```bash
go install github.com/poppyszn/super-ssh-v2/cmd/sssh@latest
```

`fzf` is recommended but optional — without it, `sssh` prints a numbered list.

```bash
brew install fzf
```

## Getting started

```bash
sssh import          # adopt the hosts already in ~/.ssh/config
sssh add             # add a new one
sssh ls --tree       # see what you have
sssh                 # pick one and connect
sssh prod-web        # or connect by name
```

## Commands

| Command | Does |
| --- | --- |
| `sssh` | pick a connection and connect |
| `sssh <name>` | connect to a saved profile |
| `sssh <name> <cmd>` | run a command remotely |
| `sssh ls [--tree]` | list profiles |
| `sssh add` | add a profile, by flags or prompts |
| `sssh edit <name>` | change fields; with no name, open the store in `$EDITOR` |
| `sssh rm <name>` | remove a profile |
| `sssh cp <name>:/path ./` | copy files with `scp` |
| `sssh import` | adopt hosts from `~/.ssh/config` |
| `sssh backup -o FILE` | write an encrypted archive |
| `sssh restore FILE` | restore from an archive |
| `sssh doctor` | check the installation |
| `sssh sync` | regenerate the config (normally automatic) |

## Where things live

| Path | What |
| --- | --- |
| `~/.config/sssh/connections.toml` | the source of truth — plain TOML, safe to edit |
| `~/.ssh/sssh.d/config` | generated; do not edit |
| `~/.ssh/sssh.d/keys/` | adopted private keys, `0600` |
| `~/.ssh/config` | gets one `Include` line inside marked comments |

sssh writes nothing outside `~/.ssh/sssh.d/` and that marked block. The one
exception is `known_hosts`, which a restore *merges* rather than overwrites.

## Backups

```bash
sssh backup -o ~/Dropbox/sssh-backup.sssh
```

One file: profiles, keys, and `known_hosts`, compressed and encrypted with a
passphrase using [age](https://age-encryption.org). On a new machine:

```bash
sssh restore ~/Dropbox/sssh-backup.sssh
```

Permissions are preserved, so restored keys come back at `0600`. Restore refuses
to write over an existing `~/.ssh/sssh.d` unless you pass `--force`, and
`--force` moves the old one aside rather than deleting it.

## Passwords

sssh stores none. A profile records *which* authentication method it uses so the
right config is generated; `ssh` prompts you for passwords, key passphrases, and
2FA codes itself. There is no keychain and no master password.

## Contributing

Read [CLAUDE.md](CLAUDE.md) first — it explains the architectural rules and why
they exist. `make check` is the gate.
````

- [ ] **Step 7: Run everything, then commit**

Run: `go test ./... -v` — all PASS. Run: `make check`

```bash
git add cmd/sssh README.md
git commit -m "feat: add the cp and edit commands, and document the tool"
```

- [ ] **Step 8: Verify the whole tool end to end against a throwaway home**

This exercises the real binary without touching your real `~/.ssh`.

```bash
make build
export SSSH_HOME=$(mktemp -d)
./sssh add --name demo --host example.com --user me --port 2222
./sssh ls
./sssh doctor
cat "$SSSH_HOME/.ssh/sssh.d/config"
cat "$SSSH_HOME/.ssh/config"
printf 'hunter2\nhunter2\n' | ./sssh backup -o "$SSSH_HOME/snap.sssh"
./sssh rm --yes demo
./sssh ls
printf 'hunter2\n' | ./sssh restore --force "$SSSH_HOME/snap.sssh"
./sssh ls
unset SSSH_HOME
```

Expected: `ls` shows `demo`; `doctor` reports no failures; the generated config
contains `Host demo` with `Port 2222`; `~/.ssh/config` contains the managed
block at the top; after `rm` the list is empty; after `restore` `demo` is back.

- [ ] **Step 9: Final commit**

```bash
git add -A
git commit -m "chore: verify the end-to-end flow against a throwaway home"
```

---

## Done Criteria

The foundation is complete when all of these hold:

- `make check` passes on macOS and Linux — `gofmt`, `go vet`, `go test -race`, and all five guardrails.
- The end-to-end script in Task 17 Step 8 produces the expected output.
- `sssh doctor` reports no failures on a fresh install.
- Direct dependencies number exactly two.
- No `.go` file exceeds 400 lines; non-test Go stays under 2500.
- `syscall.Exec` appears once, in `internal/launch/launch.go`.
- `CLAUDE.md` and `README.md` exist and describe what was actually built.

## Deliberately Not In This Plan

These are features of v1 that v2 will not have. They are listed so a future
session does not treat their absence as an oversight:

embedded terminals, panes, tabs, splits, scrollback, text selection, mouse
support, any TUI framework, terminal emulation, an SSH protocol implementation,
a port-forwarding engine, a session manager, connection multiplexing, stored
passwords or passphrases, and RDP.

Each was in v1. Each is now a line of ssh config, an OpenSSH feature, or a
deliberate omission.
