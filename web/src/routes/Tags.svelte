<script lang="ts">
  import { errorMessage } from '../lib/api';
  import TagItem from '../lib/TagItem.svelte';
  import { createTag, deleteTag, fetchTags, type TagNode } from '../lib/tags';

  let tree = $state<TagNode[]>([]);
  let loaded = $state(false);
  let path = $state('');
  let error = $state('');

  async function reload() {
    tree = await fetchTags();
    loaded = true;
  }

  async function create(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    try {
      await createTag(path);
      path = '';
      await reload();
    } catch (e) {
      error = errorMessage(e);
    }
  }

  async function remove(target: string) {
    error = '';
    try {
      await deleteTag(target);
    } catch (e) {
      error = errorMessage(e);
    }
    await reload();
  }

  reload().catch((e) => (error = errorMessage(e)));
</script>

<main>
  <h2>Tags</h2>
  <form onsubmit={create}>
    <label for="new-tag">Neuer Tag</label>
    <div class="field">
      <input id="new-tag" type="text" bind:value={path} autocomplete="off" />
      <button type="submit">Anlegen</button>
    </div>
    {#if error}
      <p role="alert">{error}</p>
    {/if}
  </form>
  {#if loaded && tree.length === 0}
    <p>Noch keine Tags.</p>
  {/if}
  <ul class="root">
    {#each tree as node (node.path)}
      <TagItem {node} onDelete={remove} />
    {/each}
  </ul>
</main>

<style>
  main { flex: 1; min-height: 0; overflow-y: auto; }
  .field { display: flex; gap: 0.5rem; }
  input { flex: 1; min-width: 0; min-height: 44px; padding: 0 0.5rem; font-size: 1rem; box-sizing: border-box; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  label { display: block; margin: 0.5rem 0 0.25rem; }
  .root { list-style: none; margin: 1rem 0; padding: 0; }
</style>
