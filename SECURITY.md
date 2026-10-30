# Security policy

Relay manages SSH connections, so security reports matter. Thank you for taking the time.

## Reporting a vulnerability

**Please do not open a public issue.** Report it privately through GitHub:
**Security → Report a vulnerability** on this repository (private vulnerability reporting).

Include what you found, how to reproduce it, and the impact you expect. You will get an acknowledgement
and we will keep you updated on the fix. Please allow reasonable time to ship a fix before disclosing
publicly.

## What is in scope

- Anything that causes Relay to **store, log or leak a secret** (passwords, passphrases, keys, 2FA codes).
- Writes outside Relay's sandbox (`~/.ssh/relay-ssh.d/`, its marked block in `~/.ssh/config`, its own
  config directory).
- Command or argument **injection** into `ssh`, `sftp` or `ssh-keygen` through connection names, hosts,
  file names, folders, forward specs or other user-controlled input.
- Weaknesses in the **local terminal socket** (token, Origin checks, loopback binding) or the desktop
  app's webview boundary.
- Backup and restore problems (weak encryption use, path traversal on restore, unsafe permissions).

## What Relay does today

- It never stores secrets; `ssh` itself prompts for them.
- Backups use [age](https://age-encryption.org) with a scrypt passphrase and are written with `0600`
  permissions.
- The terminal WebSocket listens on `127.0.0.1` only, requires a random per-launch token and an exact
  Origin match.
- File and forward operations validate paths and ports against strict patterns, and reject control
  characters so a crafted name cannot add extra `sftp` commands.

## Supported versions

Relay is pre-1.0. Fixes land on the `main` branch.
