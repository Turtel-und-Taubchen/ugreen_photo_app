<script lang="ts">
  import { errorMessage } from './api';
  import { createJob } from './jobs';
  import { toRequest } from './selection';
  import { selection } from './selectionStore.svelte';
  import TagListInput from './TagListInput.svelte';

  let { onClose }: { onClose: () => void } = $props();

  let dialog: HTMLDialogElement;
  let tags = $state<string[]>([]);
  let error = $state('');
  let busy = $state(false);

  // Native modal: focus trap, Escape and inert background come from the browser.
  function modal(node: HTMLDialogElement) {
    dialog = node;
    node.showModal();
  }

  async function assign() {
    error = '';
    busy = true;
    try {
      await createJob(toRequest(selection.state), 'tags.add', { tags });
      onClose();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }
</script>

<dialog use:modal aria-labelledby="assign-title" onclose={onClose}>
  <h2 id="assign-title">Tags zuweisen</h2>
  <TagListInput bind:tags />
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  <div class="actions">
    <button type="button" onclick={() => dialog.close()}>Abbrechen</button>
    <button type="button" disabled={tags.length === 0 || busy} onclick={assign}>Zuweisen</button>
  </div>
</dialog>

<style>
  dialog { width: min(32rem, calc(100vw - 2rem)); box-sizing: border-box; }
  h2 { margin-top: 0; }
  .actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
