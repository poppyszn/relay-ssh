<script lang="ts">
  // Relay's mark: a terminal chevron cut out of a block, with a lighter block behind it.
  // The geometry is the icon's 2000-unit grid (see desktop/build/icon/make-icon.swift).
  let { size = 64 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="500 500 1000 1000" role="img" aria-label="Relay">
  <path fill="#318EDC" fill-rule="evenodd" d="M726 500H1250V1500H750V1250H500V726Z M750 750H904L1156 1001L904 1250H750L1001 1001Z" />
  <polygon fill="#7AB2EE" points="1250,750 1500,750 1500,1250 1250,1500" />
</svg>
