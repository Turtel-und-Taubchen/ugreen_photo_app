<script lang="ts">
  import { errorMessage } from './api';
  import Dialog from './Dialog.svelte';
  import { createJob } from './jobs';
  import { toRequest } from './selection';
  import { selection } from './selectionStore.svelte';
  import { fetchSelectionTags, type SelectionTag } from './tags';

  let { onClose }: { onClose: () => void } = $props();

  let available = $state<SelectionTag[]>([]);
  let loaded = $state(false);
  let checked = $state<string[]>([]);
  let error = $state('');
  let busy = $state(false);

  const label = (t: SelectionTag) => `${t.path} (${t.count} ${t.count === 1 ? 'Foto' : 'Fotos'})`;

  fetchSelectionTags(toRequest(selection.state))
    .then((tags) => (available = tags))
    .catch((e) => (error = errorMessage(e)))
    .finally(() => (loaded = true));

  async function remove() {
    error = '';
    busy = true;
    try {
      await createJob(toRequest(selection.state), 'tags.remove', { tags: checked });
      onClose();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }
</script>

<Dialog title="Tags entfernen" {onClose}>
  {#if loaded && available.length === 0 && !error}
    <p>Die ausgewählten Fotos haben keine Tags.</p>
  {/if}
  <ul>
    {#each available as tag (tag.path)}
      <li>
        <label>
          <input type="checkbox" value={tag.path} bind:group={checked} />
          {label(tag)}
        </label>
      </li>
    {/each}
  </ul>
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  {#snippet actions(close)}
    <button type="button" onclick={close}>Abbrechen</button>
    <button type="button" disabled={checked.length === 0 || busy} onclick={remove}>Entfernen</button>
  {/snippet}
</Dialog>

<style>
  ul { list-style: none; margin: 0; padding: 0; }
  label { display: flex; align-items: center; gap: 0.5rem; min-height: 44px; }
  input[type='checkbox'] { width: 1.5rem; height: 1.5rem; }
</style>
