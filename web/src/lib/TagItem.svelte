<script lang="ts">
  import type { TagNode } from './tags';
  import TagItem from './TagItem.svelte';

  let { node, onDelete }: { node: TagNode; onDelete: (path: string) => void } = $props();
</script>

<li>
  <div class="row">
    <span>{node.name} ({node.count})</span>
    <button type="button" aria-label="{node.name} löschen" disabled={node.count > 0} onclick={() => onDelete(node.path)}>
      Löschen
    </button>
    {#if node.count > 0}
      <small>Zuerst über „Tags entfernen“ von den Fotos lösen</small>
    {/if}
  </div>
  {#if node.children.length > 0}
    <ul>
      {#each node.children as child (child.path)}
        <TagItem node={child} {onDelete} />
      {/each}
    </ul>
  {/if}
</li>

<style>
  ul { list-style: none; margin: 0; padding-left: 1.25rem; }
  .row { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; min-height: 44px; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
