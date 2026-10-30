<script lang="ts">
  import { selection } from './selectionStore.svelte';
  import RemovePlaceDialog from './RemovePlaceDialog.svelte';
  import SetPlaceDialog from './SetPlaceDialog.svelte';
  import TagAssignDialog from './TagAssignDialog.svelte';
  import TagRemoveDialog from './TagRemoveDialog.svelte';

  let dialog = $state<'assign' | 'remove' | 'place-set' | 'place-remove' | null>(null);
</script>

{#if selection.active}
  <div class="bar" role="region" aria-label="Auswahl">
    <span data-testid="selection-count" aria-live="polite">{selection.count} ausgewählt</span>
    <button type="button" disabled={selection.count === 0} onclick={() => (dialog = 'assign')}>Tags zuweisen</button>
    <button type="button" disabled={selection.count === 0} onclick={() => (dialog = 'remove')}>Tags entfernen</button>
    <button type="button" disabled={selection.count === 0} onclick={() => (dialog = 'place-set')}>Ort setzen</button>
    <button type="button" disabled={selection.count === 0} onclick={() => (dialog = 'place-remove')}>Ort entfernen</button>
    <button type="button" onclick={() => selection.clear()}>Auswahl aufheben</button>
  </div>
  {#if dialog === 'assign'}
    <TagAssignDialog onClose={() => (dialog = null)} />
  {:else if dialog === 'remove'}
    <TagRemoveDialog onClose={() => (dialog = null)} />
  {:else if dialog === 'place-set'}
    <SetPlaceDialog onClose={() => (dialog = null)} />
  {:else if dialog === 'place-remove'}
    <RemovePlaceDialog onClose={() => (dialog = null)} />
  {/if}
{/if}

<style>
  .bar { display: flex; flex-wrap: wrap; align-items: center; gap: 1rem; padding: 0.25rem 0.5rem; background: #e8f0fe; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
