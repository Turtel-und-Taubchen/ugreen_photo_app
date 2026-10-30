<script lang="ts">
  import { errorMessage } from './api';
  import Dialog from './Dialog.svelte';
  import { createJob } from './jobs';
  import { toRequest } from './selection';
  import { selection } from './selectionStore.svelte';

  let { onClose }: { onClose: () => void } = $props();

  let error = $state('');
  let busy = $state(false);
  // The question names the count it was asked with, even if the selection changes behind the modal.
  const count = selection.count;

  async function remove() {
    error = '';
    busy = true;
    try {
      await createJob(toRequest(selection.state), 'location.remove', {});
      onClose();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }
</script>

<Dialog title={`Ort aus ${count} ${count === 1 ? 'Foto' : 'Fotos'} entfernen?`} {onClose}>
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  {#snippet actions(close)}
    <button type="button" onclick={close}>Abbrechen</button>
    <button type="button" disabled={busy} onclick={remove}>Entfernen</button>
  {/snippet}
</Dialog>
