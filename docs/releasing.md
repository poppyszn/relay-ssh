# Releasing Relay

Pushing to the `release` branch builds and publishes a release. Installed copies of Relay find it and offer
to update.

## Cut a release

1. On `main`, change `VERSION` to the new number (`MAJOR.MINOR.PATCH`, no leading `v`) and merge it.
2. Fast-forward `release` to `main` and push:

   ```bash
   git push origin main:release
   ```

3. The `release` workflow checks that `v<VERSION>` does not exist yet, builds the app for macOS (Apple
   silicon, Intel) and Linux (x86-64), signs an update manifest, and publishes a GitHub release named
   `v<VERSION>` with the downloads, `SHA256SUMS` and `update.json`. Release notes are the commit subjects
   since the previous release.

A push with an unchanged `VERSION` fails at the first step on purpose: every release needs its own number.

## How an update reaches an installed app

- The app asks `https://github.com/poppyszn/relay-ssh/releases/latest/download/update.json`, which lists each
  platform's download with its SHA-256 digest and an ed25519 signature of that digest.
- It only installs a download whose signature verifies against the **public key built into the app**
  (`desktop/update_pubkey.txt`). A different key, or a download that differs from what was signed, is
  rejected before anything on disk changes.
- The new version is staged first; the running app is replaced only when the user chooses Restart.
- Local builds (`wails3 dev`, `wails3 package` from a checkout) report a version that is not
  `MAJOR.MINOR.PATCH`, so they never check for or install updates.
- `RELAY_UPDATE_URL` points the app at another manifest, to rehearse a release against a local server. The
  signature is still checked.

The feed is only reachable by people who can open this repository, so while it is private, only you
can download releases and the in-app updater finds nothing. It works for everyone once the repository is
public. Nothing else needs to change.

## The signing key

The private key signs every release and lives only in the repository secret `RELAY_UPDATE_SIGNING_KEY`.
Whoever holds it can ship code to every installed copy, so treat it like a password.

- Create the key pair (once): `cd desktop && go run ./cmd/mkupdate -genkey -out update.key` writes the
  private key and prints the public key. Put the public key in `desktop/update_pubkey.txt`, and the private
  key in the secret: `gh secret set RELAY_UPDATE_SIGNING_KEY < update.key`. Keep a copy somewhere safe, then
  delete the file.
- **Losing the key** means no release can be signed for the apps already installed; they would have to be
  reinstalled by hand. **A leaked key** needs the same: generate a new pair, ship an app with the new public
  key, and reinstall by hand, because the old key can no longer be trusted to deliver the update.

## What is not here yet

- **macOS notarization.** Builds are signed ad hoc, so a first download from a browser needs a right-click,
  Open. Updates installed by the app itself do not hit this. Notarizing needs an Apple Developer account.
- **Windows.** The app starts terminals with a Unix pseudo-terminal, which Windows does not have, and some
  process handling is Unix-only. A Windows build needs ConPTY support first.
- **Linux packages.** The Linux download is a tarball holding one binary. An AppImage or deb is a later step.
