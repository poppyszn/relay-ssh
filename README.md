<p align="center">
  <img src="docs/assets/relay-icon.png" width="128" height="128" alt="Relay logo">
</p>

<h1 align="center">Relay</h1>

<p align="center"><b>Your SSH and remote desktop connections, organised, in one fast app. Built on OpenSSH.</b></p>

Relay is a desktop app for people who connect to many servers every day. It keeps your connections in a
tree of folders, opens them in a real terminal with tabs and split panes, and adds the tools you otherwise
cobble together around SSH: jump hosts and port forwards you set once, a file browser, encrypted backups,
key management, and Windows remote desktop. It ships with **`relay-ssh`**, a small command-line companion
that reads and writes the same data.

> **Status: early, macOS first.** Relay is developed and tested on macOS (Apple silicon). There are no
> signed releases or installers yet, so for now you build it from source (see [Install](#install-and-build)).
> Windows and Linux builds are planned. See the [Roadmap](#roadmap).

## The problem

If you manage more than a handful of servers, your connection details end up scattered:

- a long, hand-edited `~/.ssh/config` nobody can see the shape of;
- jump hosts, tunnels and agent forwarding that you re-type as flags or have to remember;
- keys and settings that are lost the day you change machines;
- one terminal window per server, a second tool for files, a third for Windows boxes.

Many tools that fix this replace SSH with their own client, or keep your passwords somewhere you cannot
inspect. Relay takes the opposite approach.

## What Relay does about it

- **Connections you can see.** Folders and subfolders, tags, search, quick open. Jump hosts, agent
  forwarding and port forwards are saved properties of a connection, not flags to remember. Two
  connections may share a name as long as they sit in different folders (`Prod/db` and `Stage/db`).
- **OpenSSH stays the SSH client.** Relay generates standard OpenSSH config and runs your system `ssh`,
  `sftp` and `ssh-keygen`. Agents, hardware keys, passphrases, jump hosts and 2FA behave exactly as in a
  plain terminal, and `ssh <name>` keeps working outside Relay too.
- **A new machine is a restore away.** One encrypted backup holds your connections and keys.
- **Your data is one readable file.** Connections live in `connections.toml`; you can edit or repair it
  with any editor, and the app and the CLI always agree.
- **Honest about secrets.** By default Relay stores none. If you opt in for a connection, its password
  goes in an [age](https://age-encryption.org)-encrypted vault on disk, unlocked by a master password
  (details in [Secrets and security](#secrets-and-security)).
- **A terminal that stays out of the way.** Bytes flow from a real PTY to [xterm.js](https://xtermjs.org)
  untouched, with flow control; Go does no terminal emulation. A 200 MB `cat` finishes in about 2.5 s with
  the UI still responsive.

## Features

**Desktop app (Relay)**

- **Connections:** tree with folders, tags and search. Double-click to open, drag to reorganise (every move
  asks first), duplicate a connection, import from `~/.ssh/config`. Drag a connection onto a terminal to
  split it; drag tabs and pane headers to rearrange.
- **Terminal:** tabs, split panes, scrollback search, fonts, four themes (three dark, one light, or follow
  the system), keep-alives so idle sessions do not drop, and click-to-move-the-cursor on the command line.
  With shell integration on, each command becomes a **block** with its directory, git branch, exit status
  and duration, and a ⋯ menu copies the command, its output, or both. Local zsh and bash are set up
  automatically; for a remote host you opt in.
- **Snippets:** saved commands you insert into, or run in, any terminal.
- **Files:** browse, upload and download over `sftp`, reusing the login of the connection's terminal tab.
- **Port forwards:** add, stop and save local, remote and SOCKS forwards on a live connection, with a
  diagram of how each one works.
- **Remote Desktop (RDP):** connect to Windows servers in their own tabs, through the same saved jump hosts,
  with a clipboard you can switch off. Experimental; see [Remote Desktop](#remote-desktop-rdp).
- **Tools:** health check with one-click fixes, passphrase-encrypted backup and restore, key management
  (for a key with a passphrase, Relay asks once and hands it to the macOS Keychain and ssh-agent so ssh
  stops asking; Relay never stores it), and the optional encrypted password vault.

**Command line (`relay-ssh`)**

```
relay-ssh                    pick a connection and connect
relay-ssh <name>             connect to a saved profile (name, Folder/name, or id)
relay-ssh ls [--tree]        list profiles
relay-ssh add | edit | rm    manage profiles
relay-ssh cp <name>:/p ./l   copy files with scp
relay-ssh import             adopt hosts from ~/.ssh/config
relay-ssh backup | restore   encrypted archive of connections and keys
relay-ssh doctor [--fix]     check the installation
relay-ssh sync               regenerate ~/.ssh/relay-ssh.d/config
relay-ssh completion <shell> bash or zsh completions
```

`relay-ssh` builds an argv and replaces itself with `ssh`; it is not running while you are connected. If a
bare name matches several connections, it lists the `Folder/name` forms that would work.

## Install and build

There are no prebuilt releases yet. You need [Go](https://go.dev) 1.27+ and, for the desktop app,
[Node.js](https://nodejs.org) 22 or newer and the [Wails 3](https://v3.wails.io) CLI.

**CLI**

```bash
go install github.com/poppyszn/relay-ssh/cmd/relay-ssh@latest
# or, from a checkout:
make build            # produces ./relay-ssh
```

**Desktop app (macOS)**

```bash
git clone https://github.com/poppyszn/relay-ssh && cd relay-ssh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
make rdp-client                       # optional: builds the Remote Desktop client (needs git, node, rustup, wasm-pack)
cd desktop
wails3 dev -config ./build/config.yml -port 9245     # run with live reload
wails3 package                                       # build desktop/bin/relay.app
```

To try it without touching your real SSH setup, point it at a throwaway home first:

```bash
RELAY_SSH_HOME=/tmp/relay-home wails3 dev -config ./build/config.yml -port 9245
```

`RELAY_SSH_HOME` redirects where Relay reads and writes its own files (`connections.toml` and
`~/.ssh/relay-ssh.d/`).

## Getting started

1. Open Relay and choose **New connection** (or **Tools → Import** to adopt the hosts already in your
   `~/.ssh/config`).
2. Give it a name, host, user and an auth method (agent, key or password). Pick a folder and, if the host
   is behind a bastion, a jump host.
3. Double-click it to connect. `ssh <name>` and `relay-ssh <name>` now work in any terminal as well.
4. Make a backup (**Tools → Backup & restore**) so a new machine is one restore away.

## Secrets and security

- **Nothing is stored by default.** Passwords, key passphrases and 2FA codes are left to OpenSSH, the
  ssh-agent and the macOS Keychain.
- **Optional password vault.** For password-only servers you can save a password per connection. It is
  kept only in an age-encrypted file (`~/.config/relay-ssh/vault.age`, mode `0600`), unlocked with a
  master password and locked again after idle time. It never appears in `connections.toml`, the generated
  SSH config, logs or backups.
- **Auto-fill is narrow.** A saved password is only sent to the host, user, port and jump host it was
  saved for, and only to a password prompt that matches one hop of that connection. Viewing a saved
  password requires the master password every time.
- **Key passphrases and 2FA codes are never stored by Relay.**
- Local sockets listen on loopback only, with per-session tokens and exact Origin checks. Relay writes
  only inside `~/.ssh/relay-ssh.d/` and one marked block of `~/.ssh/config`.

The full list is in [SECURITY.md](SECURITY.md), including how to report a vulnerability privately. Relay
has not had an independent security audit.

## How it works

```
Relay window (webview)          Go backend                      Your machine
-----------------------         ---------------------------     -------------------------
xterm.js (WebGL)        <--->   PTY session (raw bytes)  <--->  ssh <profile>
IronRDP (WebAssembly)   <--->   Loopback gateway (bytes) <--->  rdp / ssh -L tunnel
connection tree UI      <--->   internal/core (shared)   <--->  connections.toml
tools, files, forwards          ssh / sftp / ssh-keygen         ~/.ssh/relay-ssh.d/config
```

- `connections.toml` is the single source of truth. Saving regenerates `~/.ssh/relay-ssh.d/config`,
  pulled in by one marked `Include` block in `~/.ssh/config`.
- Every connection has a permanent id, so renaming or moving it never breaks a jump host or loses a saved
  password. A name stays the `ssh` alias while it is unique; when two connections share one, the alias
  becomes `Folder/name`.
- File transfers and live port forwards attach to the connection's terminal tab through OpenSSH
  connection sharing (`ControlMaster`), so they never need your credentials again.
- Backups are a gzip tarball encrypted with age (scrypt passphrase). Relay never stores the passphrase.

## Remote Desktop (RDP)

Relay can open Windows remote desktop sessions alongside SSH. **It is experimental.** It has been tested by
hand against a real Windows server with NLA (login, keyboard, mouse, resizing, text clipboard both ways, a
saved vault password and a jump host) and against an xrdp server in a container. File transfer is not built.

- **Pure relay:** RDP bytes go directly between the server and an in-browser WebAssembly client
  ([IronRDP](https://github.com/Devolutions/IronRDP)) over a loopback WebSocket. Go does no RDP decoding.
- **Building the client:** run `make rdp-client` before building the app (the commit is pinned in
  `desktop/rdp-client/ironrdp.pin`); without it an RDP tab says so instead of connecting.
- **Host certificates:** pinned on first use in `~/.config/relay-ssh/rdp_hosts.toml`; a changed
  fingerprint stops the connection until you confirm.
- **Jump hosts:** an RDP connection can tunnel through any saved SSH connection (`ssh -L`), torn down when
  the tab closes.
- **Credentials and clipboard:** a password can be saved in the vault or typed per session. Clipboard
  sharing can be switched off in Settings or per tab.
- Saved RDP connections use `protocol = "rdp"` in `connections.toml`; the `relay-ssh` CLI skips them.

## Example `connections.toml`

```toml
version = 1
folders = ["archive"]            # optional: folders that exist even when empty

[[profile]]
id = "@a1b2c3d4e5f6"             # permanent; added for you on save
name = "bastion"
host = "bastion.example.com"
user = "ops"
port = 22
auth = "agent"                   # agent | key | password

[[profile]]
id = "@0f9e8d7c6b5a"
name = "web-1"
host = "10.0.0.5"
user = "deploy"
port = 2222
auth = "key"
key = "id_ed25519"
folder = "prod/eu"               # names only need to be unique within a folder
tags = ["web", "critical"]
jump = "@a1b2c3d4e5f6"           # the id of another saved profile
forward_agent = true

  [[profile.tunnel]]
  type = "local"                 # local | remote | dynamic
  listen = "8080"
  target = "localhost:80"

[[profile]]
id = "@112233445566"
name = "win-dc"
protocol = "rdp"
host = "10.0.0.12"
user = "Administrator"
domain = "CORP"
port = 3389
jump = "@a1b2c3d4e5f6"
```

A `jump` written as a profile name (older files) is converted to the id the next time the file is saved.
Moved from the old `sssh` tool? Relay migrates `~/.config/sssh` and `~/.ssh/sssh.d` on first start.

## Roadmap

- [x] CLI: vault, generated OpenSSH config, backup, doctor, import
- [x] Desktop: connections, tabs, split panes, command blocks, search, themes
- [x] Desktop: tools (health, backup, import, keys), file browser, port forwards
- [ ] Windows (ConPTY, no connection sharing) and Linux builds
- [x] CI for the desktop app (macOS, on GitHub)
- [ ] Signed and notarized macOS releases, installers
- [ ] DevOps: environment tags with coloured tabs; AWS SSM (SSH over SSM and EC2 import); Kubernetes contexts
  pinned per tab with port forwards; a host-key manager for `known_hosts`
- [x] Encrypted vault for opt-in SSH and RDP password auto-fill
- [x] Embedded Remote Desktop (RDP) tabs with WebAssembly IronRDP, TOFU cert pinning and SSH jump tunnels (works against Windows with NLA, tested by hand; file transfer not built)
- [ ] RDP file transfer between your Mac and a remote desktop (IronRDP has a file-transfer provider; needs an upload/download UI)
- [ ] RDP: copy and paste of images and other non-text content
- [x] Per-block menu: copy command, output, or both
- [x] Light theme (Daylight) and System, which follows the OS
- [x] Block header: directory and git branch on every block
- [x] Snippets: saved commands to insert or run in any terminal

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first; it explains the project's
principles, the checks to run (`make check`, `make desktop-check`) and the code budget the CLI keeps.
Security issues: see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2026 Precious Okpor. Third-party software and the bundled fonts keep their own licenses;
see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
