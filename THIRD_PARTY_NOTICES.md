# Third-party notices

Relay and `relay-ssh` are released under the [MIT License](LICENSE). They build on the
open-source software below, each under its own license. This list covers direct
dependencies and everything bundled into the desktop app; transitive dependencies
carry their own licenses, which their packages include.

## Go (the CLI and the desktop backend)

| Module | License |
| --- | --- |
| [filippo.io/age](https://github.com/FiloSottile/age) | BSD-3-Clause |
| [github.com/BurntSushi/toml](https://github.com/BurntSushi/toml) | MIT |
| [github.com/wailsapp/wails/v3](https://github.com/wailsapp/wails) (desktop only) | MIT |
| [github.com/coder/websocket](https://github.com/coder/websocket) (desktop only) | ISC |
| [github.com/creack/pty](https://github.com/creack/pty) (desktop only) | MIT |

## Desktop frontend (bundled into the app)

| Package | License |
| --- | --- |
| [@xterm/xterm](https://github.com/xtermjs/xterm.js) and its fit, search and WebGL addons | MIT |
| [@wailsio/runtime](https://github.com/wailsapp/wails) | MIT |
| [Svelte](https://github.com/sveltejs/svelte) | MIT |

## Fonts (bundled into the desktop app)

The fonts are licensed under the SIL Open Font License 1.1, which permits
bundling and redistribution provided the license text accompanies them. Both
texts are in [`licenses/`](licenses/).

| Font | Source | License text |
| --- | --- | --- |
| JetBrains Mono | [JetBrains/JetBrainsMono](https://github.com/JetBrains/JetBrainsMono), packaged by [Fontsource](https://fontsource.org) | [`licenses/OFL-JetBrainsMono.txt`](licenses/OFL-JetBrainsMono.txt) |
| Inter | [rsms/inter](https://github.com/rsms/inter), packaged by [Fontsource](https://fontsource.org) | [`licenses/OFL-Inter.txt`](licenses/OFL-Inter.txt) |

## Tools used to build, not shipped

Vite, TypeScript, Vitest and svelte-check (MIT/Apache-2.0) are used during
development only.
