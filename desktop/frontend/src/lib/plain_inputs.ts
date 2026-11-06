// Host names, users, paths, ports and commands are not prose, but the system treats text fields as if they were: it
// capitalises the first letter, "corrects" words and underlines them. WebKit honours these attributes to stop that.
// Fields are marked as they appear (dialogs open and close all the time); a field that sets its own is left alone.
const PLAIN = { autocapitalize: "off", autocorrect: "off", spellcheck: "false" } as const;
const SELECTOR = "input, textarea";
const NOT_TEXT = new Set(["checkbox", "radio", "button", "submit", "reset", "file", "range", "color", "image", "hidden"]);

function mark(el: Element) {
  if (el instanceof HTMLInputElement && NOT_TEXT.has(el.type)) return;
  for (const [name, value] of Object.entries(PLAIN)) if (!el.hasAttribute(name)) el.setAttribute(name, value);
}

function markAll(root: ParentNode) {
  if (root instanceof Element && root.matches(SELECTOR)) mark(root);
  root.querySelectorAll?.(SELECTOR).forEach(mark);
}

/** Marks every text field in the page, now and as they are added. Returns a function that stops watching. */
export function installPlainInputs(root: Document = document): () => void {
  markAll(root);
  const observer = new MutationObserver((records) => {
    for (const r of records) r.addedNodes.forEach((n) => n instanceof Element && markAll(n));
  });
  observer.observe(root, { childList: true, subtree: true });
  return () => observer.disconnect();
}
