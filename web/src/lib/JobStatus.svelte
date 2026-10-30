<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchJob, fetchJobs, isActive, progressText, resultText, type Job, type JobDetail } from './jobs';

  const POLL_RUNNING_MS = 2000;
  const POLL_IDLE_MS = 30000;
  const HINT =
    "UGREEN Fotos zeigt nur den Ort an – nach ‚Aktualisieren' in UGREEN Fotos. Tags und Prüfstatus sind nur in PhotoCurator sichtbar.";

  let jobs = $state<Job[]>([]);
  let detail = $state<JobDetail | null>(null);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let gone = false;

  // The server lists newest first; the oldest active job is the one being written.
  const active = $derived([...jobs].reverse().filter(isActive));
  const current = $derived(active[0]);
  const last = $derived(jobs.find((j) => !isActive(j)));

  // Re-arms itself after every response so the interval follows the job state.
  async function poll() {
    clearTimeout(timer);
    try {
      jobs = await fetchJobs();
      const shown = jobs.find((j) => !isActive(j));
      if (shown && detail?.id !== shown.id) detail = null;
      if (shown && !detail) detail = await fetchJob(shown.id);
    } catch {
      // The next poll retries; a stale display is better than an error flash.
    }
    if (!gone) timer = setTimeout(poll, active.length > 0 ? POLL_RUNNING_MS : POLL_IDLE_MS);
  }

  onMount(() => {
    poll();
    return () => {
      gone = true;
      clearTimeout(timer);
    };
  });
</script>

{#if current || last}
  <section class="jobs" aria-label="Schreibjobs">
    {#if current}
      <p>{progressText(current, active.length - 1)}</p>
    {:else if last}
      <p>{resultText(last)}</p>
      {#if detail && detail.items.some((i) => i.state !== 'ok')}
        <details>
          <summary>Details</summary>
          <ul>
            {#each detail.items.filter((i) => i.state !== 'ok') as item (item.rel_path)}
              <li><span class="path">{item.rel_path}</span> · {item.reason}</li>
            {/each}
          </ul>
        </details>
      {/if}
      <p class="hint">{HINT}</p>
    {/if}
  </section>
{/if}

<style>
  .jobs { flex-basis: 100%; }
  p { margin: 0; }
  .hint { font-size: 0.875rem; }
  .path { overflow-wrap: anywhere; }
</style>
