<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchScan, startScan, isRunning, scanStatusText, scanErrorText, type ScanStatus } from './scan';

  const POLL_RUNNING_MS = 2000;
  const POLL_IDLE_MS = 30000;

  let status = $state<ScanStatus | null>(null);
  let failed = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let gone = false;

  const running = $derived(status ? isRunning(status) : false);

  // Re-arms itself after every response so the interval follows the scan state.
  async function apply(request: () => Promise<ScanStatus>) {
    clearTimeout(timer);
    try {
      status = await request();
      failed = false;
    } catch {
      failed = true;
    }
    if (!gone) timer = setTimeout(() => apply(fetchScan), running ? POLL_RUNNING_MS : POLL_IDLE_MS);
  }

  onMount(() => {
    apply(fetchScan);
    return () => {
      gone = true;
      clearTimeout(timer);
    };
  });
</script>

<section class="scan" aria-label="Bibliothek">
  <div class="row">
    <p role="status">{status ? scanStatusText(status) : ''}</p>
    <button type="button" disabled={running} onclick={() => apply(startScan)}>Neu einlesen</button>
  </div>
  {#if running && status}
    <progress max={Math.max(status.total, 1)} value={status.done} aria-label="Fortschritt"></progress>
  {/if}
  {#if status && scanErrorText(status)}
    <p class="warn">{scanErrorText(status)}</p>
  {/if}
  {#if failed}
    <p role="alert">Status konnte nicht geladen werden</p>
  {/if}
  {#if status && status.unreadable_count > 0}
    <h2>Nicht lesbar ({status.unreadable_count})</h2>
    <ul>
      {#each status.unreadable as file (file.rel_path)}
        <li><span class="path">{file.rel_path}</span> · nicht lesbar: {file.error}</li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .row { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 0.5rem; }
  p { margin: 0; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  progress { width: 100%; }
  .path { overflow-wrap: anywhere; }
  ul { padding-left: 1.25rem; }
  .warn { color: #b00020; }
</style>
