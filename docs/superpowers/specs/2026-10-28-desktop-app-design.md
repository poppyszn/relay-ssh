# Relay (super-ssh Desktop): Design and Handoff

**Date:** 2026-10-28
**Status:** Draft. Decision D1 below needs the owner's explicit confirmation before any desktop code is written.
**Owner:** Precious Okpor
**Module:** `github.com/poppyszn/relay-ssh`

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
- Code directory stays `desktop/` (Section 6). The CLI keeps the name `sssh`. The Go module was renamed from `github.com/poppyszn/super-ssh-v2` to `github.com/poppyszn/relay-ssh` on 2026-10-29 to match the public repository.
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
- **D2:** `desktop/` with its own `go.mod`, tied by `go.work`. Go's `internal/` rule is path-based, so a module whose import path sits under `github.com/poppyszn/relay-ssh/` can import `internal/core`; confirm in Phase 1. Phase 0 extracted the UI-free logic into `internal/core` (profile resolve/remove, ssh_config import, backup/restore, doctor checks); `internal/cmd` is now a thin prompt/print layer over it.
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

## 9c. Phase 2 status and design direction (2026-10-29)

Built: sidebar tree (folders, tags, search with `#tag` prefix filter), create/edit/delete form including jump host, agent forwarding and tunnels, quick-open palette (Cmd/Ctrl+K), New connection (Cmd/Ctrl+N). All changes go through `internal/core` (`SaveProfile`, `RemoveProfile`), so the CLI sees them. Renaming a profile repoints other profiles' jump references; deleting a jump host still in use is refused.

Visual direction (owner-provided reference, a Windows terminal with a deep navy-teal theme): dark navy surfaces, slim tab strip with close buttons, rounded chrome, muted chips. The reference's per-command blocks (a header with working directory, git branch and duration; red-tinted failed commands) are the model for Phase 3. They must be built in the frontend with xterm markers/decorations driven by shell-integration escape sequences (OSC 133), never by Go parsing the stream. Needs a remote shell that emits them, so treat it as an optional enhancement with a plain terminal as the fallback.

Known gaps: only one terminal at a time (tabs are Phase 3); no import-from-`ssh_config` or key picker in the UI yet.

## 9d. Phase 3 progress (2026-10-29)

Design: the approved Claude Design canvas (flat structure with square tabs, edge-to-edge bar highlights, hairline borders; small radii only on controls; round status dots; Inter for UI, JetBrains Mono for the terminal, both bundled).

Done (stage A/B): tab strip as the macOS title bar, flat sidebar, status bar, quick open with actions, redesigned connection form with a live preview from the real generator (`ProfilesService.Preview`) and a native key-file picker, tabs, split panes (drag to resize, per-pane headers when split), reconnect (button and Enter), native app menu with shortcuts (Cmd on macOS; Ctrl+Shift elsewhere so Ctrl+D and Ctrl+W still reach the shell). Verified under `wails3 dev` with `RELAY_BENCH=smoke`: tabs, splits, a command in every pane, dropped session detected, reconnect, close pane.

Done (stage D, command blocks): the shell prints OSC 133 markers (A prompt start, B prompt end, D;exit) and the frontend (`blocks.ts`) draws each command as an xterm decoration: top divider, green or red left stripe, red tint on failure, and a right-aligned `exit N · duration` badge. Go never reads the markers. Local zsh and bash get the hooks injected (wrapper `ZDOTDIR` / `--rcfile`, user's own startup files sourced first, nothing modified). Remote hosts: a Shell menu / palette action "Enable command blocks in this session" types a visible one-liner into the user's bash or zsh; it is never sent automatically. Type-ahead and full-screen programs are handled; tests cover the marker state machine, and `RELAY_BENCH=smoke` verifies a green and a red block in the real window.

Layout follows the owner's Warp reference (2026-10-29): content is anchored to the bottom of the pane (the screen is shifted down by the unused rows until the buffer fills; `anchorBottom` in `terminal.ts`), each block has a blank row above and below it printed by the shell hooks (the block is finished at the next prompt so it includes the bottom row), dividers bleed edge to edge through the terminal padding, and type is 14px at line-height 1.25 (about 22.5px rows). Padding is on the `.xterm` element because FitAddon subtracts that, not a parent's padding.

Scrollbar: xterm 6 draws its own scrollbar, so it is restyled in `theme.css` to a 5px slider with a 10px grab area, pushed out through the terminal padding to the pane's right edge.

Done (stage C): scrollback search (xterm search addon; find bar with match count, previous/next, match case; Cmd/Ctrl+F from the Shell menu; Enter and Shift+Enter step through matches; Esc closes). Settings panel (Cmd+, or the gear): three dark themes (Midnight, Graphite, Obsidian), font (JetBrains Mono or system monospace), size, line spacing, copy on select, cursor blink. Themes set CSS variables on the page and the xterm theme on every open terminal, live; settings persist in localStorage (best effort). Also fixed: a failed connection left a half-built terminal and its WebGL context in the pane, and the ack timer and resize observer were never cleaned up on close.

Remaining: a clipboard check in the real window (native Edit menu copy/paste is expected to work; copy on select is the explicit fallback), snippets (section 9e), Windows ConPTY, and Phases 4 and 5.

## 9e. Backlog notes from the owner (2026-10-29)

- **Per-block options menu (done 2026-10-29):** a `⋯` button on the block's command row, shown while the pointer is over the block, opens a menu with Copy command, Copy output and Copy command and output, with a Copied confirmation. The frontend reads the text from the xterm buffer (the command starts at the column where the prompt ended; wrapped rows are joined; the trailing padding rows and rows of written spaces are dropped), so Go stays out of it, and copying goes through the desktop runtime clipboard. More actions (re-run, collapse, share) can be added to the same menu. Original note: a three-dot button on every command block, like Warp's, opening actions such as copy command, copy output, copy all (command plus output), and likely more (for example re-run, share, collapse). Blocks already know their start and command markers; the output range is the rows between the command row and the block end, so copy actions can read it from the xterm buffer without Go involvement. Needs a hover affordance on the block overlay (the overlay is `pointer-events: none` today, so the button needs its own hit area), keyboard access, and tests for the range extraction.
- **Block header (done 2026-10-29):** each block shows its directory (with `~` for home) and git branch on its first row, the blank padding row above the prompt, so the user's own prompt is left alone and nothing about the shell prompt changes (the original idea of Relay drawing its own prompt was dropped for that reason). The shell hooks send it in the existing prompt marker, `OSC 133;A;<branch>;<cwd>`; the frontend draws it as a decoration. Duration and exit status stay on the command row. Safety: control characters are stripped from both values in the shell, and `;` from the branch, so no directory or branch name can inject terminal sequences; the frontend renders them as text, never HTML. The branch is `git symbolic-ref` or the short commit when detached (one cheap git call per prompt, nothing when git is absent). Hooks from before this change send no context and simply get no header; remote shells need the integration snippet again. Tests run real zsh and bash in a PTY (a git repo, a sibling directory sharing the home prefix, a directory with an escape character in its name) and the remote snippet.

## 9f. Phase 4 progress (2026-10-29)

Done: a Tools dialog (wrench in the tab bar, Shell menu, or palette) over a `ToolsService` that is a thin layer on `internal/core`:
- **Health**: the doctor checks with one-click "Fix N issues" (regenerate config, restore the Include block) and a re-check.
- **Backup and restore**: passphrase-encrypted backup via a native save dialog (written 0600, removed on failure, passphrase never stored); restore via a native open dialog with an explicit confirm because it replaces current data.
- **Import**: `~/.ssh/config` Host entries become connections. Fixed along the way: adjacent Host blocks with no blank line were skipped, one incomplete entry aborted the whole import, and entries without HostName or User now get ssh's defaults (alias, current user). Bad entries are skipped and reported. The CLI line budget went 2900 to 3000 for this (recorded in CLAUDE.md).
- **Keys**: adopted keys (in backups) and private keys found in ~/.ssh with the connections that use each; Adopt copies a key into the vault and repoints connections at it; Generate opens a local tab with `ssh-keygen` typed but not run, so passphrase prompts stay in the terminal (no passphrase ever passes through argv).

File browser (D3, done): a Files dialog (folder icon in the tab bar, Shell menu, palette) for the focused saved connection. It lists folders (path bar, up, refresh, hidden toggle), uploads, downloads (native dialogs), makes folders, renames and deletes (confirm; only empty folders). It shells out to the system `sftp` and reuses the terminal's authenticated connection through OpenSSH multiplexing: profile sessions now start with `ControlMaster=auto` (`ControlPersist=60s`, socket under `/tmp/relay-ssh-<uid>`), and file operations attach with `ControlMaster=no`, so passwords, passphrases and 2FA are never needed or seen here. Safety: batch paths are quoted and anything containing a control character (a newline would inject another sftp command) is refused; a rejected login or an untrusted or changed host key produces a plain-language message. Verified against a real throwaway `sshd` (list, upload, download, mkdir, rename, delete, auth failure, injection attempt) and in the dev window.

Not done: Windows (OpenSSH there has no multiplexing, so file operations would need their own login), recursive delete, drag and drop, transfer progress.

Port forwards (done): a Port forwards dialog (tab bar, Shell menu, palette) for the focused saved connection. It shows whether the connection is open and every forward on it: the saved connection's tunnels plus any added here, with a live state dot (local and SOCKS ports are probed; remote ones are held by the server, so a stop made here is remembered). Add local, remote or SOCKS forwards, Stop or Start any of them, and Save a forward you added on the fly into the connection so it starts with every login. It drives OpenSSH multiplexing (`ssh -O check`, `-O forward`, `-O cancel`) against the master the terminal tab started; no tunnelling code lives in Relay. Notes: ssh cannot list a master's forwards, so runtime ones are tracked in memory (lost on restart of the app, though still active on the master); local and SOCKS ports are checked free before asking ssh, because ssh reports success if it can bind only `::1` and `127.0.0.1` is taken by another program; ports and targets are validated against a strict pattern so nothing can smuggle in extra ssh options. Verified against a real throwaway `sshd` with real traffic through local, SOCKS and remote forwards, and in the dev window against a live master. A deadlock found by those tests (a lock held across a probe) is fixed.

Phase 4 is complete except for Windows, where OpenSSH has no connection sharing (file transfers and live forwards would need their own login).

## 9g. Data model change: explicit folders (2026-10-29)

`connections.toml` gains an optional top-level `folders = ["archive/old", ...]` listing folders that exist even when empty, so the sidebar can create a folder or subfolder before anything is in it. Folders holding connections exist without being listed, and every ancestor of a folder exists ("a/b" implies "a"). Files without the key load unchanged; the list is sorted and de-duplicated on save; each entry is validated (non-empty, no leading or trailing slash, no empty segment). A folder emptied by moving, editing or deleting its last connection is kept (added to the list), because folders are user-managed. Operations (desktop `ProfilesService`, outside the CLI line budget): create folder, rename or move folder (everything beneath follows; refuses moving into itself or onto an existing folder), delete folder (refuses one holding connections and names them; removes empty subfolders), move connection. The CLI is unaffected: `ls --tree` still reads each connection's `folder`.

## 9h. Light theme (2026-10-29)

Themes are plain data in `themes.ts` (UI variables, xterm palette, search-highlight colours) instead of values scattered through components; `settings.apply()` writes them to `:root` and sets `color-scheme`. A fourth theme, **Daylight**, is light, and **System** follows the OS (`prefers-color-scheme`, live) by resolving to Daylight or Midnight. Every colour a component used to hard-code (whites, scrims, shadows, accent and error tints, status stripes, block bands) is now a variable, derived from each theme's accent, ok and err colours where possible, so a new theme is one entry. A unit test checks the contrast of every theme (text and muted text on each surface, accent and error text, button labels, terminal foreground, the sixteen ANSI colours, and that xterm's background matches the page). Verified in the browser pane against every view and in the real window (CSS variables, `color-scheme` and xterm background change for Daylight and System). Not verified: the native window frame and shadow in the light theme. Also fixed: generated `.svelte-check` files were tracked in git and are now ignored.

## 9i. Desktop CI (2026-10-29)

`.github/workflows/desktop.yml` runs `make desktop-check` (desktop guardrails, typecheck, frontend tests and build, then Go vet and tests, including the real-shell and throwaway-`sshd` integration tests) on a macOS runner for pushes to main and pull requests. It runs only when `desktop/`, `internal/`, the root Go files, the Makefile or the workflow change, and a newer push cancels the older run, because macOS minutes count ten times against a private repository's free allowance. The existing `ci.yml` keeps running the CLI gate on Linux and macOS. Both workflows now request read-only `contents` permission. The Makefile target now builds the frontend before the Go checks: the app embeds `frontend/dist`, which is not tracked, so on a fresh checkout `go vet` failed before. The pre-push hook stays as the early local check. Windows and Linux runners wait for those builds to exist.

## 9j. Connecting state and a sidebar crash (2026-10-29)

Opening a saved connection used to show a blank terminal marked "Connected" the moment the local socket opened, while ssh was still dialling (an unreachable host stays silent for over a minute). A pane now has a `connecting` state, from start until ssh first writes anything (a host-key question, a password prompt, a banner, an error or the shell): a loading screen shows the connection, its user@host:port and jump host, the elapsed time, a hint after 8 s (slow) and 30 s (may be unreachable, offline or firewalled), and a Cancel button. The status bar, the tab dot and the split-pane header show amber "Connecting…". Go still reads no terminal bytes; "ssh has spoken" is the signal. Local shells skip it.

Found while testing this in the real window: the sidebar's "Open now" list rendered rows at depth -1 and `Array(-1)` throws, so the moment any connection was opened the sidebar failed and the UI stopped updating. Fixed (depth is clamped), with a component test; the smoke run now records uncaught JavaScript errors and drives a slow (unroutable) connection through connecting, hint and cancel. Component tests need Svelte's browser build, so tests have their own `vitest.config.ts`.

## 9k. Sidebar and top bar changes (2026-10-29)

The connection filter moved from the sidebar into the top bar (a field left of the tools; its ⌘K badge still opens quick open, Escape or the × clears it, and typing reveals the sidebar if it was hidden). The sidebar no longer has "Folders" and "Unfiled" section bars: connections that are in no folder sit at the top of the list, outside any folder, with the folder tree below them ("Open now" stays above both). The Add menu was positioned with a fixed 300px width in a 268px sidebar and so was cut off on the left; it now fills the sidebar width.

## 9l. Drag fix and split buttons (2026-10-29)

Dragging connections and folders in the sidebar never started with a real mouse: the drag handler ignored presses on any button, and a row's content is a button. The smoke test pressed the row element itself, so it passed. Now the row's name button is the handle and only the caret, the edit and menu buttons, inputs and open menus opt out (`data-nodrag`); the unit tests press the inner buttons and the smoke drives them too. Splitting panes had only a menu item and shortcuts (Shell menu, ⌘D right, ⇧⌘D down, quick-open actions), which nobody could find: the top bar now has Split right and Split down buttons. Then built: **tabs reorder by dragging** (an accent line shows the gap; Escape cancels), and **a connection dragged from the sidebar onto a terminal** splits that pane toward the nearest edge (left, right, top or bottom, with a highlighted half and a "Split left" style label) or, dropped in the middle, opens in a new tab. It uses the sidebar's pointer-based drag (no native drag and drop), so folder moves and pane drops share one gesture and a connection over a pane never asks for a folder-move confirmation. Pure helpers (`reorder`, `slotAt`, `paneEdge`, `splitLeaf(..., before)`) are unit tested; the smoke run drives both drags in the real window. Not built: dragging a pane or a tab into another tab's layout, and rearranging existing panes.

## 9m. Opening, pane dragging and remote blocks (2026-10-29)

- **"Open now" removed** from the sidebar: the status dot on each row already shows which connections are open.
- **Double-click opens.** A single click only selects a row; a double-click (or Enter on the focused row) opens a new tab. It is taught three ways: the row tooltip, a one-line tip under the list (until the first double-click is made, remembered in local storage), and the empty-window text.
- **Panes are draggable by their header** (shown when a tab has more than one pane): dropped on another pane's edge it moves beside it on that side, dropped in the middle it swaps places with it. The session keeps running (the terminal element is re-parented, not recreated; a reused leaf component re-adopts the right element). Same pointer-based drag as the sidebar and tabs, with a ghost and a highlighted half; the ghost now lives in its own component so it shows even when the sidebar is hidden. Pure helpers `movePane` and `swapLeaves` are unit tested.
- **Why command blocks do not appear over ssh by themselves:** blocks come from markers the shell prints. For local shells Relay injects the hooks. A remote shell is not Relay's to configure: the only way is to type a setup line into it, and typing before the login finishes could land in a password prompt, so Relay never does it unasked. That opt-in was only reachable through the Shell menu or quick open. It is now offered: a pill in the pane a moment after the login (Enable, or × to stop being asked) and an "Enable command blocks" button in the status bar for every remote pane that does not have them. Works for bash and zsh on the remote; the setup line is not saved on the host.

## 9n. Automatic command blocks over ssh (2026-10-29)

**Setting.** `Settings > Behaviour > "Command blocks over SSH"`: *Ask* (default: the pill and the status bar button of 9m) or *Automatic*. Each connection can override it: a new optional profile field `blocks` in `connections.toml`, `"on"` or `"off"` (absent means follow the app setting), set in the connection form as "Command blocks: App default / On / Off". It is a profile property because it belongs to the host (a router with no bash should say off), travels with backups and is repairable in an editor. The CLI ignores it. **Data model and budget:** the field and its check cost 6 non-test Go lines in `internal/config`, so the CLI budget goes 3000 to 3010 (CLAUDE.md and `scripts/guardrails.sh` record it).

**When is it safe to type the setup line?** Only after the login has finished: before that the keystrokes could land in a password, passphrase or host-key prompt. Go still reads no terminal bytes. Two signals say "authenticated":
- a new login runs ssh with `PermitLocalCommand=yes` and a `LocalCommand` that prints a private escape sequence (OSC 7771) to the terminal; ssh runs it only after authentication succeeds, and the frontend's parser swallows it (it is added to the terminal's own ssh only, not the generated config, so jump-host child processes never run it);
- a session that attaches to an already-open login (ControlMaster) never authenticates, so LocalCommand does not run; Go sees the live master (`ssh -O check`) before starting and sends the page a `mux` control frame instead.

Once authenticated, the frontend waits for 700 ms without output, then types the same line the button types (at most once per session, only if blocks are not already on, and never for a local shell). The line is only buffered by the remote tty if the shell is not ready, so timing is not critical. If the remote shell is not bash or zsh the line prints a one-line notice and does nothing.

Done as specified above. Verified against a real throwaway `sshd` in a Go test (the marker appears for a new login and not for a session sharing one; `masterAlive` tracks the master) and in the real window both ways (no open login: marker; open login: `mux` frame), each ending with blocks on and no pill. A login through a jump host prints the marker exactly once and still works.

## 9o. Snippets and dragging across tabs (2026-10-29)

**Snippets.** Saved commands (a name and a command, possibly several lines) kept in `~/.config/sssh/snippets.toml`, a file next to `connections.toml` that is read and written only by the desktop app (`SnippetsService`; the CLI budget is untouched). Atomic writes at 0600; a malformed file is reported rather than treated as empty, because the next save would overwrite it; names are unique, at most 80 characters, no control characters; commands at most 20,000 characters and no NUL. The Snippets dialog (top bar, Shell menu, quick open, ⇧⌘S) lists, searches, adds, edits and deletes them, and each row has **Insert** (types it into the focused pane without Enter, to edit first) and **Run** (types it and presses Enter). Typing goes through xterm's paste, so a shell that asks for bracketed paste receives a multi-line snippet as one paste rather than line by line, and Run's Enter goes through the terminal's input so command blocks see it. Not included yet: snippets are not part of backups (the backup archive is built in `internal/core`, under the CLI line budget), and there are no placeholders or variables.

**Dragging across tabs.** A pane dragged by its header onto another tab in the top bar moves there (beside that tab's focused pane; the tab highlights), and onto the + button it becomes a tab of its own. A tab dragged out of the strip onto a terminal that belongs to another tab folds all of its panes in on the side nearest the pointer ("Add tab right"). Sessions keep running through every move. Pure helper `insertBeside` is unit tested; the smoke run drives all three in the real window. Side fix found there: tabs had a fixed minimum width, so past a dozen of them the later tabs were pushed out of reach; they now shrink.

## 10. Working agreement (carried over from the CLI)

TDD (watch the test fail first), hermetic tests using `t.TempDir()` with no real `$HOME` and no network, atomic writes, conventional commits per task, spec before code for any new command or data model change, and `make check` before claiming anything is done. Do not report work as complete on the strength of having written it.
