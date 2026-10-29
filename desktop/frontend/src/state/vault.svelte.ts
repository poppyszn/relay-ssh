import { VaultService, type VaultStatus } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

const REMEMBERED_KEY = "relay.vault.remembered.v1";

function loadRemembered(): string[] {
  try {
    return JSON.parse(localStorage.getItem(REMEMBERED_KEY) ?? "[]");
  } catch {
    return [];
  }
}

function saveRemembered(list: string[]) {
  try {
    localStorage.setItem(REMEMBERED_KEY, JSON.stringify(list));
  } catch {}
}

class VaultStore {
  exists = $state(false);
  unlocked = $state(false);
  remembered = $state<string[]>(loadRemembered());

  async refresh(): Promise<VaultStatus> {
    try {
      const st = await VaultService.Status();
      this.exists = st.exists;
      this.unlocked = st.unlocked;
      return st;
    } catch {
      return { exists: false, unlocked: false };
    }
  }

  hasRemembered(name: string): boolean {
    return this.remembered.includes(name);
  }

  remember(name: string) {
    if (!this.remembered.includes(name)) {
      this.remembered = [...this.remembered, name];
      saveRemembered(this.remembered);
    }
  }

  forget(name: string) {
    this.remembered = this.remembered.filter((n) => n !== name);
    saveRemembered(this.remembered);
  }

  rename(oldName: string, newName: string) {
    if (this.remembered.includes(oldName)) {
      this.remembered = this.remembered.map((n) => (n === oldName ? newName : n));
      saveRemembered(this.remembered);
    }
  }

  async lock() {
    await VaultService.Lock();
    await this.refresh();
  }

  async reset(confirm: string) {
    await VaultService.Reset(confirm);
    this.remembered = [];
    saveRemembered(this.remembered);
    await this.refresh();
  }
}

export const vault = new VaultStore();
