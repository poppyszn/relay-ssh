// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { buildConfig, loadRdpClient, mountClient, needsResize, type RdpClientModule } from "./rdp_client";

// A stand-in for the IronRDP web component: a custom element that announces itself with `ready`.
function fakeElementClass(ui: object) {
  return class extends HTMLElement {
    module: unknown;
    connectedCallback() {
      queueMicrotask(() => this.dispatchEvent(new CustomEvent("ready", { detail: { irgUserInteraction: ui } })));
    }
  };
}

describe("loading the client", () => {
  it("is null when the client was not built into this copy of the app", async () => {
    expect(await loadRdpClient({})).toBeNull();
    expect(await loadRdpClient({ "../../vendor/ironrdp/iron-remote-desktop.js": async () => ({}) })).toBeNull();
  });

  it("registers the web component, initialises the RDP backend once and returns it", async () => {
    const component = vi.fn(async () => ({}));
    const init = vi.fn(async () => {});
    const backend = { Backend: {}, displayControl: () => ({}), init };
    const files = { "../../vendor/ironrdp/iron-remote-desktop.js": component, "../../vendor/ironrdp/iron-remote-desktop-rdp.js": async () => backend };
    const client = await loadRdpClient(files);
    expect(client).toBe(backend);
    expect(component).toHaveBeenCalledOnce();
    expect(init).toHaveBeenCalledOnce();
  });
});

describe("mounting the web component", () => {
  it("hands the backend to the element and resolves with its user-interaction API once ready", async () => {
    const ui = { marker: "ui" };
    customElements.define("iron-remote-desktop", fakeElementClass(ui));
    const host = document.createElement("div");
    document.body.append(host); // custom elements start only once they are in the document
    const client = { Backend: { id: "backend" } } as unknown as RdpClientModule;
    const got = await mountClient(host, client);
    expect(got).toBe(ui);
    const el = host.querySelector("iron-remote-desktop") as unknown as { module: unknown; getAttribute(n: string): string | null };
    expect(el.module).toEqual({ id: "backend" });
    expect(el.getAttribute("scale")).toBe("real");
  });
});

describe("the connection config", () => {
  it("points the client at our gateway with the session token in the URL, and the exact destination the gateway registered", () => {
    const calls: [string, unknown][] = [];
    const builder: Record<string, unknown> = {};
    for (const m of ["withUsername", "withPassword", "withDestination", "withProxyAddress", "withServerDomain", "withAuthToken", "withDesktopSize", "withExtension"]) {
      builder[m] = (v: unknown) => (calls.push([m, v]), builder);
    }
    builder.build = () => ({ built: true });
    const ui = { configBuilder: () => builder } as never;
    const client = { displayControl: (on: boolean) => ({ displayControl: on }) } as unknown as RdpClientModule;

    const cfg = buildConfig(ui, client, { username: "Administrator", password: "pw", domain: "CORP", host: "10.0.2.50", port: 3389, wsUrl: "ws://127.0.0.1:5555/rdp", token: "tok", width: 1440, height: 900 });

    expect(cfg).toEqual({ built: true });
    const by = Object.fromEntries(calls);
    expect(by.withDestination).toBe("10.0.2.50:3389");
    expect(by.withProxyAddress).toBe("ws://127.0.0.1:5555/rdp?token=tok");
    expect(by.withAuthToken).toBe("tok");
    expect(by.withUsername).toBe("Administrator");
    expect(by.withPassword).toBe("pw");
    expect(by.withServerDomain).toBe("CORP");
    expect(by.withDesktopSize).toEqual({ width: 1440, height: 900 });
    expect(by.withExtension).toEqual({ displayControl: true });
  });
});

describe("following the tab's size", () => {
  it("only asks the server to resize when the size really changed", () => {
    expect(needsResize({ width: 1000, height: 640 }, { width: 1000, height: 640 })).toBe(false);
    expect(needsResize({ width: 1000, height: 640 }, { width: 1000.4, height: 640.9 })).toBe(false); // sub-pixel noise
    expect(needsResize({ width: 1000, height: 640 }, { width: 1001, height: 640 })).toBe(true);
    expect(needsResize({ width: 1000, height: 640 }, { width: 900, height: 600 })).toBe(true);
  });
  it("never asks for a size the server would reject", () => {
    expect(needsResize({ width: 1000, height: 640 }, { width: 0, height: 600 })).toBe(false);
    expect(needsResize({ width: 1000, height: 640 }, { width: 600, height: 0 })).toBe(false);
  });
});
