// @vitest-environment happy-dom
import { afterEach, describe, expect, it } from "vitest";
import { installPlainInputs } from "./plain_inputs";

let stop: (() => void) | null = null;
afterEach(() => { stop?.(); stop = null; document.body.innerHTML = ""; });
const flush = () => new Promise((r) => setTimeout(r, 0));

// Host names, users, paths and commands are not prose: the system must not capitalise, correct or spell-check them.
describe("plain text inputs", () => {
  it("turns off auto-capitalisation, auto-correction and spell check on the fields already there", () => {
    document.body.innerHTML = '<input id="a" type="text"><textarea id="b"></textarea><input id="c" type="search"><input id="p" type="password">';
    stop = installPlainInputs();
    for (const id of ["a", "b", "c", "p"]) {
      const el = document.getElementById(id)!;
      expect(el.getAttribute("autocapitalize"), id).toBe("off");
      expect(el.getAttribute("autocorrect"), id).toBe("off");
      expect(el.getAttribute("spellcheck"), id).toBe("false");
    }
  });

  it("covers fields added later, such as a dialog that opens", async () => {
    stop = installPlainInputs();
    const wrap = document.createElement("div");
    wrap.innerHTML = '<form><label><input id="late" type="text"></label></form>';
    document.body.append(wrap);
    await flush();
    expect(document.getElementById("late")!.getAttribute("autocapitalize")).toBe("off");
  });

  it("leaves checkboxes, buttons and fields that chose their own setting alone", async () => {
    document.body.innerHTML = '<input id="x" type="checkbox"><input id="y" type="button"><input id="own" type="text" autocapitalize="sentences" spellcheck="true">';
    stop = installPlainInputs();
    expect(document.getElementById("x")!.hasAttribute("autocapitalize")).toBe(false);
    expect(document.getElementById("y")!.hasAttribute("autocapitalize")).toBe(false);
    expect(document.getElementById("own")!.getAttribute("autocapitalize")).toBe("sentences");
    expect(document.getElementById("own")!.getAttribute("spellcheck")).toBe("true");
  });

  it("stops watching when told to", async () => {
    stop = installPlainInputs();
    stop();
    stop = null;
    document.body.insertAdjacentHTML("beforeend", '<input id="after" type="text">');
    await flush();
    expect(document.getElementById("after")!.hasAttribute("autocapitalize")).toBe(false);
  });
});
