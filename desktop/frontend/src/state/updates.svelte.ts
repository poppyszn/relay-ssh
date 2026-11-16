import { UpdateService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

export type UpdatePhase = "idle" | "checking" | "uptodate" | "available" | "installing" | "ready" | "error";

const msg = (e: unknown) => (e as Error)?.message ?? String(e);

/** The state of app updates, shared by Settings and the launch-time check. */
export class Updates {
  current = $state("");
  /** False for local builds, which never update themselves. */
  enabled = $state(false);
  phase = $state<UpdatePhase>("idle");
  latest = $state("");
  notes = $state("");
  error = $state("");

  async load() {
    try {
      const i = await UpdateService.Info();
      this.current = i.current;
      this.enabled = i.enabled;
    } catch {
      this.enabled = false;
    }
  }

  async check() {
    if (this.phase === "checking" || this.phase === "installing") return;
    this.phase = "checking";
    this.error = "";
    try {
      const i = await UpdateService.Check();
      this.current = i.current;
      this.latest = i.latest;
      this.notes = i.notes;
      this.phase = i.available ? "available" : "uptodate";
    } catch (e) {
      this.error = msg(e);
      this.phase = "error";
    }
  }

  /** The check made shortly after launch. True only when there is a newer release to tell the user about. */
  async checkOnLaunch(): Promise<boolean> {
    await this.load();
    if (!this.enabled) return false;
    await this.check();
    return this.phase === "available";
  }

  async install() {
    if (!this.latest || (this.phase !== "available" && this.phase !== "error")) return;
    this.phase = "installing";
    this.error = "";
    try {
      await UpdateService.Install();
      this.phase = "ready";
    } catch (e) {
      this.error = msg(e);
      this.phase = "error";
    }
  }

  async restart() {
    try {
      await UpdateService.Restart();
    } catch (e) {
      this.error = msg(e);
      this.phase = "error";
    }
  }
}

export const updates = new Updates();
