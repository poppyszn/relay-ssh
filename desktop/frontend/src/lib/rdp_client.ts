// The remote desktop client is IronRDP's official browser client (a WebAssembly module, its web component
// and the RDP backend), built by `make rdp-client` into frontend/vendor/ironrdp/ and not committed.
// Everything here tolerates its absence: a copy of the app built without it simply has no RDP.

export interface RdpUi {
  configBuilder(): ConfigBuilder;
  connect(config: unknown): Promise<{ run(): Promise<{ reason(): string }> }>;
  setVisibility(visible: boolean): void;
  setEnableClipboard(enable: boolean): void;
  setEnableAutoClipboard(enable: boolean): void;
  resize(width: number, height: number, scale?: number): void;
  ctrlAltDel(): void;
  shutdown(): void;
}

interface ConfigBuilder {
  withUsername(v: string): ConfigBuilder;
  withPassword(v: string): ConfigBuilder;
  withDestination(v: string): ConfigBuilder;
  withProxyAddress(v: string): ConfigBuilder;
  withServerDomain(v: string): ConfigBuilder;
  withAuthToken(v: string): ConfigBuilder;
  withDesktopSize(v: { width: number; height: number }): ConfigBuilder;
  withExtension(v: unknown): ConfigBuilder;
  build(): unknown;
}

export interface RdpClientModule {
  Backend: unknown;
  displayControl(enable: boolean): unknown;
  init(logLevel: string): Promise<void>;
}

type Loaders = Record<string, () => Promise<unknown>>;

const COMPONENT = "../../vendor/ironrdp/iron-remote-desktop.js";
const BACKEND = "../../vendor/ironrdp/iron-remote-desktop-rdp.js";
// A glob (not a plain import) so a missing build is "no files" rather than a build error.
const built: Loaders = import.meta.glob("../../vendor/ironrdp/*.js");

let loaded: Promise<RdpClientModule | null> | null = null;

/** Loads and initialises the client once, or null when this build of the app does not include it. */
export function loadRdpClient(files: Loaders = built): Promise<RdpClientModule | null> {
  const load = async () => {
    if (!files[COMPONENT] || !files[BACKEND]) return null;
    await files[COMPONENT](); // defines <iron-remote-desktop>
    const backend = (await files[BACKEND]()) as RdpClientModule;
    await backend.init("ERROR");
    return backend;
  };
  if (files !== built) return load();
  return (loaded ??= load().catch((e) => ((loaded = null), Promise.reject(e))));
}

/** Puts an <iron-remote-desktop> element in `host` and resolves with its API once it announces itself. */
export function mountClient(host: HTMLElement, client: RdpClientModule): Promise<RdpUi> {
  return new Promise((resolve, reject) => {
    const el = document.createElement("iron-remote-desktop") as HTMLElement & { module?: unknown };
    el.setAttribute("scale", "real"); // the remote desktop is resized to the tab, so show it 1:1
    el.setAttribute("flexcenter", "true");
    el.module = client.Backend;
    const timer = setTimeout(() => reject(new Error("The remote desktop client did not start")), 15000);
    el.addEventListener("ready", (e) => {
      clearTimeout(timer);
      resolve((e as CustomEvent).detail.irgUserInteraction as RdpUi);
    });
    host.append(el);
  });
}

export interface RdpConnect {
  username: string;
  password: string;
  domain: string;
  host: string;
  port: number;
  wsUrl: string;
  token: string;
  width: number;
  height: number;
}

/** The connection settings: our loopback gateway (with the session token in its URL) in front of the host. */
export function buildConfig(ui: RdpUi, client: RdpClientModule, o: RdpConnect): unknown {
  return ui
    .configBuilder()
    .withUsername(o.username)
    .withPassword(o.password)
    .withServerDomain(o.domain)
    // Must be exactly what the gateway registered for this session, or it refuses the connection.
    .withDestination(`${o.host}:${o.port}`)
    .withProxyAddress(`${o.wsUrl}?token=${o.token}`)
    .withAuthToken(o.token)
    .withDesktopSize({ width: o.width, height: o.height })
    .withExtension(client.displayControl(true)) // lets the remote desktop follow the tab's size
    .build();
}

export interface Size {
  width: number;
  height: number;
}

/** Whether the remote desktop should be resized: the whole-pixel size changed, and is a usable size. */
export function needsResize(applied: Size, next: Size): boolean {
  const w = Math.floor(next.width), h = Math.floor(next.height);
  if (w <= 0 || h <= 0) return false;
  return w !== Math.floor(applied.width) || h !== Math.floor(applied.height);
}
