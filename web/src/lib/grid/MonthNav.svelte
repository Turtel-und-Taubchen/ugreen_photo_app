<script lang="ts">
  import type { FolderCount } from '../photos';
  import { monthLabel } from './labels';

  interface Props {
    folders: FolderCount[];
    /** Receives the `YYYY/MM` folder that was picked. */
    onPick: (folder: string) => void;
  }
  let { folders, onPick }: Props = $props();

  let open = $state(false);

  const years = $derived.by(() => {
    const list: [string, string[]][] = [];
    for (const { folder } of [...folders].sort((a, b) => b.folder.localeCompare(a.folder))) {
      const [year, mm] = folder.split('/');
      if (!/^\d{4}$/.test(year) || !/^\d{2}$/.test(mm ?? '')) continue;
      const last = list[list.length - 1];
      if (last && last[0] === year) last[1].push(folder);
      else list.push([year, [folder]]);
    }
    return list;
  });

  function pick(folder: string) {
    open = false;
    onPick(folder);
  }
</script>

<nav class="nav" aria-label="Zeitraum wählen">
  <button type="button" aria-expanded={open} onclick={() => (open = !open)}>Zeitraum</button>
  {#if open}
    <div class="panel">
      {#each years as [year, list] (year)}
        <section>
          <h3>{year}</h3>
          <div class="months">
            {#each list as folder (folder)}
              <button type="button" onclick={() => pick(folder)}>{monthLabel(folder.replace('/', '-'))}</button>
            {/each}
          </div>
        </section>
      {/each}
      {#if years.length === 0}
        <p>Keine Monate vorhanden</p>
      {/if}
    </div>
  {/if}
</nav>

<style>
  .nav { position: relative; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  .panel { position: absolute; z-index: 2; left: 0; top: 100%; width: min(22rem, calc(100vw - 16px)); max-height: 60vh; overflow-y: auto; padding: 0.5rem; background: #fff; border: 1px solid #ccc; box-sizing: border-box; }
  h3 { margin: 0.5rem 0 0.25rem; }
  .months { display: flex; flex-wrap: wrap; gap: 0.25rem; }
</style>
