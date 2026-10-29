# relay-ssh (sssh and Relay) — working agreement

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
| Non-test Go lines, whole repo | 3050 |
| `syscall.Exec` call sites | 1, in `internal/launch` |
| TUI / SSH-protocol / PTY imports | 0 |

Raising a budget requires changing the spec first and saying why in the commit
message. Silently bumping a number in `guardrails.sh` defeats the entire point
of this file.

**2500 -> 2900, 2026-09-19.** The original budget covered the 17-task
foundation plan. Real, requested functionality since then -- folder-qualified
profile addressing (`sssh Acme/dev-app-server`), auto-regenerating
`~/.ssh/sssh.d/config` after every mutating command instead of requiring a
separate `sssh sync`, a real picker for jump-host selection replacing free-text
entry, and (in progress at the time of this bump) hidden passphrase entry for
`backup`/`restore` and shell tab-completion for profile names -- pushed past
2500 on genuine feature work, not scope creep. 2900 leaves headroom for the
in-progress items without needing a second bump immediately after this one.

**2900 -> 3000, 2026-10-29.** The shared `internal/core` package (UI-free logic
the CLI and the Relay desktop app both call) gained a real `ssh_config` import:
adjacent `Host` blocks with no blank line between them were silently skipped,
one incomplete entry aborted the whole import, and entries without `HostName` or
`User` failed validation instead of getting ssh's own defaults. The fix parses
blocks properly, applies ssh's defaults, and reports what was skipped and why.
It is tested, fixes two real bugs, and the `desktop/` code is excluded from this
count. That left about 65 lines of headroom, and the same day the optional `folders`
list (empty folders for the desktop sidebar: model field, validation, and a small
helper so deleting or moving a connection never makes its folder vanish) used
it up: the CLI is now exactly at 3000. The folder operations themselves live in
`desktop/`, outside this count, so the next CLI change needs a real saving or a
reasoned bump.

**3000 -> 3010, 2026-10-29.** One optional profile field, `blocks` (`"on"`, `"off"`
or empty), the Relay desktop app's per-connection command-block mode, with its
validation: 6 lines in `internal/config`. It belongs in the profile (it travels
with backups and is repairable in an editor) and the CLI ignores it; the logic that
uses it lives in `desktop/`. See spec section 9n.

**3010 -> 3050, 2026-10-29.** A security audit found that profile values were written into the
generated ssh config unchecked, so a line break in a host, user, key or tunnel address (from a
crafted `connections.toml`, a restored backup or imported data) could add directives such as
`ProxyCommand` and run a command on connect. Validation now rejects line breaks and other control
characters in those fields (and spaces where they would split a value, and a host starting with
`-`), and restore refuses a backup whose connections would not load instead of installing them,
and caps how much of one archive entry it reads (`config.Parse` is `Load` without the file).
About 40 lines in `internal/config` and `internal/core`, each covered by a test that fails
without it. See spec section 9u.

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
