import { describe, expect, it } from "vitest";
import page from "../index.html?raw";

// The IronRDP client is WebAssembly, which the browser refuses to compile unless the page's policy says so,
// and it fetches its own module from a data: URL. Everything else stays locked to the app's own files.
describe("the page's Content-Security-Policy", () => {
  const csp = /Content-Security-Policy" content="([^"]*)"/.exec(page)![1];
  const directive = (name: string) => csp.split(";").map((d) => d.trim()).find((d) => d.startsWith(name + " "))!.split(/\s+/).slice(1);

  it("lets WebAssembly compile, but still no inline or eval'd script", () => {
    expect(directive("script-src")).toEqual(["'self'", "'wasm-unsafe-eval'"]);
  });
  it("lets the client run its own worker from a blob: URL (and no other source)", () => {
    expect(directive("worker-src")).toEqual(["'self'", "blob:"]);
  });
  it("lets the client load its inlined module and reach only the loopback gateway", () => {
    const connect = directive("connect-src");
    expect(connect).toContain("data:");
    expect(connect.filter((c) => /^(ws|http)s?:/.test(c)).every((c) => /(127\.0\.0\.1|localhost)/.test(c))).toBe(true);
  });
});
