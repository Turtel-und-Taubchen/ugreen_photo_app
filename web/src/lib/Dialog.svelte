<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    onClose,
    children,
    actions,
    wide = false,
  }: {
    title: string;
    onClose: () => void;
    children: Snippet;
    /** Receives close() so a cancel button goes through the dialog's own close event. */
    actions: Snippet<[() => void]>;
    wide?: boolean;
  } = $props();

  const titleId = $props.id();
  let dialog: HTMLDialogElement;

  // Native modal: focus trap, Escape and inert background come from the browser.
  function modal(node: HTMLDialogElement) {
    dialog = node;
    node.showModal();
  }
</script>

<dialog use:modal aria-labelledby={titleId} onclose={onClose} class:wide>
  <h2 id={titleId}>{title}</h2>
  {@render children()}
  <div class="actions">
    {@render actions(() => dialog.close())}
  </div>
</dialog>

<style>
  dialog { width: min(32rem, calc(100vw - 2rem)); box-sizing: border-box; max-height: calc(100vh - 2rem); overflow-y: auto; }
  dialog.wide { width: min(40rem, calc(100vw - 2rem)); }
  h2 { margin-top: 0; }
  .actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
  .actions :global(button) { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
