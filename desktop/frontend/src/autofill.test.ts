// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { parsePasswordPrompt } from "./lib";

describe("frontend prompt detection and autofill handling", () => {
  it("detects account prompt and builds fill control frame", () => {
    const line = "alice@10.0.0.5's password: ";
    const prompt = parsePasswordPrompt(line);
    expect(prompt).toEqual({ kind: "account", user: "alice", host: "10.0.0.5" });

    const frame = JSON.stringify({ t: "fill", ...prompt });
    expect(JSON.parse(frame)).toEqual({
      t: "fill",
      kind: "account",
      user: "alice",
      host: "10.0.0.5",
    });
  });

  it("detects generic prompt and builds generic fill control frame", () => {
    const line = "(alice@host) Password: ";
    const prompt = parsePasswordPrompt(line);
    expect(prompt).toEqual({ kind: "generic" });

    const frame = JSON.stringify({ t: "fill", ...prompt });
    expect(JSON.parse(frame)).toEqual({
      t: "fill",
      kind: "generic",
    });
  });

  it("builds authed control frame on successful authentication", () => {
    const frame = JSON.stringify({ t: "authed" });
    expect(JSON.parse(frame)).toEqual({ t: "authed" });
  });

  it("does not build fill frame for non-password prompts", () => {
    const prompts = [
      "Enter passphrase for key '/Users/user/.ssh/id_ed25519': ",
      "Verification code: ",
      "[sudo] password for alice: ",
      "Are you sure you want to continue connecting (yes/no/[fingerprint])? ",
    ];
    for (const p of prompts) {
      expect(parsePasswordPrompt(p)).toBeNull();
    }
  });
});
