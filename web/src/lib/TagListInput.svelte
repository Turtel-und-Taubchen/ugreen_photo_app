<script lang="ts">
  let { tags = $bindable<string[]>([]) }: { tags?: string[] } = $props();

  let draft = $state('');

  function add(event: SubmitEvent) {
    event.preventDefault();
    const path = draft.trim();
    if (path && !tags.includes(path)) tags = [...tags, path];
    draft = '';
  }
</script>

<section aria-label="Tag-Liste">
  <form onsubmit={add}>
    <label for="batch-tag">Tag</label>
    <div class="field">
      <input id="batch-tag" type="text" bind:value={draft} autocomplete="off" />
      <button type="submit">Hinzufügen</button>
    </div>
  </form>
  <ul>
    {#each tags as tag (tag)}
      <li>
        <span>{tag}</span>
        <button type="button" aria-label="{tag} entfernen" onclick={() => (tags = tags.filter((t) => t !== tag))}>×</button>
      </li>
    {/each}
  </ul>
</section>

<style>
  .field { display: flex; gap: 0.5rem; }
  input { flex: 1; min-width: 0; min-height: 44px; padding: 0 0.5rem; font-size: 1rem; box-sizing: border-box; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  label { display: block; margin: 0.5rem 0 0.25rem; }
  ul { list-style: none; margin: 0.5rem 0; padding: 0; }
  li { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
</style>
