# Task: an encrypted vault for SSH passwords, with safe auto-fill

Hand-off document for another coding agent. It is meant to be read cold: it says what the project
is, what is already decided, what to build, how to build it safely, and how it will be reviewed.
Work through it top to bottom. Do not skip the "Rules of the road" and "Security requirements"
sections: this feature stores secrets, and the project was built on the rule that it does not.

---

## 1. Context

**Relay** is a desktop SSH connection manager and terminal (Wails v3 + Svelte 5 + xterm.js, macOS
first). It ships with **`sssh`**, a small CLI that reads and writes the same data.
Repo: `https://github.com/poppyszn/relay-ssh` (private). Read these first, in this order:

1. `CLAUDE.md`: the working agreement, the invariants and the budgets. Binding.
2. `docs/superpowers/specs/2026-10-28-desktop-app-design.md`: the design record. Sections 9a to 9q
   are the decisions and progress so far; read 9m to 9q (blocks over ssh, keys and passphrases).
3. `README.md`: what the product claims, including "No secrets are stored".

Key facts about the architecture:

- **OpenSSH stays the client.** Relay builds an `ssh` argv and runs it in a PTY. Go never interprets
  terminal bytes; the frontend (xterm.js) does all parsing. Bytes flow PTY <-> loopback WebSocket
  untouched, with flow control.
- `connections.toml` (in `~/.config/sssh/`) is the source of truth for connections. Relay regenerates
  an OpenSSH include file from it. **No secret may ever be written to `connections.toml` or to the
  generated ssh config.**
- The shared packages live in `internal/` (the CLI uses them too). The desktop app lives in
  `desktop/` (Go module `github.com/poppyszn/relay-ssh/desktop`, frontend in `desktop/frontend`).
- The desktop code is where all of this feature goes. **Do not touch `internal/`**: the CLI has a hard
  line budget (`scripts/guardrails.sh`, currently 3006 of 3010 non-test lines) and does not need this.

### What exists today for passwords

Nothing. Choosing "Password" as a connection's auth method only adds `PreferredAuthentications
password,keyboard-interactive` to the generated config; ssh then asks in the terminal every time. The
form says "ssh asks for the password in the terminal. It is never stored."

### What already exists that you will build on

| Need | Where | Notes |
| --- | --- | --- |
| Encrypt with a passphrase | `filippo.io/age` scrypt, used in `internal/core/backup.go` | Already a dependency of the root module. The desktop module can import it (add to `desktop/go.mod`; `go mod tidy`). |
| Atomic file writes | `internal/fsutil.WriteFile(path, data, mode)` | Writes a temp file in the same directory and renames. Use mode `0600`. |
| Config dir | `paths.Default()` then `filepath.Dir(p.StoreFile)` (`~/.config/sssh`); `SSSH_HOME` redirects it for tests | See `desktop/snippets.go` for a desktop-only file store with tests. |
| "Login finished" signal | `loginMarker` (OSC 7771) and the `mux` text frame, `Session.authed` in `desktop/frontend/src/terminal.ts` | A password may only ever be typed **before** this fires. Study `desktop/terminal.go`, `masterAlive`, and `workspace.svelte.ts` `autoBlocks` for the same pattern applied to command blocks. |
| Control messages over the socket | `control` struct and the read loop in `desktop/session.go` | Currently `ack` and `resize`. |
| Passphrase handling for **keys** | `desktop/keypass.go` | Keys already work through ssh-agent and the macOS Keychain. **Out of scope here.** Do not change it. |
| Pattern for a service + tests | `desktop/snippets.go`, `desktop/snippets_test.go` | Copy this shape. |

---

## 2. Goal

Let a user opt in, per connection, to having Relay **remember an SSH account password** and **type it
for them** at ssh's own password prompt, so they are not asked every time. The password is stored in
an **encrypted vault file protected by a master password**, owned by Relay (not the OS Keychain).

### In scope

1. A vault: create, unlock, lock, change master password, reset (delete), set/forget/has a secret per
   connection, rename/forget a secret when a connection is renamed/deleted.
2. Auto-fill of the account password for connections whose auth method is `password`, with the
   safeguards in section 6.
3. UI: a password field in the connection form, vault create/unlock/lock UI, status indication.
4. Docs: update the invariants honestly (section 9).

### Out of scope (do not build)

- Key passphrases (already solved via ssh-agent / Keychain, see `keypass.go`).
- The macOS Keychain as a password store (but see section 4: design the storage behind an interface so
  it could be added later).
- Including the vault in backups (backups are built in `internal/core`, under the CLI budget). The
  restore path must simply **not overwrite or delete** `vault.age`. Note it as a known gap in the spec.
- Windows/Linux specifics, cloud sync, sharing a vault between people, biometrics.
- 2FA codes, host-key confirmations, `sudo` prompts: never auto-answered.

---

## 3. Rules of the road (from `CLAUDE.md`, restated)

- **TDD.** Write the failing test, watch it fail for the right reason, write the smallest code that
  passes, watch it pass. A test you never saw fail proves nothing.
- **Spec before code** for a data-model change: add a section **9r** to the design doc first (decisions,
  file format, threat model, the known gaps). Keep it factual, in the style of 9m to 9q.
- **Hermetic tests**: `t.TempDir()`, `t.Setenv("SSSH_HOME", ...)`, no real `$HOME`, no network, **never
  touch the real Keychain or the user's real ssh-agent**. No real passwords in tests or fixtures.
- **Atomic writes** for anything that could lock the user out of their data.
- **Conventional commits**, one logical step per commit. Do not add attribution lines naming a tool
  other than the one you are.
- **Gates**, run before you claim anything: `make check` (Go fmt/vet/test -race + CLI guardrails) and
  `make desktop-check` (desktop guardrails, svelte typecheck, vitest, build, Go vet + tests). Both must
  exit 0. Read the output; do not trust a pipe's exit status.
- **Branch**: work on `feature/vault`. Push the branch. **Do not merge to `main` and do not force-push.**
  The reviewer (Claude) will review the branch, run the real-window smoke harness, and merge.
- Never put a secret in a log line, an error message, a URL, `localStorage`, a query string, a test
  name, or a screenshot.

---

## 4. Design (decided; deviate only with a written reason in the spec)

### 4.1 Storage interface

```go
// desktop/secrets.go
type SecretStore interface {
    Status() VaultStatus                 // exists, unlocked
    Get(connection string) (string, bool, error)   // Go-internal use only, never bound to the frontend
    Set(connection, secret string) error
    Forget(connection string) error
    Rename(oldName, newName string) error
    Has(connection string) bool
}
```

`Vault` is the one implementation. The fill logic depends on `SecretStore`, not on `Vault`, so a
Keychain-backed store could be dropped in later. `Get` must **not** be exposed through Wails bindings
(only methods on registered services are bound; keep `Get` on an unexported type or lowercase method).

### 4.2 The vault file

- Path: `<config dir>/vault.age` (next to `connections.toml`, `snippets.toml`).
- Format: the whole file is an `age` message encrypted with a scrypt recipient from the master
  password. Plaintext payload is JSON: `{"version":1,"secrets":{"<connection name>":"<password>"}}`.
- Writes: encrypt the full payload, `fsutil.WriteFile(path, data, 0o600)` (atomic). Re-encrypt the whole
  file on every change; it is tiny.
- Work factor: the `age` default. Unlocking takes about a second by design. Do not lower it.
- A file that cannot be decrypted or parsed is an **error**, never an empty vault: the next save would
  overwrite it. A wrong master password and a corrupt file must give distinguishable errors.
- `Reset()` deletes `vault.age` after the frontend has confirmed (UI confirmation is the frontend's job,
  but the Go method must require an explicit argument such as `confirm string == "RESET"` so it cannot be
  called by accident).

### 4.3 Lifecycle

- States: `none` (no file), `locked`, `unlocked`.
- `Create(master)`: refuses if a file exists. Enforce a minimum master length (12 characters) and reject
  empty. Leaves the vault unlocked.
- `Unlock(master)`: derive/decrypt, keep the decrypted map and the master as `[]byte` in memory.
- `Lock()`: overwrite the master bytes and every secret's bytes with zeros, drop references, set state.
- **Idle auto-lock**: default 15 minutes since the last vault operation, implemented with an injectable
  clock so it is testable. A frontend setting may change it (`SetIdleTimeout(minutes)`; 0 means "until
  the app quits"); store that preference in the frontend's local storage and send it at unlock time, not
  inside the vault file.
- `ChangeMaster(old, new)`: verify `old`, re-encrypt with `new` to a temp file, rename. A crash mid-way
  must leave the old file intact.
- Lock on app quit (best effort).
- Go strings are immutable and may linger in memory; use `[]byte` for the master and for stored
  secrets where practical, and document that Go cannot guarantee scrubbing. Do not claim otherwise.

### 4.4 Wails service (what the frontend may call)

`VaultService` (register in `desktop/main.go` next to the other services; regenerate bindings, see
section 8):

```
Status() VaultStatus              // {exists bool, unlocked bool}
Create(master string) error
Unlock(master string) error
Lock()
ChangeMaster(old, new string) error
Reset(confirm string) error
SetIdleTimeout(minutes int)
HasSecret(connection string) bool          // does this connection have a remembered password
SetSecret(connection, password string) error   // requires unlocked
ForgetSecret(connection string) error
```

The **only** time a password crosses from JavaScript to Go is `SetSecret` (and master passwords in
`Create/Unlock/ChangeMaster`). No bound method may ever return a secret. The frontend must clear its
input fields right after each call.

### 4.5 Keeping secrets consistent with connections

`ProfilesService.Save(oldName, p)` and `Delete(name)` (in `desktop/profiles.go`) must keep the vault in
step: rename moves the secret (`Rename`), delete forgets it, and a profile whose auth is changed away
from `password` forgets it. These calls must **not fail** the save if the vault is locked: when locked,
leave the secret alone and record nothing; the safe failure mode is a stale entry, never a lost
connection. (Decide how stale entries are cleaned up, for example on the next unlock by comparing
against the profile list, and write the decision down in the spec.)

---

## 5. The fill mechanism (the risky part: read carefully)

### 5.1 Why not `SSH_ASKPASS`

`SSH_ASKPASS_REQUIRE=force` would route **every** prompt (host-key questions, 2FA codes, key
passphrases) to the helper, which cannot answer them. Relay's terminal must keep showing those to the
user. So filling happens at the terminal, narrowly.

### 5.2 Where each piece lives

- **Frontend** detects the prompt (it already parses terminal output) and sends a control message.
  Go never reads terminal bytes (invariant 1 of the project's philosophy). Detection is a **pure
  function** in `desktop/frontend/src/lib.ts`:

  ```ts
  parsePasswordPrompt(line: string):
    | { kind: "account"; user: string; host: string }   // "alice@10.0.0.5's password: "
    | { kind: "generic" }                                // "Password: "   /   "(alice@host) Password: "
    | null
  ```

  It must **reject**: `Enter passphrase for key ...`, `Verification code`, `Are you sure you want to
  continue connecting`, `[sudo] password for ...`, `Permission denied`, anything not ending at the
  prompt, and lines that merely *contain* the word password in output. Test it with real prompt strings
  from OpenSSH (password, keyboard-interactive, with and without a PAM banner), including odd spacing
  and ANSI-free text (xterm's buffer, not raw bytes).
- Detection runs only while the pane is a saved connection whose profile auth is `password`, the
  session is **not** `authed` (see `terminal.ts`), and the pane has not already been filled.
- On a match the frontend sends, over the same WebSocket, a text frame
  `{"t":"fill","kind":"account","user":"...","host":"..."}` (or `"kind":"generic"`).
- **Go** handles `fill` in `session.go`'s read loop. The session must know its **target profile name**
  (pass it into the session at creation in `terminal.go` `handle()`). Go then decides, using only data
  it owns, and writes `password + "\n"` straight to the PTY. **The password never goes to the
  frontend.** If any check fails, Go writes nothing and does not tell the frontend why (a generic
  `{"t":"fillrefused"}` frame is fine for UI state, without reasons that help an attacker).

### 5.3 Go-side checks (all must hold, each has a test)

1. The vault is unlocked and has a secret for the resolved connection.
2. The target profile's auth method is `password`.
3. **Account prompts:** resolve which saved connection the prompt is for by matching `user` and `host`
   against saved profiles (host compared to the profile's `Host`, case-insensitively, and to its name).
   Use that profile's secret. If the match is ambiguous (more than one profile with that user@host and a
   secret) refuse. This is what lets a jump host's own saved password work, and what stops the **target's
   password from ever being sent to a jump host**: a jump host's prompt names the jump host.
4. **Generic prompts** ("Password:" with no host) are only ever answered with the **session's own
   target** secret, and only if the session has **no jump host** (a hop could be the one asking).
5. **At most one fill per session per connection**, tracked in Go. If the prompt appears again (wrong
   password) do not retype: this prevents account lockouts.
6. A per-session cap of 2 fills total across all connections (target + at most one jump host).
7. Never fill after the session has authenticated. Go does not see the OSC marker, so the frontend must
   not send `fill` once `authed` is true, **and** Go must also refuse if the frontend reports `authed`
   in a `{"t":"authed"}` frame (add this frame; the frontend sends it when it flips `authed`). Belt and
   braces: a malicious remote that prints "password:" mid-session must not get the secret.
8. The session was started by Relay for a saved connection (not a local shell).

### 5.4 Other behaviour

- Connecting to a `password` connection that has a remembered password **while the vault is locked**:
  show the unlock dialog first. "Skip" opens the session normally (ssh will ask in the terminal).
  Do not block the connection on the vault.
- After a successful fill, show a subtle note in the pane header or status bar ("password filled")
  so the user knows it happened. No password text anywhere.
- If a fill was sent and ssh immediately asks again, the frontend shows a one-line hint ("The stored
  password was rejected. Update it in the connection's settings.") and does not retry.

---

## 6. Security requirements (acceptance criteria, not suggestions)

1. No bound Wails method returns a secret. A test asserts this by reflecting over `VaultService`'s
   exported method signatures.
2. No secret appears in: logs (`log.Printf` and friends), error strings, JSON sent to the frontend,
   `connections.toml`, the generated ssh config, `localStorage`, URLs. Add a test that creates a vault
   with a distinctive secret, runs the save/fill paths, and greps the config dir, the generated config
   and captured logs for it.
3. `vault.age` is `0600`; the config directory stays `0700`. Writes are atomic. A corrupt or wrongly
   keyed file is never replaced by an empty one.
4. `Lock()` zeros the in-memory secret and master bytes. Test that the byte slices were cleared.
5. Idle auto-lock works (injectable clock test) and `Get` after lock fails.
6. Filling follows every rule in 5.3. Each rule has a test that fails if the rule is removed.
7. Passwords are never typed before `authed`/after `authed` and never answer non-password prompts.
8. The unlock/create UI never echoes the master password, never stores it, and clears the input on
   success and failure.
9. Tests and fixtures contain only obviously fake secrets (for example `correct horse battery staple`
   style test strings), never anything resembling a real credential.

Threat model to write into the spec: the vault protects against theft of the file and casual disk
access. It does **not** protect against malware running as the user while the vault is unlocked (it can
read process memory or the PTY). State this plainly; do not oversell.

---

## 7. UI

Follow the existing visual language (flat, square structure, small control radii, theme variables from
`desktop/frontend/src/theme.css`; never hard-code colours, use CSS variables, the app has a light and
dark themes). Svelte 5 runes (`$state`, `$derived`, `$props`).

1. **Connection form** (`ProfileForm.svelte`, the Authentication section, around the
   `{#if p.auth === "key"}` block): when auth is `password`, show an optional **Password** field and a
   short note ("Stored encrypted in Relay's vault, unlocked with your master password. Leave empty to be
   asked each time."). On save: if a password was typed and the vault does not exist, open the **create
   vault** dialog; if it is locked, open the **unlock** dialog; then call `SetSecret`. Editing an existing
   connection shows "A password is saved" with Change and Forget actions (never the password). Replace
   the old note text that says passwords are never stored.
2. **Vault dialog** (new `VaultPanel.svelte`, opened from Tools or the status bar): create (master +
   confirm, with the "cannot be recovered" warning), unlock, lock, change master, reset (confirm by typing
   the word). Add a "Vault" tab to `ToolsPanel.svelte` or a standalone dialog; your call, keep it small.
3. **Status bar** (`StatusBar.svelte`): a small lock indicator (locked/unlocked), clickable.
4. **Unlock on connect** (section 5.4) as a lightweight modal.
5. **Settings** (`SettingsPanel.svelte`): "Lock the vault after" (5 / 15 / 60 minutes / until quit).

Every new interactive element needs an `aria-label` or visible text. Handle Escape and Enter sensibly.

---

## 8. How to build, run and test

```bash
make check            # CLI gate (Go fmt/vet/test -race, guardrails). Must stay green; you should not touch internal/.
make desktop-check    # desktop gate: guardrails, svelte-check (0 errors), vitest, vite build, go vet, go test
```

- Frontend: `cd desktop/frontend && npm ci`; `npm run check -- --output machine`; `npx vitest run`.
  Vitest has its own `vitest.config.ts` (Svelte browser build); DOM tests use `// @vitest-environment
  happy-dom` and `flushSync()` after `mount()` so actions attach. happy-dom ignores `detail` on
  `MouseEvent` and some `PointerEvent` fields: see `src/rows.test.ts` for the workarounds.
- Wails bindings: after adding or changing a bound service or its types, regenerate:
  `cd desktop && wails3 generate bindings -f '' -clean=true -ts -i` (the `-i` matters: the repo's
  bindings use interfaces). Commit the generated files under `desktop/frontend/bindings/`.
  Install the CLI with `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0` if missing.
- Running the app: `cd desktop && wails3 dev -config ./build/config.yml -port 9245`. Use
  `SSSH_HOME=/tmp/relay-home` to keep it away from your real data.
- macOS notes: the filesystem is case-insensitive (do not create `sidebar.svelte.ts` next to
  `Sidebar.svelte`); `timeout` does not exist; unix socket paths are limited to about 100 bytes.
- **Testing the fill logic without a real password server.** An unprivileged `sshd` cannot verify system
  passwords, so do not try. Instead write a tiny fake "ssh" script (a shell script that prints
  `alice@host's password: `, reads one line from the PTY, prints `GOT:<length>` or `DENIED`, and, for the
  jump host case, prints a second prompt naming another host) and drive it through the real
  `startSession` + control-message path in a Go test (see `desktop/integration_test.go` and
  `desktop/login_signal_test.go` for PTY test helpers such as `readUntil`). Assert what was typed by
  having the fake script echo a hash of what it received, never the secret.
- The reviewer has a real-window smoke harness (not in the repo) that drives the actual UI. To make your
  work verifiable in it, add a block to the smoke function in `desktop/frontend/src/App.svelte`
  (`RELAY_BENCH=smoke`), following the existing blocks (`out.snippets`, `out.keysWithPassphrase`): drive
  the vault dialog and the form through the DOM and record the observable results in `out`. Keep secrets
  out of `out`.

---

## 9. Documentation changes (honesty matters)

The project currently promises "No secrets are stored". After this feature that sentence is false and
must be changed rather than left to rot. Update, in the same branch:

- `CLAUDE.md`: the question "Does it store a secret?" and the "What this tool is" section. New truth:
  Relay stores **no** secrets by default; passwords are stored only if the user opts in per connection,
  only in the encrypted vault, never in `connections.toml`, the generated config or logs; key
  passphrases and 2FA are never stored by Relay. Keep the three invariants intact (they concern the CLI
  and the config file, and still hold).
- `README.md`: the "No secrets are stored" bullet and the Tools/Features text.
- `docs/superpowers/specs/2026-10-28-desktop-app-design.md`: new section **9r** (written first, see
  section 3) and a pointer from section 9q, which says password logins are unchanged.
- The text in `ProfileForm.svelte` that says a password is never stored.

Also record known gaps honestly in 9r: the vault is not part of backups; scrubbing in Go is best effort;
no recovery from a forgotten master password.

---

## 10. Suggested order of work (one commit per step, tests first)

1. Spec: section 9r (decisions, file format, threat model, gaps). Commit.
2. `desktop/secrets.go` + `desktop/vault.go`: the vault with create/unlock/lock/change/reset/get/set/
   forget/rename and the idle timer. Full unit tests (wrong master, corrupt file, atomicity, perms,
   zeroing, clock-driven auto-lock, rename/forget). Commit.
3. `VaultService` bindings + `ProfilesService` hooks (rename/delete/auth-change) + tests incl. "locked
   vault never fails a profile save" and "no bound method returns a secret". Regenerate bindings. Commit.
4. `parsePasswordPrompt` in `lib.ts` with a thorough table-driven vitest suite. Commit.
5. Go-side fill: `session.go` control frames (`fill`, `authed`), target tracking, the checks in 5.3, with
   the fake-ssh PTY tests (one test per rule). Commit.
6. Frontend wiring: detection in `workspace.svelte.ts`/`terminal.ts`, sending `fill`/`authed`, the
   rejected-password hint. Commit.
7. UI: form field, vault dialog, status bar, unlock-on-connect, settings. Component tests for the form
   and dialog where practical. Commit.
8. Docs (section 9). Smoke block in `App.svelte`. Run both gates. Push the branch and write a short
   summary for the reviewer: what changed, what you verified, what you could not verify, and any
   deviation from this document with the reason.

---

## 11. Definition of done

- [ ] `make check` and `make desktop-check` both exit 0, output read.
- [ ] Section 6 criteria all demonstrably met, each backed by a test that fails without the behaviour.
- [ ] `internal/` untouched; CLI guardrails unchanged and green.
- [ ] No secret anywhere it should not be (the grep test from 6.2 passes).
- [ ] Docs updated per section 9; spec section 9r present and matches the code.
- [ ] Bindings regenerated and committed; no stray files; `.svelte-check` and `dist` not tracked.
- [ ] Branch `feature/vault` pushed, not merged.

## 12. What the reviewer will check

Expect these questions, and answer them in your hand-off summary:

1. Can any path send a password to a host other than the one that asked for it? (Jump hosts, generic
   prompts, ambiguous user@host.)
2. Can a remote that prints "password:" mid-session obtain the secret?
3. Can a wrong stored password cause a lockout through retries?
4. Does any failure path (locked vault, corrupt file, crash during write) lose a connection or a
   secret, or silently replace a file?
5. Is every secret-bearing value kept out of logs, errors, config files, bindings and the DOM after
   submit?
6. Does the fill logic still hold when the frontend is wrong or malicious? (Go must not trust the
   frontend's claims beyond what 5.3 allows.)
7. Are the docs honest about what the vault does not protect against?
