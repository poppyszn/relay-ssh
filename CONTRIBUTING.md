# Contributing to Relay

Thanks for helping. Relay is small on purpose, so a few principles decide most questions. Please read
them before opening a pull request, and open an issue first for anything bigger than a fix.

## Principles

1. **OpenSSH stays the SSH client.** Do not add SSH protocol code. Generate config and run `ssh`,
   `sftp` or `ssh-keygen`.
2. **Never store a secret.** No passwords, passphrases, tokens or 2FA codes on disk or in memory beyond
   the moment of use. They belong to `ssh`.
3. **One human-editable source of truth:** `connections.toml`. The CLI and the app share it through
   `internal/`, so a change made in one must appear in the other.
4. **Stay inside the sandbox.** Relay writes only to `~/.ssh/relay-ssh.d/` and one marked block in
   `~/.ssh/config` (plus `known_hosts` merges). Every other write outside Relay's own config directory
   is a bug.
5. **The CLI never sits in the data path.** `relay-ssh` builds an argv and `exec`s `ssh`: no child process, no
   PTY, no terminal emulation. `CLAUDE.md` explains the history behind this (an earlier version failed
   by emulating a terminal in Go).
6. **The desktop terminal is bytes in, bytes out.** Go moves raw bytes between a PTY and xterm.js with
   flow control and does no terminal parsing. Anything that interprets terminal output (command
   blocks, search, decorations) lives in the frontend.
7. **The terminal socket stays closed to the world:** loopback only, a per-launch token, an exact Origin
   allow-list.

## Setup

You need Go 1.27+, Node.js 22 or newer, and the Wails 3 CLI (`go install
github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`).

```bash
git clone https://github.com/poppyszn/relay-ssh && cd relay-ssh
make check                      # CLI: format, vet, race tests, guardrails
make desktop-check              # app: guardrails, Go tests, typecheck, frontend tests, build
RELAY_SSH_HOME=/tmp/relay-home wails3 dev -config desktop/build/config.yml -port 9245   # run the app
```

Run `make hooks` once to install the pre-push hook: on macOS it runs `make desktop-check` before every
push (skip once with `git push --no-verify`). On other systems it does nothing.

`RELAY_SSH_HOME` points Relay at a throwaway home so you never touch your real SSH setup. Run `desktop`
commands from the `desktop/` directory.

## Before you open a pull request

- **Write the test first** and watch it fail. Tests must be hermetic: use `t.TempDir()`, never the real
  `$HOME`, and no network. Some desktop tests start a throwaway unprivileged `sshd` on a loopback
  port; they skip themselves when `sshd` is unavailable.
- **Run both gates** and read the output: `make check` and `make desktop-check`. Do not report work
  as done on the strength of having written it.
- **Respect the CLI budgets** enforced by `scripts/guardrails.sh` (dependency count, file length,
  total non-test lines, and where `syscall.Exec` may appear). The numbers are documented in
  `CLAUDE.md`. Raising one means updating `CLAUDE.md` first and explaining why in the commit; the
  desktop app has its own guardrails in `desktop/scripts/guardrails.sh`.
- **Design before code** for a new command or any change to the data model: explain the reasoning in
  the pull request description before building it.
- **Use conventional commits** (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`), one logical
  change per commit.
- **UI changes:** match the existing flat design (square structure, small radii on controls only, round
  status dots) and say how you checked it. Screenshots help.

## Reporting bugs and ideas

Use the issue templates. For a bug, include your OS, how you ran Relay (`wails3 dev` or a build), and
the steps to reproduce. For security problems, do **not** open a public issue: see
[SECURITY.md](SECURITY.md).

## License

By contributing you agree that your contributions are licensed under the [MIT License](LICENSE).
