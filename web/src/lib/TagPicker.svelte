<script lang="ts">
  import { onMount } from 'svelte';
  import { errorMessage } from './api';
  import { buildTagIndex, findExact, suggest, type TagIndex } from './tagSuggest';
  import { createTag, fetchRecentTags, fetchTags } from './tags';

  let { selected = $bindable<string[]>([]) }: { selected?: string[] } = $props();

  let index = $state.raw<TagIndex>({ entries: [] });
  let recents = $state<string[]>([]);
  let query = $state('');
  let open = $state(true);
  let active = $state(-1);
  let error = $state('');
  let busy = $state(false);

  const uid = $props.id();
  const listId = `${uid}-list`;
  const optionId = (i: number) => `${uid}-opt-${i}`;

  const input = $derived(query.trim());
  const suggestions = $derived(suggest(index, query, selected));
  const canCreate = $derived(input !== '' && findExact(index, input) === undefined);
  const optionCount = $derived(suggestions.length + (canCreate ? 1 : 0));
  const expanded = $derived(open && optionCount > 0);

  onMount(async () => {
    try {
      const [tree, recent] = await Promise.all([fetchTags(), fetchRecentTags()]);
      index = buildTagIndex(tree);
      recents = recent;
    } catch (e) {
      error = errorMessage(e);
    }
  });

  function take(path: string) {
    if (!selected.includes(path)) selected = [...selected, path];
    query = '';
    active = -1;
    error = '';
  }

  function toggle(path: string) {
    selected = selected.includes(path) ? selected.filter((t) => t !== path) : [...selected, path];
  }

  async function create() {
    if (busy || !input) return;
    busy = true;
    error = '';
    try {
      const path = await createTag(input);
      const known = new Set(index.entries.map((e) => e.path));
      if (!known.has(path)) {
        index = buildTagIndex([...index.entries.map((e) => ({ name: e.path, path: e.path, count: 0, children: [] })), { name: path, path, count: 0, children: [] }]);
      }
      take(path);
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }

  function choose(i: number) {
    if (i < suggestions.length) take(suggestions[i]);
    else void create();
  }

  function onkeydown(event: KeyboardEvent) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      if (optionCount === 0) return;
      event.preventDefault();
      open = true;
      const step = event.key === 'ArrowDown' ? 1 : -1;
      active = active < 0 ? (step === 1 ? 0 : optionCount - 1) : (active + step + optionCount) % optionCount;
    } else if (event.key === 'Escape') {
      if (expanded) {
        event.preventDefault();
        event.stopPropagation();
        open = false;
        active = -1;
      }
    } else if (event.key === 'Enter') {
      event.preventDefault();
      if (!input) return;
      if (expanded && active >= 0 && active < optionCount) return choose(active);
      const exact = findExact(index, input);
      if (exact !== undefined) take(exact);
      else void create();
    }
  }
</script>

<div class="picker">
  {#if recents.length > 0}
    <section aria-labelledby="{uid}-recent">
      <h3 id="{uid}-recent">Zuletzt verwendet</h3>
      <div class="chips">
        {#each recents as path (path)}
          <button type="button" class="chip" aria-pressed={selected.includes(path)} onclick={() => toggle(path)}>{path}</button>
        {/each}
      </div>
    </section>
  {/if}

  <label for="{uid}-input">Tag suchen oder neu anlegen</label>
  <input
    id="{uid}-input"
    type="text"
    role="combobox"
    autocomplete="off"
    aria-expanded={expanded}
    aria-controls={listId}
    aria-autocomplete="list"
    aria-activedescendant={expanded && active >= 0 ? optionId(active) : undefined}
    bind:value={query}
    oninput={() => {
      open = true;
      active = -1;
    }}
    {onkeydown}
  />
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  <ul id={listId} role="listbox" aria-label="Vorschläge" hidden={!expanded}>
    {#if expanded}
      {#each suggestions as path, i (path)}
        <li id={optionId(i)} role="option" aria-selected={active === i} class:active={active === i}>
          <button type="button" tabindex="-1" onclick={() => choose(i)}>{path}</button>
        </li>
      {/each}
      {#if canCreate}
        {@const i = suggestions.length}
        <li id={optionId(i)} role="option" aria-selected={active === i} class:active={active === i}>
          <button type="button" tabindex="-1" onclick={() => choose(i)}>„{input}“ neu anlegen</button>
        </li>
      {/if}
    {/if}
  </ul>

  {#if selected.length > 0}
    <ul class="chips selected" aria-label="Ausgewählte Tags">
      {#each selected as tag (tag)}
        <li class="chip-row">
          <span class="chip static">{tag}</span>
          <button type="button" class="remove" aria-label="{tag} entfernen" onclick={() => (selected = selected.filter((t) => t !== tag))}>×</button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .picker { display: block; }
  h3 { font-size: 1rem; margin: 0.5rem 0 0.25rem; }
  label { display: block; margin: 0.5rem 0 0.25rem; }
  input { width: 100%; min-width: 0; min-height: 44px; padding: 0 0.5rem; font-size: 1rem; box-sizing: border-box; }
  .chips { display: flex; flex-wrap: wrap; gap: 0.5rem; list-style: none; margin: 0.5rem 0; padding: 0; }
  .chip, .remove, li button { min-height: 44px; min-width: 44px; font-size: 1rem; padding: 0 0.75rem; box-sizing: border-box; }
  .chip-row { display: flex; align-items: center; gap: 0.25rem; }
  .chip.static { display: inline-flex; align-items: center; }
  .chip[aria-pressed='true'] { font-weight: 600; outline: 2px solid currentColor; }
  [role='listbox'] { list-style: none; margin: 0.25rem 0; padding: 0; }
  [role='listbox'][hidden] { display: none; }
  [role='option'] button { display: block; width: 100%; text-align: left; }
  [role='option'].active button { outline: 2px solid currentColor; }
  [role='alert'] { margin: 0.25rem 0; }
</style>
