<script lang="ts">
  import { selection } from './selectionStore.svelte';
</script>

{#if selection.active}
  <div class="bar" role="region" aria-label="Auswahl">
    <span data-testid="selection-count" aria-live="polite">{selection.count} ausgewählt</span>
    <button type="button" onclick={() => selection.clear()}>Auswahl aufheben</button>
  </div>
{/if}

<style>
  .bar { display: flex; align-items: center; gap: 1rem; padding: 0.25rem 0.5rem; background: #e8f0fe; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
