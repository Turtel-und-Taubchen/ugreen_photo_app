<script lang="ts">
  import { errorMessage } from './api';
  import { createJob } from './jobs';
  import { toRequest } from './selection';
  import { selection } from './selectionStore.svelte';
  import { fetchSelectionTags, type SelectionTag } from './tags';

  let { onClose }: { onClose: () => void } = $props();

  let dialog: HTMLDialogElement;
  let available = $state<SelectionTag[]>([]);
  let loaded = $state(false);
  let checked = $state<string[]>([]);
  let error = $state('');
  let busy = $state(false);

  const label = (t: SelectionTag) => `${t.path} (${t.count} ${t.count === 1 ? 'Foto' : 'Fotos'})`;

  function modal(node: HTMLDialogElement) {
    dialog = node;
    node.showModal();
    fetchSelectionTags(toRequest(selection.state))
      .then((tags) => (available = tags))
      .catch((e) => (error = errorMessage(e)))
      .finally(() => (loaded = true));
  }

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

<dialog use:modal aria-labelledby="remove-title" onclose={onClose}>
  <h2 id="remove-title">Tags entfernen</h2>
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
  <div class="actions">
    <button type="button" onclick={() => dialog.close()}>Abbrechen</button>
    <button type="button" disabled={checked.length === 0 || busy} onclick={remove}>Entfernen</button>
  </div>
</dialog>

<style>
  dialog { width: min(32rem, calc(100vw - 2rem)); box-sizing: border-box; }
  h2 { margin-top: 0; }
  ul { list-style: none; margin: 0; padding: 0; }
  label { display: flex; align-items: center; gap: 0.5rem; min-height: 44px; }
  input[type='checkbox'] { width: 1.5rem; height: 1.5rem; }
  .actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
