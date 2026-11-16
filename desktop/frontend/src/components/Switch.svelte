<script lang="ts">
  import type { HTMLButtonAttributes } from "svelte/elements";

  let {
    checked = $bindable(false),
    disabled = false,
    class: className = "switch",
    ...rest
  }: Omit<HTMLButtonAttributes, "type" | "role" | "aria-checked"> & {
    checked?: boolean;
    disabled?: boolean;
    class?: string;
  } = $props();

  function toggle() {
    if (disabled) return;
    checked = !checked;
  }

  function onkeydown(e: KeyboardEvent) {
    if (disabled) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      if (!e.repeat) toggle();
    }
  }

  function onkeyup(e: KeyboardEvent) {
    if (e.key === " ") e.preventDefault();
  }
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked ? "true" : "false"}
  data-state={checked ? "checked" : "unchecked"}
  {disabled}
  class={className}
  onclick={toggle}
  {onkeydown}
  {onkeyup}
  {...rest}
></button>
