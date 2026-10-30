<script lang="ts">
  import { onMount } from 'svelte';
  import Thumbnail from '../Thumbnail.svelte';
  import { fetchGroups, type PhotoGroups, type PhotoItem } from '../photos';
  import { buildRows, columnsFor, monthForFolder, offsetOfMonth, tileSizeFor, windowAt, type Row } from './layout';
  import { createPhotoLoader } from './loader';
  import { dayLabel, monthLabel } from './labels';
  import MonthNav from './MonthNav.svelte';
  import SelectionBar from '../SelectionBar.svelte';
  import { selection } from '../selectionStore.svelte';
  import { longpress } from '../longpress';

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

  function onTileClick(event: MouseEvent, item: PhotoItem) {
    const ref = { id: item.id, takenAt: item.takenAt };
    if (event.shiftKey) selection.selectRange(ref);
    else if (selection.active) selection.toggle(ref);
    else onOpen(item.id);
  }

  // Photo whose action menu is open; only ever set while selecting.
  let menuFor = $state<PhotoItem | null>(null);

  function onTileLongPress(item: PhotoItem) {
    const ref = { id: item.id, takenAt: item.takenAt };
    if (selection.active) menuFor = item;
    else selection.toggle(ref);
  }

  function rangeToMenuPhoto() {
    if (menuFor) selection.selectRange({ id: menuFor.id, takenAt: menuFor.takenAt });
    menuFor = null;
  }

  const fileName = (relPath: string) => relPath.split('/').pop() ?? relPath;
</script>

<section class="grid" aria-label="Fotos">
  {#if groups}
    <div class="toolbar">
      <MonthNav folders={groups.folders} onPick={jump} />
      {#if !selection.active}
        <button type="button" class="action" onclick={() => selection.start()}>Auswählen</button>
      {/if}
    </div>
  {/if}
  <SelectionBar />
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
              <button type="button" class="action" disabled={selection.isGroupFullySelected(row.month)} onclick={() => selection.selectGroup(row.month)}>Alle auswählen</button>
            </div>
          {:else if row.kind === 'day'}
            <div class="head">
              <h3>{dayLabel(row.day)}</h3>
              <span data-testid="group-count">{row.count}</span>
              <button type="button" class="action" disabled={selection.isGroupFullySelected(row.day)} onclick={() => selection.selectGroup(row.day)}>Alle auswählen</button>
            </div>
          {:else}
            {#each photosOf(row) as { index, item } (index)}
              <div class="cell" style:left="{(index - row.start) * tile}px" style:width="{tile}px" style:height="{tile}px">
                {#if item}
                  <button
                    type="button"
                    class="tile"
                    class:selected={selection.active && selection.isSelected({ id: item.id, takenAt: item.takenAt })}
                    data-testid="grid-tile"
                    data-id={item.id}
                    data-taken={item.takenAt ?? ''}
                    aria-label={fileName(item.relPath)}
                    aria-pressed={selection.active ? selection.isSelected({ id: item.id, takenAt: item.takenAt }) : undefined}
                    onclick={(e) => onTileClick(e, item)}
                    use:longpress={() => onTileLongPress(item)}
                  >
                    <Thumbnail id={item.id} mediaType={item.mediaType} thumbKey={item.thumbKey} label={fileName(item.relPath)} />
                  </button>
                  {#if selection.active}
                    <button type="button" class="detail" aria-label="Details öffnen" onclick={() => onOpen(item.id)}>⤢</button>
                  {/if}
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

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') menuFor = null; }} />

{#if menuFor}
  <!-- The backdrop turns any tap outside the menu into "close". -->
  <div class="backdrop" role="presentation" onclick={() => (menuFor = null)}></div>
  <div class="menu" role="menu" aria-label="Aktionen">
    <button type="button" role="menuitem" disabled={!selection.state.anchor} onclick={rangeToMenuPhoto}>Bereich bis hier</button>
  </div>
{/if}

<style>
  .grid { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  .scroller { flex: 1; min-height: 0; overflow-x: hidden; overflow-y: auto; }
  .canvas { position: relative; width: 100%; }
  .row { position: absolute; left: 0; right: 0; overflow: hidden; }
  .head { display: flex; align-items: baseline; gap: 0.5rem; height: 100%; padding: 0 4px; box-sizing: border-box; }
  h2, h3 { margin: 0; }
  .cell { position: absolute; top: 0; padding: 2px; box-sizing: border-box; }
  .toolbar { display: flex; gap: 0.5rem; }
  .action { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  .head .action { margin-left: auto; }
  .tile { user-select: none; -webkit-user-select: none; -webkit-touch-callout: none; touch-action: manipulation; }
  .backdrop { position: fixed; inset: 0; z-index: 10; }
  .menu { position: fixed; left: 0; right: 0; bottom: 0; z-index: 11; display: flex; flex-direction: column; padding: 0.5rem; background: #fff; box-shadow: 0 -2px 8px #0004; }
  .menu button { min-height: 44px; font-size: 1rem; }
  .tile.selected { outline: 4px solid #1a73e8; outline-offset: -4px; opacity: 0.8; }
  .detail { position: absolute; top: 2px; right: 2px; width: 44px; height: 44px; font-size: 1.25rem; background: #fffc; border-radius: 4px; }
  button.tile { display: block; width: 100%; height: 100%; padding: 0; border: 0; background: none; cursor: pointer; }
  .placeholder { width: 100%; height: 100%; background: #eee; }
</style>
