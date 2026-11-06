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
- Weaknesses in the **local terminal socket or RDP gateway** (token, Origin checks, loopback binding, SSRF)
  or the desktop app's webview boundary.
- Flaws in the RDP host certificate TOFU verification, ASN.1 RDCleanPath validation, or SSH jump tunnel handling.
- Backup and restore problems (weak encryption use, path traversal on restore, unsafe permissions).

## What Relay does today

- It never stores secrets unencrypted; SSH relies on agent/keychain/prompts and optional passwords live in an age-encrypted secret vault (`~/.config/relay-ssh/vault.age`, mode `0600`) unlocked with a master password.
- Backups use [age](https://age-encryption.org) with a scrypt passphrase and are written with `0600`
  permissions.
- Terminal and RDP gateway WebSockets listen on `127.0.0.1` only, requiring per-session random tokens and exact
  Origin matches to prevent local SSRF or unauthorized access.
- RDP host certificates use Trust-On-First-Use (TOFU) with SHA-256 fingerprints saved in `~/.config/relay-ssh/rdp_hosts.toml` (`0600`). Any certificate change produces a clear warning and stops the connection until confirmed.
- RDP connection handshakes are validated via ASN.1 RDCleanPath parsing before raw RDP byte streaming begins.
- The RDP gateway only dials the destination registered for that session (a different one in the client's request is refused), speaks TLS 1.2 to the host, bounds every handshake step to 20 seconds, caps a client message at 128 KiB, and logs why it refused or ended a connection (sizes and reasons only, never content or tokens). A session token is single use and lasts 5 minutes.
- The RDP client is WebAssembly, so the page's Content-Security-Policy allows `'wasm-unsafe-eval'` (to compile it), `data:` in `connect-src` (it fetches its own inlined module) and `worker-src 'self' blob:` (it runs a worker); nothing else is loosened, and the only network origins allowed are loopback.
- While a remote desktop is open the client's clipboard sync is backed by the app's native clipboard (text only), so it needs no browser permission; the browser's own clipboard is restored when the last RDP tab closes.
- The remote desktop client is built from a pinned IronRDP commit (`desktop/rdp-client/ironrdp.pin`) that the build refuses to deviate from; a weekly job proposes newer releases for human review and CI runs `cargo audit` on its lockfile.
- File and forward operations validate paths and ports against strict patterns, and reject control
  characters so a crafted name cannot add extra `sftp` commands.

## Supported versions

Relay is pre-1.0. Fixes land on the `main` branch.
