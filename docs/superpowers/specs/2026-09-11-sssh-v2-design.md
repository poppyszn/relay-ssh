# super-ssh v2 — Design

**Date:** 2026-09-11
**Status:** Approved
**Module:** `github.com/poppyszn/super-ssh-v2`
**Binary:** `sssh`

## Why v2 Exists

v1 became a terminal multiplexer. It vendored forks of `bubbletea` and `vt10x`,
emulated a VT inside the app, and rendered every byte of remote output through a
cell grid and a full-screen diff. Copying text was slow, any command with long
output stalled the session, and the machine ran hot. None of that was a bug to
fix; it was the architecture.

The value was never the terminal. It was three things:

1. One encrypted file holding profiles *and* keys, so a new laptop is a restore
   away.
2. A way to see and organise connections instead of reading `ssh_config`.
3. Not having to remember the flags for jump hosts, tunnels, `scp`, and agent
   forwarding.

v2 delivers those three and nothing else. OpenSSH remains the SSH client.

## Invariants

These are not preferences. Violating one is a defect.

1. **sssh is never running while you are connected.** It builds an argv and
   `exec`s `ssh`, replacing its own process image. No child process, no pipes, no
   PTY, no bytes flowing through Go. This is the rule that makes every symptom
   of v1 structurally impossible.
2. **sssh owns `~/.ssh/sssh.d/` and one marked block in `~/.ssh/config`.**
   Nothing else. Any write outside that sandbox is a bug.
3. **The source of truth is a file a human can repair with `vim`.**

## Architecture

```
cmd/sssh/main.go       argument dispatch
internal/config/       TOML load/save, Profile type, validation
internal/sshconfig/    generate sssh.d/config, manage the Include block
internal/keys/         adopt key material, enforce permissions
internal/launch/       build argv, exec ssh/scp
internal/backup/       tar + gzip + age archive, and restore
internal/importer/     parse an existing ~/.ssh/config
internal/pick/         fzf detection, numbered fallback
internal/fsutil/       atomic file writes
internal/doctor/       health checks
```

Direct dependencies: a TOML parser and `filippo.io/age`. Everything else is
stdlib — `archive/tar`, `compress/gzip`, `os/exec`, `flag`. v1 had eleven direct
dependencies plus two vendored forks.

### Data flow

```
connections.toml  ──generate──>  ~/.ssh/sssh.d/config
                                        │
                                   Include'd by
                                        │
                                  ~/.ssh/config
                                        │
                                  read by ssh, scp,
                                  rsync, git, VSCode
```

sssh regenerates the config after every mutation. There is no `sync` step to
forget and no drift to reconcile.

## Data Model

`~/.config/sssh/connections.toml`:

```toml
version = 1

[[profile]]
name    = "prod-web"
host    = "10.0.0.5"
user    = "deploy"
port    = 22
auth    = "key"              # key | agent | password
key     = "prod-web"         # basename under sssh.d/keys/, or an absolute path
folder  = "work/prod"
tags    = ["web"]
jump    = "bastion"          # another profile's name
forward_agent = true

  [[profile.tunnel]]
  type   = "local"           # local | remote | dynamic
  listen = "5432"
  target = "db.internal:5432"
```

Folders are a `/`-delimited path string, not a `parent_id` graph. v1 needed cycle
detection in `UpdateFolder`; a path string cannot contain a cycle.

Profile names are unique and are the ssh `Host` alias. Validation rejects names
containing whitespace or ssh pattern characters (`*`, `?`, `!`), an empty host or
user, a port outside 1–65535, an `auth = "key"` with no key, a `jump` naming an
unknown profile, and a `jump` chain that cycles.

### Generated config

```
Host prod-web
    HostName 10.0.0.5
    User deploy
    Port 22
    IdentityFile ~/.ssh/sssh.d/keys/prod-web
    IdentitiesOnly yes
    ProxyJump bastion
    ForwardAgent yes
    LocalForward 5432 db.internal:5432
```

Jump hosts and port forwarding — `connection.go`, `forward.go`, and
`session_manager.go`, roughly 1,400 lines with tests in v1 — reduce to string
formatting. OpenSSH already implements them correctly.

### The Include block

`~/.ssh/config` receives exactly this, inserted at the **top** of the file:

```
# >>> sssh managed block >>>
Include ~/.ssh/sssh.d/config
# <<< sssh managed block <<<
```

Top placement is mandatory. ssh config is first-match-wins, so an `Include`
below any `Host *` block would be silently overridden. sssh inserts the block
if absent, leaves it alone if present, and never modifies a byte outside the
markers.

## Authentication

No secrets are stored. A profile records *which* method it uses so the right
config is generated; `ssh` does all prompting natively.

| Method     | Generated                                              |
| ---------- | ------------------------------------------------------ |
| `agent`    | nothing special — ssh finds the agent                   |
| `key`      | `IdentityFile <path>` + `IdentitiesOnly yes`            |
| `password` | `PreferredAuthentications password,keyboard-interactive`|

Key material may be supplied as a path or pasted. Either way `sssh` copies it
into `~/.ssh/sssh.d/keys/<name>` at `0600` so it is covered by backup. `--link`
references a key in place instead, at the cost of excluding it from backups.

Passphrase-protected keys, password logins, and 2FA prompts all work because
`ssh` asks the user directly. This removes v1's entire secrets subsystem: the
unlock screen, the session passphrase, the encrypted `secrets` table, and the
age box around individual fields.

## Backup and Restore

`sssh backup -o snapshot.sssh` produces a single file: a `tar` of
`connections.toml`, the whole of `~/.ssh/sssh.d/`, and a copy of
`~/.ssh/known_hosts`, gzipped, then encrypted with `age` using a passphrase.
File modes travel in the tar, so keys restore at `0600` and directories at
`0700`. `known_hosts` is the one item from outside the sandbox, included because
it is worthless to re-accept every host fingerprint on a new machine.

`sssh restore snapshot.sssh` reproduces that exactly and re-inserts the Include
block, creating `~/.ssh/config` if the machine has none. Restore writes only
inside the sandbox. It refuses to run against a non-empty sandbox unless given
`--force`, and `--force` moves the existing sandbox aside to
`sssh.d.bak-<timestamp>` rather than deleting it.

Because `known_hosts` lives outside the sandbox and is shared with everything
else on the machine, it is the single exception to "restore writes only inside
the sandbox" — and it is **merged** rather than overwritten: entries are
appended and deduplicated, so a restore never discards a host the machine
already trusts.

## Import

`sssh import` parses `~/.ssh/config`, lists the hosts it found and the keys they
reference, and adopts the ones selected. Adoption copies key material into the
sandbox and writes profiles into `connections.toml`. Hosts already managed by
sssh are skipped. `Host *` and other pattern entries are reported and skipped,
since they are defaults rather than connections.

This is both the migration path off the old config and the way the "one file
holds everything" promise becomes true over time: what you adopt is what gets
backed up.

## Command Surface

```
sssh                  fzf picker, then connect
sssh <name>           connect
sssh ls [--tree]      list profiles
sssh add              line-based prompts
sssh edit <name>      $EDITOR, or flags
sssh rm <name>
sssh cp <name>:/path ./local
sssh import           adopt hosts from ~/.ssh/config
sssh backup [-o f]    encrypted archive
sssh restore <f>
sssh doctor           verify Include block, permissions, orphaned keys
sssh sync             regenerate config (normally automatic)
```

`fzf` is a recommended dependency, not a required one. When it is absent, a bare
`sssh` prints a numbered list and reads a line. sssh does not implement fuzzy
matching, scrolling, or raw terminal mode. Building a picker is how v1 started.

## Error Handling

Errors go to stderr as `sssh: <what failed>: <cause>` and exit non-zero. There
is no error UI because there is no UI.

Writes to `connections.toml` and the generated config are atomic: write a temp
file in the same directory, then rename. A crash mid-write cannot leave a
corrupt store or a config that locks you out of your hosts.

When `exec` of `ssh` fails, sssh reports the argv it attempted so the failure is
diagnosable by hand.

## Testing

Every component above is a pure function over files, so tests are fast and
hermetic against a temp `$HOME`:

- `config` — round-trip, validation cases, malformed TOML
- `sshconfig` — golden files for generation; Include insertion into an empty
  file, a file with existing content, and a file already containing the block
- `keys` — adoption, permission enforcement, name collisions
- `launch` — argv assembly per auth type, jump, tunnels (assert the argv; never
  execute)
- `backup` — round-trip with modes preserved, wrong passphrase, sandbox refusal
- `importer` — real-world config fixtures, pattern-host skipping
- `doctor` — each failure it detects

No SSH connection tests, because there is no SSH protocol code. v1 ran a
`gliderlabs/ssh` server to test an 883-line `connection_test.go`.

## Scale

Roughly 1,500 lines of Go plus tests. v1's non-vendored code was about 11,000.

## Explicitly Out of Scope

Permanently, not "not yet":

- Embedded terminals, panes, tabs, splits, scrollback, text selection
- Any TUI framework or raw terminal mode
- Terminal emulation of any kind
- Reimplementing the SSH protocol
- A port-forwarding engine, a session manager, or connection multiplexing
- Storing passwords or passphrases
- RDP and other protocols

Each of these was in v1. Each is now a line of ssh config, an OpenSSH feature,
or a deliberate omission.
