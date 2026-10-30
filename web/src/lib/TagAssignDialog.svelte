<script lang="ts">
  import { errorMessage } from './api';
  import Dialog from './Dialog.svelte';
  import { createJob } from './jobs';
  import { toRequest } from './selection';
  import { selection } from './selectionStore.svelte';
  import TagPicker from './TagPicker.svelte';

  let { onClose }: { onClose: () => void } = $props();

  let tags = $state<string[]>([]);
  let error = $state('');
  let busy = $state(false);

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

<Dialog title="Tags zuweisen" {onClose}>
  <TagPicker bind:selected={tags} />
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  {#snippet actions(close)}
    <button type="button" onclick={close}>Abbrechen</button>
    <button type="button" disabled={tags.length === 0 || busy} onclick={assign}>Zuweisen</button>
  {/snippet}
</Dialog>
