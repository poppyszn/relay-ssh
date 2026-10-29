<p align="center">
  <img src="docs/assets/relay-icon.png" width="128" height="128" alt="Relay logo">
</p>

<h1 align="center">Relay</h1>

<p align="center"><b>A fast, modern SSH connection manager and terminal, built on OpenSSH.</b></p>

Relay is a desktop app for people who live in SSH: a tree of your connections, a real terminal with
per-command blocks, tabs and split panes, and the everyday tools around SSH (encrypted backup, key
management, file browsing, port forwards) in one place. It ships with **`sssh`**, a small command-line
companion that reads and writes the same data.

> **Status: early and macOS first.** Relay is being built in the open. The desktop app is developed
> and tested on macOS (Apple silicon). Windows and Linux builds, signed releases and installers are not
> done yet. See [Roadmap](#roadmap).

## Why another SSH client?

- **OpenSSH stays the SSH client.** Relay never reimplements the protocol. It generates standard
  OpenSSH config and runs your system `ssh`, `sftp` and `ssh-keygen`, so jump hosts, agents, hardware
  keys, passwords, passphrases and 2FA all keep working exactly as they do in a plain terminal.
- **No secrets are stored.** Passwords, passphrases and 2FA codes are only ever typed into `ssh` itself.
- **Your data is one human-readable file.** Connections live in `connections.toml`, which you can edit
  with any editor. Relay regenerates an OpenSSH include file from it, and the CLI and the app always
  agree.
- **The terminal is not in the way.** Terminal bytes flow from a real PTY to [xterm.js](https://xtermjs.org)
  untouched, with flow control. Go does no terminal emulation, which is what makes it fast with huge
  output (a 200 MB `cat` finishes in about 2.5 s with the UI staying responsive).

## Features

**Desktop app (Relay)**

- Connection tree with folders and subfolders, tags, search, jump hosts, agent forwarding and saved port
  forwards. Double-click a connection to open it. Drag connections and folders to reorganise (every move asks
  for confirmation first), drag a connection onto a terminal to split it, drag tabs to reorder them and
  panes by their header to rearrange them.
- Terminal with tabs, split panes, scrollback search, fonts and four themes (three dark, one light, or follow the system). With shell integration on,
  each command becomes a block with its directory, git branch, exit status and duration, started from the bottom of the pane, and a ⋯ menu
  on each block copies the command, its output, or both.
  Local zsh and bash are set up automatically; for a remote host you opt in explicitly.
- **Tools:** health check with one-click fixes, passphrase-encrypted backup and restore, import from
  `~/.ssh/config`, and key management.
- **Files:** browse, upload and download over `sftp`, reusing the login of the connection's terminal tab.
- **Port forwards:** add, stop and save local, remote and SOCKS forwards on a live connection, with a
  diagram of how each one works.

**Command line (`sssh`)**

```
sssh                    pick a connection and connect
sssh <name>             connect to a saved profile
sssh ls [--tree]        list profiles
sssh add | edit | rm    manage profiles
sssh cp <name>:/p ./l   copy files with scp
sssh import             adopt hosts from ~/.ssh/config
sssh backup | restore   encrypted archive of connections and keys
sssh doctor [--fix]     check the installation
sssh sync               regenerate ~/.ssh/sssh.d/config
sssh completion <shell> bash or zsh completions
```

`sssh` builds an argv and replaces itself with `ssh`; it is not running while you are connected.

## Install and build

You need [Go](https://go.dev) 1.27+ and, for the desktop app, [Node.js](https://nodejs.org) 22 or newer and the
[Wails 3](https://v3.wails.io) CLI.

**CLI**

```bash
go install github.com/poppyszn/relay-ssh/cmd/sssh@latest
# or, from a checkout:
make build            # produces ./sssh
```

**Desktop app (macOS)**

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
cd desktop
wails3 dev -config ./build/config.yml -port 9245     # run with live reload
```

To try it without touching your real SSH setup, point it at a throwaway home first:

```bash
SSSH_HOME=/tmp/relay-home wails3 dev -config ./build/config.yml -port 9245
```

`SSSH_HOME` redirects where Relay reads and writes its own files (`connections.toml`,
`~/.ssh/sssh.d/`). A production build is `npm run build` in `desktop/frontend`, then `go build` in
`desktop/`.

## How it works

```
Relay window (webview)          Go backend                      Your machine
-----------------------         ---------------------------     -------------------------
xterm.js (WebGL)        <--->   PTY session (raw bytes)  <--->  ssh <profile>
connection tree UI      <--->   internal/core (shared)   <--->  connections.toml
tools, files, forwards          ssh / sftp / ssh-keygen         ~/.ssh/sssh.d/config
```

- `connections.toml` is the single source of truth. Saving regenerates
  `~/.ssh/sssh.d/config`, pulled in by one marked `Include` block in `~/.ssh/config`. Relay only writes
  inside `~/.ssh/sssh.d/` and that block.
- Terminal bytes travel over a loopback WebSocket guarded by a per-launch random token and an exact
  Origin allow-list.
- File transfers and live port forwards attach to the connection's terminal tab through OpenSSH
  connection sharing (`ControlMaster`), so they never need your credentials.
- Backups are a gzip tarball encrypted with [age](https://age-encryption.org) (scrypt passphrase). Relay
  never stores the passphrase.

The design notes behind these decisions are in [`docs/superpowers/specs/`](docs/superpowers/specs/).

## Example `connections.toml`

```toml
version = 1
folders = ["archive"]            # optional: folders that exist even when empty

[[profile]]
name = "web-1"
host = "10.0.0.5"
user = "deploy"
port = 2222
auth = "key"                     # agent | key | password
key = "id_ed25519"
folder = "prod/eu"
tags = ["web", "critical"]
jump = "bastion"                 # another saved profile
forward_agent = true

  [[profile.tunnel]]
  type = "local"                 # local | remote | dynamic
  listen = "8080"
  target = "localhost:80"
```

## Roadmap

- [x] CLI: vault, generated OpenSSH config, backup, doctor, import
- [x] Desktop: connections, tabs, split panes, command blocks, search, themes
- [x] Desktop: tools (health, backup, import, keys), file browser, port forwards
- [ ] Windows (ConPTY, no connection sharing) and Linux builds
- [x] CI for the desktop app (macOS, on GitHub)
- [ ] Signed and notarized macOS releases, installers
- [x] Per-block menu: copy command, output, or both
- [x] Light theme (Daylight) and System, which follows the OS
- [x] Block header: directory and git branch on every block
- [ ] Snippets

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first; it explains the project's
principles, the checks to run (`make check`, `make desktop-check`) and the code budget the CLI keeps.
Security issues: see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2026 Precious Okpor. Third-party software and the bundled fonts keep their own licenses;
see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
