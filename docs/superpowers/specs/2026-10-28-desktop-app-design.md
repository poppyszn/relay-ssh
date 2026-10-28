# Relay (super-ssh Desktop): Design and Handoff

**Date:** 2026-10-28
**Status:** Draft. Decision D1 below needs the owner's explicit confirmation before any desktop code is written.
**Owner:** Precious Okpor
**Module:** `github.com/poppyszn/super-ssh-v2`

## 0. How to use this document (read first)

You are picking this project up in a new session with no memory of earlier ones. Do these in order:

1. Read `CLAUDE.md`, then `docs/superpowers/specs/2026-09-11-sssh-v2-design.md`, then `docs/superpowers/specs/2026-09-19-v2-feature-backlog.md`. They explain why the CLI is built the way it is.
2. Look at what has actually been built: `git log --oneline --all`, `git worktree list`, `git status`, and the code under `internal/` and `cmd/sssh/`. Do not trust this document over the code. If they disagree, the code wins, and you should say so.
3. Run `make check` and read the output. Record the baseline (pass or fail, non-test line count versus the 2500 budget) before changing anything.
4. Do not start desktop feature work until Section 3 (D1) is confirmed by the owner and the Phase 1 spike in Section 8 has passed.

## 1. Where things stand today

Facts below come from inspecting the repo on 2026-10-28. Verify them.

- `sssh` v2 is a Go CLI: a connection vault that generates OpenSSH config and then `exec`s `ssh`. Direct dependencies: `filippo.io/age` and `github.com/BurntSushi/toml`. Go 1.27.0.
- Packages present: `internal/cmd`, `config`, `fsutil`, `keys`, `launch`, `paths`, `sshconfig`, `style`. Backup, import, doctor, and the pickers currently live inside `internal/cmd` alongside their prompts, not in separate packages as the original spec proposed.
- Commit history suggests all six features in the backlog are implemented: encrypted backup and restore, interactive edit, tags, jump host and agent forwarding, `doctor --fix`, and tunnels. Confirm with `make check` and a manual smoke test.
- Branch state: `main` is behind `fix-review-followups` by three commits (folder-qualified addressing with auto-sync, hidden backup passphrase entry, shell tab-completion). There is also a worktree under `.claude/worktrees/sssh-v2-foundation`. Work out which branch is current, and merge or clean up before building on it.
- Housekeeping: an untracked `sssh-backup.tar.gz.age` sits in the repo root and `.claude/` is untracked. Add both to `.gitignore` (the backup file must never be committed) after checking it is only a test artifact.
- v1 history matters. v1 was a terminal multiplexer with an in-process VT emulator and was abandoned because bytes flowing through Go caused slow copy, stalls on long output, and heat. `CLAUDE.md` records this and forbids terminals, TUIs, and PTY imports in the CLI.

## 2. Product goal

A fast, modern desktop app for DevOps engineers, in the spirit of Devolutions RDM but without the slowness: a connection tree, a very good terminal, and the everyday tools in one place. Targets: macOS and Windows first, Linux as a bonus. The owner develops on macOS (Apple silicon).

Non-goals for the first release: RDP, a web or hosted version, team sync or shared vaults, storing passwords, reimplementing the SSH protocol.

## 2a. Name

The desktop app is called **Relay**. The CLI stays `sssh`. The owner has confirmed that `relay` is already taken, so every identifier that must be unique uses `relay-ssh`:

- Display name in the app and window title: Relay.
- Binary, package, app bundle, repo, and Homebrew names: `relay-ssh`.
- Code directory stays `desktop/` (Section 6). Do not rename the existing module or the CLI.
- In Phase 0, still check that `relay-ssh` is free on Homebrew (formula and cask), GitHub, and a usable domain, and record the result in a commit message. If it is also taken, stop and ask the owner.

## 3. D1: this reverses a documented decision (needs owner confirmation)

`CLAUDE.md` and the v2 spec say, permanently, that sssh will never contain an embedded terminal, tabs, PTYs, or terminal emulation. The desktop app requires all of those. This is a deliberate change of direction, not a refinement.

Proposed resolution, to confirm with the owner:

- The CLI keeps every invariant and every guardrail in `CLAUDE.md`. Nothing about `sssh` the command changes.
- The desktop app is a separate front end with its own spec, its own invariants (Section 5), and its own budgets. It shares the data model and core logic with the CLI.
- v1's failure was architectural: Go parsed, gridded, and diffed every byte. The desktop app avoids that by not doing terminal emulation in Go at all. Go only moves raw bytes between a real `ssh` process and `xterm.js`, which renders in the webview.

If the owner does not confirm D1, stop and ask what to build instead.

## 4. Architecture

```
Frontend (webview)           Go backend (Wails)                OS
-------------------          --------------------------        -----------------
xterm.js + WebGL     <--->   session manager (raw bytes) <-->  PTY running `ssh <profile>`
connection tree UI   <--->   core library (config, keys,        ~/.ssh/config (Include)
key / backup / doctor        sshconfig, backup, doctor)         ~/.config/sssh/connections.toml
```

Principles:

1. OpenSSH stays the SSH client. The app spawns the system `ssh` binary inside a PTY, addressing the profile by its Host alias from the generated config. Jump hosts, tunnels, agent forwarding, passwords, passphrases, and 2FA all keep working because `ssh` handles them and prompts inside the terminal. No SSH protocol code, no stored secrets.
2. Go never interprets terminal output. No VT parser, no cell grid, no screen diff. Bytes in, bytes out.
3. One source of truth: `connections.toml`. The app and the CLI read and write the same file and regenerate the same `sssh.d/config` through the same code. A change made in one must appear in the other.
4. The `~/.ssh/sssh.d/` sandbox rule still applies. Atomic writes still apply.

## 5. Desktop invariants and performance rules

Treat violations as defects.

- No terminal emulation in Go. The renderer is `xterm.js` with the WebGL renderer addon (verify it works in WKWebView and WebView2).
- Flow control is mandatory. Go reads the PTY in bounded chunks, the frontend acknowledges after `term.write` completes, and Go pauses reading when unacknowledged bytes pass a high watermark. Without this, a large `cat` will stall the UI exactly as v1 did.
- Move terminal bytes over the cheapest channel that works. Wails events serialize through JSON and may be too slow for bulk output. Measure it in the spike. A loopback WebSocket bound to `127.0.0.1`, with a random per-launch token and an Origin check, is the likely alternative.
- Idle sessions cost near zero CPU. Hidden tabs must not render.
- Measured targets for the spike (record real numbers, do not assume): `seq 1 5000000` completes while the UI stays responsive and typing remains instant; `cat` of a roughly 200 MB file does not freeze the window; selecting and copying 100k lines is immediate; five idle sessions sit near 0% CPU.

## 6. Repo layout and guardrails

`scripts/guardrails.sh` enforces CLI budgets across the whole repo: 2 direct dependencies, 2500 non-test lines, 0 PTY imports, and so on. Desktop code would break them on day one. Do not silently raise any number, since `CLAUDE.md` forbids it.

Proposed layout, to be evaluated and confirmed by Claude Code:

- Keep the CLI module exactly as is, with guardrails untouched.
- Put the app in `desktop/` with its own `go.mod` (tied together with `go.work`), its own dependencies, and its own guardrail script.
- Core logic the app needs is currently trapped inside `internal/cmd` next to interactive prompts. Extract the UI-free parts (backup and restore, import, doctor checks, profile CRUD) into importable packages first, as a refactor with no behavior change and the existing tests as the safety net. Check that Go's `internal/` visibility rules allow the app to import them, or move them to a public package path.
- The guardrails change that excludes `desktop/` from the CLI counts needs a commit message explaining why, per `CLAUDE.md`.

## 7. Cross-platform notes

- Webviews differ: WKWebView (macOS), WebView2 (Windows), WebKitGTK (Linux). Test terminal rendering and fonts on each.
- PTY: Unix PTY on macOS and Linux, ConPTY on Windows. Windows also needs the OpenSSH client available. Detect and report it in `doctor`.
- ssh agent: `SSH_AUTH_SOCK` on Unix, the Windows OpenSSH agent or Pageant on Windows.
- Paths, shortcuts (Cmd versus Ctrl), and window chrome need small per-OS handling.
- Build each OS on its own CI runner (GitHub Actions). macOS needs code signing and notarization (Apple Developer account). Windows signing is optional early.
- Check the current stable Wails release and its status (v2 versus v3) before choosing. Do not assume.

## 8. Phases and exit criteria

**Phase 0: baseline and housekeeping.** `make check` green, name availability checked (Section 2a), branches reconciled, `.gitignore` fixed, core logic extracted from `internal/cmd`. Exit: CLI behavior unchanged, tests green.

**Phase 1: terminal spike (go or no-go gate).** A minimal Wails window with one `xterm.js` tab running `ssh <profile>` in a PTY, with flow control. Measure the Section 5 targets on macOS and report real numbers. Exit: targets met, or a written explanation of what failed and the alternatives. If the targets cannot be met, stop and tell the owner before building more.

**Phase 2: connection manager UI.** Tree of folders and profiles, create, edit, delete, tags, search, and a command palette to open a connection fast. Reuses core logic, so the CLI sees every change.

**Phase 3: terminal quality.** Tabs, split panes, search in scrollback, copy and paste, themes, fonts, reconnect handling. (Tabs and panes are allowed here because the invariant moved: the terminal is rendered by xterm.js, not Go.)

**Phase 4: everyday tools.** Key management, encrypted backup and restore UI, `doctor` UI, port forward management, a file browser. Decide the file transfer approach (see D3).

**Phase 5: Windows build and CI, signing and release.**

**Later:** snippets, RDP through `guacd` as a component (not the full Guacamole stack), Linux packaging.

## 9. Open decisions

- **D1:** reverse the no-terminal invariant for the desktop app only (Section 3). Owner must confirm.
- **D2:** repo layout (Section 6). Claude Code to recommend after reading the code.
- **D3:** file browser and transfer. Options are shelling out to `sftp` or `scp` (consistent with "OpenSSH is the client") versus a Go SFTP library over an SSH connection (a new dependency and protocol code). Prefer the first unless it proves unusable.
- **D4:** terminal transport (Wails events versus loopback WebSocket). Decide from spike measurements.
- **D5:** Wails version.

## 9a. Decisions made (2026-10-29)

The owner delegated D2 to D5 with the instruction to use the latest stable tech. D1 is treated as accepted on that basis; the owner can still veto it.

- **D1:** accepted. The CLI keeps every invariant; the desktop app has its own (Section 5).
- **D2:** `desktop/` with its own `go.mod`, tied by `go.work`. Go's `internal/` rule is path-based, so a module whose import path sits under `github.com/poppyszn/super-ssh-v2/` can import `internal/core`; confirm in Phase 1. Phase 0 extracted the UI-free logic into `internal/core` (profile resolve/remove, ssh_config import, backup/restore, doctor checks); `internal/cmd` is now a thin prompt/print layer over it.
- **D3:** shell out to `sftp`/`scp`, as preferred above.
- **D4:** open until the Phase 1 spike measures Wails events against a loopback WebSocket.
- **D5:** Wails **v3.0.0** (stable, released 2026-10-26). Frontend: Svelte 5, Vite 8, TypeScript, `@xterm/xterm` 6.0 with `@xterm/addon-webgl`, `-fit`, `-search`. Re-verify versions at the start of Phase 1.
- Name check (Section 2a) is still to do in Phase 0.

## 9b. Phase 1 spike results (2026-10-29, macOS 26.6, Apple silicon, Wails 3.0.0, WKWebView)

Code: `desktop/` (Go PTY + loopback WebSocket + xterm.js 6 with WebGL). Benchmark: `RELAY_BENCH=run` / `RELAY_BENCH=idle` on the built binary, run inside the real Wails window.

| Target | Result |
| --- | --- |
| `seq 1 5000000` (59 MB, 5M lines) completes, UI stays responsive | 9.2 s; worst timer lag 16 ms, worst frame gap 35 ms. A bare PTY read of the same output takes 4.3 s, because `seq` writes one line per `write()`. |
| `cat` of 200 MB does not freeze the window | 2.6 s; worst timer lag 13 ms, worst frame gap 29 ms |
| Select 100k lines | 44 ms (`selectAll` + `getSelection`, 100k scrollback) |
| Typing stays instant | echo round trip median 1 ms idle, 1 ms after a flood |
| Five idle sessions near 0% CPU | about 0.5% total (app + 5 shells + WebKit processes), four sessions in hidden tabs |

Findings:
- **D4 decided: loopback WebSocket.** It works from the `wails://localhost` origin under WKWebView (no mixed-content or CSP block). Wails events were not measured because the WebSocket already meets every target.
- WebGL renderer loads in WKWebView. WebView2 and WebKitGTK are untested.
- The PTY reader must coalesce: one frame per read cost 13.2 s for `seq`, coalescing cut it to 9.2 s.
- The webview's own Origin is `wails://localhost`; the socket checks an exact Origin allow-list, since the WebSocket library's host comparison cannot match a custom scheme.
- Windows will need ConPTY instead of `creack/pty`.
- Gate: **go.**

## 10. Working agreement (carried over from the CLI)

TDD (watch the test fail first), hermetic tests using `t.TempDir()` with no real `$HOME` and no network, atomic writes, conventional commits per task, spec before code for any new command or data model change, and `make check` before claiming anything is done. Do not report work as complete on the strength of having written it.
