export type ToastKind = "info" | "success" | "error";

export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastOptions {
  description?: string;
  action?: ToastAction;
  duration?: number;
}

export interface ToastItem {
  id: number;
  kind: ToastKind;
  message: string;
  description?: string;
  action?: ToastAction;
}

const DEFAULT_DURATION = 4000;
const ACTION_DURATION = 8000;
const MAX_VISIBLE = 5;

class ToastStore {
  items = $state<ToastItem[]>([]);
  private nextId = 1;
  private timers = new Map<number, ReturnType<typeof setTimeout>>();

  push(kind: ToastKind, message: string, opts: ToastOptions = {}): number {
    const id = this.nextId++;
    const item: ToastItem = {
      id,
      kind,
      message,
      description: opts.description,
      action: opts.action,
    };
    const next = [...this.items, item];
    while (next.length > MAX_VISIBLE) {
      const dropped = next.shift();
      if (dropped) this.clearTimer(dropped.id);
    }
    this.items = next;

    const ms = opts.duration ?? (opts.action ? ACTION_DURATION : DEFAULT_DURATION);
    if (ms > 0) {
      this.timers.set(
        id,
        setTimeout(() => {
          this.dismiss(id);
        }, ms),
      );
    }
    return id;
  }

  dismiss(id?: number) {
    if (id === undefined) {
      this.clear();
      return;
    }
    this.clearTimer(id);
    this.items = this.items.filter((t) => t.id !== id);
  }

  clear() {
    for (const timer of this.timers.values()) clearTimeout(timer);
    this.timers.clear();
    this.items = [];
  }

  private clearTimer(id: number) {
    const timer = this.timers.get(id);
    if (timer) {
      clearTimeout(timer);
      this.timers.delete(id);
    }
  }
}

export const toasts = new ToastStore();

type ToastFn = ((message: string, opts?: ToastOptions) => number) & {
  success: (message: string, opts?: ToastOptions) => number;
  error: (message: string, opts?: ToastOptions) => number;
  dismiss: (id?: number) => void;
  clear: () => void;
};

export const toast: ToastFn = Object.assign(
  (message: string, opts?: ToastOptions) => toasts.push("info", message, opts),
  {
    success: (message: string, opts?: ToastOptions) => toasts.push("success", message, opts),
    error: (message: string, opts?: ToastOptions) => toasts.push("error", message, opts),
    dismiss: (id?: number) => toasts.dismiss(id),
    clear: () => toasts.clear(),
  },
);
