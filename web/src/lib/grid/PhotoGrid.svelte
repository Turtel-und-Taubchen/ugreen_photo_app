<script lang="ts">
  import { onMount } from 'svelte';
  import Thumbnail from '../Thumbnail.svelte';
  import { fetchGroups, type PhotoGroups } from '../photos';
  import { buildRows, columnsFor, monthForFolder, offsetOfMonth, tileSizeFor, windowAt, type Row } from './layout';
  import { createPhotoLoader } from './loader';
  import { dayLabel, monthLabel } from './labels';
  import MonthNav from './MonthNav.svelte';

  interface Props {
    onOpen: (id: number) => void;
    /** The grid reloads when this switches from true to false. */
    scanRunning?: boolean;
  }
  let { onOpen, scanRunning = false }: Props = $props();

  let groups = $state<PhotoGroups | null>(null);
  let failed = $state(false);
  let width = $state(0);
  let height = $state(0);
  let scrollTop = $state(0);
  // Bumped whenever the loader's contents change, so tiles re-read it.
  let version = $state(0);
  let scroller: HTMLElement;

  const loader = createPhotoLoader({ onChange: () => version++ });

  const columns = $derived(columnsFor(width, height));
  const tile = $derived(tileSizeFor(width, columns));
  const layout = $derived(buildRows(groups ?? { months: [] }, columns, tile));
  const win = $derived(windowAt(layout.rows, scrollTop, height));

  async function loadGroups() {
    try {
      groups = await fetchGroups();
      failed = false;
    } catch {
      failed = true;
    }
  }

  onMount(() => {
    void loadGroups();
  });

  let wasRunning = false;
  $effect(() => {
    const running = scanRunning;
    if (wasRunning && !running) {
      loader.reset();
      version++;
      void loadGroups();
    }
    wasRunning = running;
  });

  // A failed month is not retried here: its failure bumps `version`, which would loop.
  $effect(() => {
    void version;
    for (const row of win.rows) {
      if (row.kind === 'photos' && !loader.failed(row.month)) void loader.ensure(row.month, row.end - 1);
    }
  });

  function jump(folder: string) {
    if (!groups) return;
    const month = monthForFolder(groups, folder);
    const top = month === null ? null : offsetOfMonth(layout.rows, month);
    if (top === null) return;
    scroller.scrollTop = top;
    scrollTop = scroller.scrollTop;
  }

  function rowKey(row: Row): string {
    if (row.kind === 'month') return `m-${row.month}`;
    if (row.kind === 'day') return `d-${row.day}`;
    return `p-${row.day}-${row.start}`;
  }

  function photosOf(row: { month: string; start: number; end: number }) {
    void version;
    return Array.from({ length: row.end - row.start }, (_, k) => {
      const index = row.start + k;
      return { index, item: loader.get(row.month, index) };
    });
  }

  const fileName = (relPath: string) => relPath.split('/').pop() ?? relPath;
</script>

<section class="grid" aria-label="Fotos">
  {#if groups}
    <MonthNav folders={groups.folders} onPick={jump} />
  {/if}
  {#if failed}
    <p role="alert">Fotos konnten nicht geladen werden</p>
  {:else if groups && groups.total === 0}
    <p>Noch keine Fotos</p>
  {/if}
  <div
    class="scroller"
    data-testid="grid-scroller"
    bind:this={scroller}
    bind:clientWidth={width}
    bind:clientHeight={height}
    onscroll={() => (scrollTop = scroller.scrollTop)}
  >
    <div class="canvas" style:height="{layout.height}px">
      {#each win.rows as row (rowKey(row))}
        <div class="row" style:top="{row.top}px" style:height="{row.height}px">
          {#if row.kind === 'month'}
            <div class="head">
              <h2>{monthLabel(row.month)}</h2>
              <span data-testid="group-count">{row.count}</span>
            </div>
          {:else if row.kind === 'day'}
            <div class="head">
              <h3>{dayLabel(row.day)}</h3>
              <span data-testid="group-count">{row.count}</span>
            </div>
          {:else}
            {#each photosOf(row) as { index, item } (index)}
              <div class="cell" style:left="{(index - row.start) * tile}px" style:width="{tile}px" style:height="{tile}px">
                {#if item}
                  <button
                    type="button"
                    data-testid="grid-tile"
                    data-id={item.id}
                    data-taken={item.takenAt ?? ''}
                    aria-label={fileName(item.relPath)}
                    onclick={() => onOpen(item.id)}
                  >
                    <Thumbnail id={item.id} mediaType={item.mediaType} thumbKey={item.thumbKey} label={fileName(item.relPath)} />
                  </button>
                {:else}
                  <div class="placeholder" data-testid="grid-tile"></div>
                {/if}
              </div>
            {/each}
          {/if}
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .grid { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  .scroller { flex: 1; min-height: 0; overflow-x: hidden; overflow-y: auto; }
  .canvas { position: relative; width: 100%; }
  .row { position: absolute; left: 0; right: 0; overflow: hidden; }
  .head { display: flex; align-items: baseline; gap: 0.5rem; height: 100%; padding: 0 4px; box-sizing: border-box; }
  h2, h3 { margin: 0; }
  .cell { position: absolute; top: 0; padding: 2px; box-sizing: border-box; }
  button { display: block; width: 100%; height: 100%; padding: 0; border: 0; background: none; cursor: pointer; }
  .placeholder { width: 100%; height: 100%; background: #eee; }
</style>
