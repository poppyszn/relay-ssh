# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

Ships as a native desktop window (Wails shell, Svelte UI in a webview), macOS first, with the native window frame and app menu. Windows and Linux builds are planned. No iOS or Android.

## Users

Developers and sysadmins who connect to many servers every day, through jump hosts and tunnels, and who want OpenSSH to keep behaving exactly as it does in a plain terminal. Power users set the bar. First-time screens should still make sense to someone who does not know ssh config syntax.

## Product Purpose

Relay organises SSH connections and puts a real terminal around them: a tree of connections, tabs and split panes, per-command blocks, a file browser, port forwards, backup and key management, all in one window. It exists so that connection details stop living in a buried `~/.ssh/config` and a new machine is a restore away. Success is opening the right server in a click or a keystroke, and never having to remember a flag.

## Positioning

OpenSSH stays the SSH client. Relay generates standard OpenSSH config and runs the system `ssh`, `sftp` and `ssh-keygen`; it never reimplements the protocol, so agents, hardware keys, passphrases, passwords and 2FA all work as in a terminal. The source of truth is one human-readable file (`connections.toml`) that the app and the command-line companion both read and write. Terminal bytes are not interpreted in Go, which is why it stays fast on huge output.

## Operating Context

Used all day, often with many tabs and panes open, mostly keyboard-driven (quick open, shortcuts, drag and drop where it helps). Dark themes are the default; one light theme and a follow-the-system option exist. Servers are reached directly, through saved jump hosts, or over sessions that already have a login open.

## Capabilities and Constraints

- Connections with folders, tags, jump hosts, agent forwarding and saved port forwards; import from `~/.ssh/config`.
- Terminal with tabs, split panes, scrollback search, themes, and optional command blocks (directory, git branch, exit status, duration; a setup line is typed into remote shells only with consent).
- Tools: health check, passphrase-encrypted backup and restore, key adoption (passphrases go to the macOS Keychain and ssh-agent, never stored by Relay), snippets, file browser over sftp, live port forwards.
- Optional encrypted vault for SSH passwords, opt-in per connection, filled only for the host a password was saved for.
- Relay stores no secrets by default. It writes only inside `~/.ssh/sssh.d/` and one marked block of `~/.ssh/config` (naming under review).
- Terminology: "connection" (not "host" or "server profile"), "jump host", "port forward", "command blocks", "vault".
- Undecided: Windows and Linux design details; signed and notarized macOS releases.

## Brand Commitments

The product is named Relay (repository `relay-ssh`). The app icon and logo are in `docs/assets/`. Open source, MIT, private until the owner decides otherwise.

## Evidence on Hand

Working app and its test and smoke harness; README and CLAUDE.md describe the principles. There are no user testimonials, usage data or published benchmarks beyond the large-output throughput measurements in the README. Do not invent any.

## Product Principles

1. OpenSSH does the work; Relay organises and gets out of the way. Never put the app in the data path.
2. Honest about secrets: nothing is stored without an explicit opt-in, and every stored thing says where it lives.
3. The data stays human-repairable: one readable file, standard OpenSSH config.
4. Speed and keyboard first. The common path (find a connection, open it) should take seconds and no mouse.
5. Plain language over ssh jargon: say what happened and what to do next, especially on errors and first run.

## Accessibility & Inclusion

Keyboard operation of every control with a visible focus ring; text contrast of at least 4.5:1 and field borders of at least 3:1 in all four themes; motion respects reduced-motion.
