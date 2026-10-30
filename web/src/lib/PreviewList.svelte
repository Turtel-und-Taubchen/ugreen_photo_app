<script lang="ts">
  import { api } from './api';
  import Thumbnail from './Thumbnail.svelte';

  interface Item {
    id: number;
    relPath: string;
    mediaType: string;
    thumbKey: string;
  }

  // A single page is enough for now; the server caps the page size.
  const items = api<{ items: Item[] }>('/api/photos?limit=200').then((r) => r.items);
</script>

<section aria-label="Vorschau">
  {#await items then list}
    <ul>
      {#each list as item (item.id)}
        <li data-testid={`thumb-${item.relPath}`}>
          <Thumbnail
            id={item.id}
            mediaType={item.mediaType}
            thumbKey={item.thumbKey}
            label={item.relPath.split('/').pop() ?? item.relPath}
          />
        </li>
      {/each}
    </ul>
  {:catch}
    <p role="alert">Vorschau konnte nicht geladen werden</p>
  {/await}
</section>

<style>
  ul { list-style: none; margin: 0; padding: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(104px, 1fr)); gap: 4px; }
</style>
