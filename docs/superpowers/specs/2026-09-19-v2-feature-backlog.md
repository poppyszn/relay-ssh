# sssh v2 — Feature Backlog

**Date:** 2026-09-19
**Status:** Ready for implementation
**Audience:** An implementing agent (any model/tool), reviewed by Claude before merge.

This is a work order, not a suggestion list. Read the whole document — including
the ground rules — before writing any code. If a requirement is ambiguous, stop
and ask rather than guessing; do not silently narrow or expand scope.

## Context you must have before touching anything

Read these first, in full:

1. `/CLAUDE.md` — the working agreement. Every rule in it applies to this work
   without exception.
2. `docs/superpowers/specs/2026-09-11-sssh-v2-design.md` — why v2 exists, the
   three invariants, the architecture.

The short version, restated here because it is the most commonly violated part
of any handoff:

1. **sssh is never running while a session is open.** It `exec`s `ssh`,
   replacing its own process. Nothing you build may add a code path that spawns
   ssh as a child process, pipes its output through Go, or opens a PTY.
2. **sssh owns `~/.ssh/sssh.d/` and one marked block in `~/.ssh/config`.**
   Every write outside that sandbox is a bug, `known_hosts` merges excepted.
3. **The source of truth is `connections.toml`, a file a human can fix by hand
   in `vim`.** Nothing you build may make the store unreadable or unrepairable
   without sssh itself — no binary blobs, no fields only sssh can decode, in
   the plaintext store.

## Ground rules (non-negotiable)

- **TDD.** Write the failing test, run it, confirm it fails for the *reason
  you expect*, write the minimal code to pass it, run it, confirm it passes,
  then commit. A test you never watched fail proves nothing about what it
  tests. Follow this for every function, not just the feature as a whole.
- **Hermetic tests, no exceptions.** Every test that touches the filesystem
  builds its own root with `t.TempDir()` and a `paths.New(root)`. Never read or
  write the real `$HOME`. Never open a network connection. Never invoke the
  real `ssh` or `scp` binary — `internal/launch` already has a pattern
  (`execFunc`, overridable in tests) for exactly this; reuse it, don't
  reinvent it.
- **Atomic writes.** Anything that touches `connections.toml`,
  `~/.ssh/sssh.d/config`, or `~/.ssh/config` goes through `internal/fsutil`.
  Never `os.WriteFile` directly to one of those paths.
- **No new UI patterns.** All interactive prompting reuses the existing
  primitives in `internal/cmd`: `prompt`, `scanLine`, `menuPick`, `tryFzf`,
  and the `internal/style` package for coloring. Do not add a new prompting
  library, a new menu renderer, or anything resembling a TUI. If a feature
  needs a menu, it uses `menuPick`/`tryFzf`, the same way `PickFolder` and
  `PickKeyFile` already do.
- **No new dependencies beyond what this document explicitly authorizes.**
  The budget is 2 direct dependencies (see `scripts/guardrails.sh`). One slot
  is spent on the TOML library. Feature 1 below spends the second, explicitly,
  on `filippo.io/age` — this was budgeted for in the original design spec, so
  it is pre-approved. No other feature in this document may add a dependency.
  If you believe one is unavoidable, stop and say so instead of adding it.
- **No `golang.org/x/term`, or any raw-mode/termios manipulation, for any
  reason.** It is explicitly on the guardrails blocklist alongside the other
  TUI/terminal libraries v1 died from — see the `BANNED` pattern in
  `scripts/guardrails.sh`. This matters concretely for Feature 1 (passphrase
  entry): do not attempt to suppress terminal echo via syscalls to work around
  the ban. See Feature 1's explicit instructions on how to handle this.
- **`make check` must pass before you claim any feature done.** This runs
  `fmt`, `vet`, `test -race`, and `scripts/guardrails.sh`. Paste its output.
  "I wrote the code" is not "it's done" — running the gate and reading the
  output is the only thing that is.
- **One feature, one commit (or a small tightly-scoped set of commits for a
  large feature), in the priority order listed below.** Do not start feature
  *N+1* if feature *N*'s tests are not passing. Conventional commit messages,
  matching the style already in `git log` on this branch.
- **Do not refactor code outside a feature's stated scope.** If you notice
  something else worth fixing (dead code, a naming inconsistency, the
  package-layout deviation from the original design spec's diagram), do not
  fix it inline — note it at the end of your work for review instead.
- **Every field-level validation rule lives in `internal/config` and is reused
  by both the interactive prompt and `Store.Validate()`.** This codebase
  already made this mistake once and fixed it (see `ValidateName`,
  `ValidatePort`, `ValidateFolder` in `internal/config/validate.go`, and how
  `add.go` calls them for immediate, per-field feedback instead of failing
  only at Save time). Do not regress to the old pattern of collecting every
  field and discovering a problem only when the whole profile is saved. If a
  field's validity can be checked without needing an entire profile in hand,
  it goes through a retry loop like `promptName`/`promptPort`/`promptRequired`
  in `add.go`.
- **Never lose already-entered answers.** Any interactive flow you build must
  follow the same discipline: a bad answer to field 3 of 6 re-asks field 3,
  not the whole form. If a project-level rule (e.g. jump-chain cycle
  detection) can only be checked at the whole-Store level, catch it at Save,
  report the error, and route the user back to *just* the field that plausibly
  caused it — not the entire flow from scratch.
- **Guard every retry loop against exhausted input.** `internal/cmd/add.go`
  has `scanLine`, which returns `ok=false` on EOF, and `errCancelled` for
  loops to return instead of re-prompting a scanner with nothing left to give.
  Use this pattern for every new retry loop. A loop that can spin forever on
  a closed/piped stdin is a bug, not an edge case — it has already had to be
  fixed once in this codebase.

## How you will be reviewed

Claude reviews each feature's diff and test output before it is considered
mergeable. Per feature, be ready to show:

1. The failing test, before the implementation existed.
2. `make check` output, in full, passing.
3. A short manual smoke-test transcript (a real `sssh <command>` run against
   a `t.TempDir()`-rooted `paths.Paths`, or the built binary against a scratch
   `$HOME`) demonstrating the feature working end-to-end — not just unit
   tests in isolation.
4. Non-test LOC total after the change (`make check`'s guardrails output
   prints this) — flag it if you're getting close to the 2500 budget.

---

## Feature 1 — Real encrypted backup/restore (highest priority)

**Problem.** `sssh backup` currently writes a *plain, unencrypted* `tar.gz` and
prints a suggestion to pipe it through an external `age` CLI the user may not
have installed. This directly contradicts the first of the three reasons v2
exists at all (see the design spec): *"One encrypted file holding profiles and
keys, so a new laptop is a restore away."* Right now, nothing sssh produces is
actually encrypted.

**What to build.** In-process, passphrase-based encryption using
`filippo.io/age` (already present in `go.sum`, currently unused and marked
`// indirect` in `go.mod` — this is the pre-approved second dependency slot).

Use age's **scrypt-based passphrase recipient/identity**
(`age.NewScryptRecipient` / `age.NewScryptIdentity`), *not* public-key
recipients. A passphrase is the right model here: it requires no key
management from the user, matching the "restore on a brand new laptop" use
case — they won't have their old sssh keys available yet, that's the whole
point of the restore.

### Requirements

- `sssh backup [-o FILE]`:
  - Prompt for a passphrase, then prompt again to confirm it (two prompts,
    must match, retry on mismatch — reuse the retry-loop pattern, don't
    special-case this one).
  - **Passphrase entry is visible on screen.** This is deliberate, not an
    oversight: suppressing terminal echo requires either
    `golang.org/x/term` (banned) or hand-rolled termios/ioctl syscalls
    (platform-specific, a maintenance burden, exactly the kind of complexity
    v2 exists to avoid). State this plainly in the command's own output
    (e.g. a one-line notice before the prompt: `"Note: the passphrase will be
    visible as you type."`) so it is not a silent surprise. This is a
    conscious, documented tradeoff, not something to "fix" by adding a banned
    dependency.
  - Also accept the passphrase from an `SSSH_BACKUP_PASSPHRASE` environment
    variable, checked first, so scripted/CI backups don't need an interactive
    terminal at all. If set, skip both prompts entirely.
  - Encrypt the existing tar.gz archive (don't change what goes *into* the
    archive — same `connections.toml` + `keys/` contents as today) with the
    scrypt recipient.
  - Output filename: if the resolved output path doesn't already end in
    `.age`, append it, so an encrypted backup is visually distinguishable from
    the old plaintext format at a glance. Update the default name accordingly
    (e.g. `sssh-backup.tar.gz.age`).
- `sssh restore FILE`:
  - If `FILE` ends in `.age`, decrypt first (passphrase from
    `SSSH_BACKUP_PASSPHRASE` or an interactive prompt, same visibility
    caveat as above), then extract exactly as today.
  - If `FILE` does *not* end in `.age`, treat it as a legacy plaintext
    backup and extract directly — do not break restoring an archive someone
    made with the current, unencrypted `sssh backup`.
  - A wrong passphrase must produce one clear, specific error
    (e.g. `"incorrect passphrase (or the file is corrupted)"`) — never a raw
    age/cipher stack trace, never a panic.
- **A passphrase must never be written to disk, logged, printed back, or
  embedded in any error message.** Error messages describe *what* failed
  ("decryption failed"), never *with what value*.
- Run `go mod tidy` once `filippo.io/age` is actually imported and used. Both
  `filippo.io/age` and `github.com/BurntSushi/toml` should then show as
  **direct** (not `// indirect`) dependencies in `go.mod` — `toml` is already
  directly imported in `internal/config/store.go` but is currently
  mismarked, which is worth fixing as part of this change since you're
  touching `go.mod` anyway. After this, `scripts/guardrails.sh`'s dependency
  check should report exactly 2 of 2 — confirm this in your `make check`
  output, don't just assume it.

### Explicitly out of scope

- Public-key (`age -r <recipient>`) support. Passphrase only.
- Any change to what's *inside* the archive (still `connections.toml` +
  `keys/`, unchanged).
- Any terminal echo suppression, by any mechanism.

### Tests to write (hermetic, `t.TempDir()`-rooted)

- Round trip: backup with a known passphrase (via `SSSH_BACKUP_PASSPHRASE` in
  the test, not interactive input), restore with the same passphrase, confirm
  the store matches.
- Wrong passphrase on restore produces the specific, clear error — not a
  panic, not a generic error.
- Restoring a `.tar.gz` (no `.age` suffix, i.e. the legacy plaintext format)
  still works unchanged.
- Output filename gets `.age` appended when the user's `-o` value doesn't
  already have it; is left alone when it does.

---

## Feature 2 — Interactive `sssh edit`

**Problem.** `sssh edit <name>` (`internal/cmd/other.go`, `EditCmd`) only
prints the profile's current fields. It cannot change anything — every edit
today means hand-opening `connections.toml`.

### Requirements

- Reuse the exact same field-level helpers `add.go` already has
  (`promptRequired`, `promptPort`, `promptAuthType`, `PickKeyFile`,
  `PickFolder`) — do not write a second copy of any of this logic.
- For every field, show the current value as part of the prompt and let a
  blank answer mean "keep it as-is". Concretely: `"Hostname or IP [10.0.0.5]:
  "` — an empty line keeps `10.0.0.5`, anything else replaces it. This is a
  different affordance than `add`'s prompts (which treat blank as "required,
  try again" for some fields) — `edit`'s prompts must distinguish "blank
  because I want the default/current value" from "blank because I have
  nothing to say" correctly for each field's semantics. Think this through
  per field rather than copy-pasting `add`'s prompt calls unchanged.
- Renaming: the new name must pass `config.ValidateName` and must not collide
  with a **different** profile — but must be allowed to "collide" with
  itself (i.e., leaving the name unchanged is always valid).
- After collecting every field, validate the **whole edited Store** (not just
  the one profile) with `Store.Validate()` before saving — this is what
  catches jump-chain cycles introduced by the edit. If it fails, report the
  error and re-open *only the jump-target prompt* for another attempt, not
  the entire edit flow — apply the same "don't lose already-entered answers"
  discipline used in `add.go`.
- Saving goes through `config.Save`, same as everywhere else (atomic write).

### Explicitly out of scope

- Editing tunnels (that's Feature 6, if you get to it).
- A "diff preview before saving" UI. Just prompt each field and save.

### Tests to write

- Editing one field (e.g. host) leaves every other field untouched.
- Pressing enter on every prompt (i.e., changing nothing) round-trips the
  profile exactly.
- Renaming to an already-taken name is rejected and re-prompts just the name.
- Renaming to the profile's own current name is accepted (no false-positive
  collision).
- Introducing a jump cycle via edit is rejected at save, and the flow returns
  to the jump prompt specifically (verify via a scripted stdin sequence, the
  same style as `add_test.go`'s `TestAddRetriesAfterDuplicateNameWithoutLosingLaterAnswers`).

---

## Feature 3 — Tags: capture in `add`/`edit`, filter in `ls`

**Problem.** `config.Profile.Tags []string` already exists in the data model
and already round-trips through TOML (see `store_test.go`), but nothing in
the CLI ever sets it or reads it. It's dead weight in the type until this
lands.

### Requirements

- `add` and `edit`: one new prompt, `"Tags (comma-separated, optional)"`.
  Split on commas, trim whitespace from each, drop empty entries. No
  validation beyond that (tags are free text; do not reuse `ValidateName`'s
  character restrictions here — no ssh config file directive is ever
  generated from a tag, so the security rationale for that restriction
  doesn't apply).
- `ls --tag <name>`: filters both the flat and `--tree` output to only
  profiles whose `Tags` slice contains an exact, case-sensitive match for
  `<name>`. No wildcards, no partial matches.
- Default `ls` output (no `--tag`) is unchanged — do not print tags inline
  unless `--tag` is used to filter. Keeping the default view uncluttered
  matters more than showing every piece of metadata at once.
- `--tag` combined with `--tree` must work (filter first, then group the
  filtered set by folder, same as today's ungrouped-then-grouped logic).

### Tests to write

- Round trip: a profile saved with tags via `add`, loaded back, has those
  exact tags.
- `ls --tag foo` includes only profiles carrying `foo`, in both flat and
  tree mode.
- `ls --tag` with a tag nobody has produces the existing "(no profiles)"
  message (or an equivalent empty-result message — reuse the existing one
  rather than inventing new copy).
- A profile with no tags is invisible to any `--tag` filter but still
  appears in unfiltered `ls`.

---

## Feature 4 — Jump host and agent forwarding in `add`/`edit`

**Problem.** `Profile.Jump` and `Profile.ForwardAgent` are fully implemented
in `sshconfig.Generate` (`ProxyJump`, `ForwardAgent yes`) and in
`Store.Validate`'s cycle detection, but `add`/`edit` never ask for them —
today, setting either means hand-editing the TOML.

### Requirements

- After the auth-type prompt (and key prompt, if key auth), add:
  - `"Jump through another profile first? (name, or blank for none)"`.
    - Blank → `Jump = ""`, move on.
    - Non-blank → must be an existing profile name (`s.Find`); if not found,
      report and re-prompt *this field only*.
    - Must reject the profile jumping through itself (compare against the
      name already chosen/being edited) inline, with a clear message,
      before it ever reaches `Store.Validate()`.
    - A jump chain that would form a *deeper* cycle (A→B→A via a third hop)
      is allowed to fall through to `Store.Validate()`'s existing
      `checkJumpCycles` check at save — on failure there, report the error
      and re-open the jump prompt specifically, not the whole form.
  - `"Forward the SSH agent to this host? (y/N)"` — blank or anything
    starting with `n`/`N` → `false`; anything starting with `y`/`Y` → `true`.
    Default (blank) is `false`.

### Explicitly out of scope

- Any validation of whether the jump target is *itself* reachable (that's
  what `ssh` will discover at connect time; sssh's job here is just to not
  save an obviously-broken reference or an immediate self-cycle).

### Tests to write

- Setting a jump to an existing profile saves correctly and round-trips.
- Setting a jump to a nonexistent name is rejected inline, re-prompts.
- Setting a jump to the profile's own name is rejected inline, re-prompts.
- A three-hop cycle introduced via the jump prompt is caught at save (not
  inline, since that's out of scope for the inline check), and the flow
  returns to the jump prompt rather than restarting the whole form.
- `ForwardAgent` defaults to `false` on blank input, `true` on `y`/`yes`.

---

## Feature 5 — `sssh doctor --fix`

**Problem.** `doctor` (`internal/cmd/doctor.go`) only ever reports; for every
fixable issue it finds, the user has to separately remember and run
`sssh sync`.

### Requirements

- Add a `--fix` flag. When set, for each issue that has a *deterministic,
  already-implemented* fix, apply it by calling the existing function (not a
  reimplementation) and re-check afterward:
  - Missing or stale generated config (`~/.ssh/sssh.d/config`) → call the
    existing `sshconfig.Write` path (the same one `Sync` uses).
  - Missing managed Include block → call the existing
    `sshconfig.EnsureInclude`.
- After applying fixes, re-run every check and print the final state, same
  format as a normal `doctor` run, so the user sees before-and-after in one
  invocation (or just the final "OK" — your call on presentation, but the
  underlying checks must actually re-run against post-fix state, not just
  print an optimistic "fixed!" without verifying).
- Anything **not** deterministically fixable by existing code (a missing
  adopted key, a `connections.toml` that fails to parse at all) is reported
  exactly as today — `--fix` must never delete, regenerate from scratch, or
  guess at content for either of those. If you're unsure whether something
  is "deterministically fixable", treat it as not fixable and just report it.
- `--fix` must not touch anything outside the sandbox (`~/.ssh/sssh.d/`) plus
  the one marked Include block in `~/.ssh/config` — Invariant 2, same as
  everywhere else in this codebase. It must never rewrite unrelated content
  in `~/.ssh/config`.

### Tests to write

- `doctor --fix` on a store with a stale generated config regenerates it and
  the re-check reports it as current.
- `doctor --fix` on a config missing the Include block adds it (verify with
  `sshconfig.HasBlock`) without disturbing existing unrelated content in
  `~/.ssh/config` (same fixture style as
  `TestEnsureIncludePrependsToExistingConfig` in `include_test.go`).
- `doctor --fix` with a missing adopted key still reports it as missing —
  confirms `--fix` didn't attempt to fabricate one.
- `doctor` (no flag) behavior is completely unchanged.

---

## Feature 6 — Tunnel configuration in `add`/`edit` (lower priority / stretch)

Only attempt this after Features 1–5 are done, reviewed, and merged. If you
run low on context or time, stop after Feature 5 and hand back a clean state
rather than leaving this half-implemented.

**Problem.** `Tunnel` (local/remote/dynamic port forwarding) is fully
implemented in the data model, `Tunnel.Validate()`, and `sshconfig.Generate`,
but is only reachable by hand-editing the TOML.

### Requirements

- After the folder step in `add` (and as an available step in `edit`), ask:
  `"Add a port forward? (y/N)"`. If yes, loop:
  - Type: numbered menu, `local` / `remote` / `dynamic` (reuse `menuPick`).
  - Listen port: required, reuse `promptPort`-style validation (it's a port
    number, same range rule — but note `Tunnel.Listen` is a `string` field
    already in the type, not an `int`; validate it *as* a port number, but
    store it as the string the type expects, matching existing behavior in
    `generate.go`).
  - Target (`host:port`): required for `local`/`remote`, must be *absent* for
    `dynamic` — don't even prompt for it in the dynamic case.
  - Validate the fully-built `Tunnel` with `Tunnel.Validate()` before
    accepting it; on failure, report and re-loop *this tunnel's fields*, not
    the ones already accepted.
  - After each tunnel, ask `"Add another? (y/N)"`.

### Tests to write

- Adding one local tunnel round-trips through save/load correctly.
- Adding multiple tunnels preserves order (matches
  `TestGenerateEmitsEveryTunnelInOrder`'s existing ordering guarantee).
- A dynamic tunnel is never prompted for a target.
- An invalid tunnel (e.g. local with no target) is rejected and re-prompts
  only that tunnel, not the whole list already accepted.

---

## Priority order (do not reorder without asking)

1. Feature 1 — encrypted backup/restore
2. Feature 2 — interactive edit
3. Feature 3 — tags
4. Feature 4 — jump host / agent forwarding
5. Feature 5 — doctor --fix
6. Feature 6 — tunnels (stretch)

## Definition of done, per feature

- [ ] Failing test written and confirmed to fail for the right reason
- [ ] Minimal implementation makes it pass
- [ ] `make check` passes in full (paste the output)
- [ ] Manual smoke-test transcript included
- [ ] Non-test LOC reported, flagged if within ~200 lines of the 2500 budget
- [ ] One commit (or a tight, clearly-related set), conventional commit
      message, no unrelated changes mixed in
- [ ] Nothing outside this feature's stated scope was refactored
