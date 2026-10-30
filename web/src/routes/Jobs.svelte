<script lang="ts">
  import { onMount } from 'svelte';
  import { errorMessage } from '../lib/api';
  import {
    actionLabel, canUndo, fetchJob, fetchJobs, formatTime, isActive, resultSummary, undoJob,
    type Job, type JobDetail,
  } from '../lib/jobs';

  const POLL_ACTIVE_MS = 2000;
  const POLL_IDLE_MS = 15000;

  let jobs = $state<Job[]>([]);
  let loaded = $state(false);
  let error = $state('');
  let details = $state<Record<number, JobDetail>>({});
  let confirming = $state<number | null>(null);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let gone = false;

  // Re-arms itself after every response so the interval follows the job state.
  async function poll() {
    clearTimeout(timer);
    try {
      jobs = await fetchJobs();
      loaded = true;
      // Open rows show a running job's progress; refresh them with the list.
      for (const id of Object.keys(details).map(Number)) details[id] = await fetchJob(id);
    } catch (e) {
      error = errorMessage(e);
    }
    if (!gone) timer = setTimeout(poll, jobs.some(isActive) ? POLL_ACTIVE_MS : POLL_IDLE_MS);
  }

  async function toggle(event: Event, id: number) {
    if (!(event.currentTarget as HTMLDetailsElement).open) {
      delete details[id];
      return;
    }
    try {
      details[id] = await fetchJob(id);
    } catch (e) {
      error = errorMessage(e);
    }
  }

  async function undo(id: number) {
    error = '';
    confirming = null;
    try {
      await undoJob(id);
    } catch (e) {
      error = errorMessage(e);
    }
    await poll();
  }

  onMount(() => {
    poll();
    return () => {
      gone = true;
      clearTimeout(timer);
    };
  });
</script>

<main>
  <h2>Jobs</h2>
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  {#if loaded && jobs.length === 0}
    <p>Noch keine Jobs.</p>
  {/if}
  <ul class="list">
    {#each jobs as job (job.id)}
      <li data-job-id={job.id}>
        <div class="head">
          <strong>{actionLabel(job, jobs)}</strong>
          <span>{job.user}</span>
          <time datetime={job.created_at}>{formatTime(job.created_at)}</time>
          <span>{job.total} {job.total === 1 ? 'Datei' : 'Dateien'}</span>
          <span>{isActive(job) ? `Läuft: ${job.done} von ${job.total}` : resultSummary(job)}</span>
        </div>
        <details ontoggle={(e) => toggle(e, job.id)}>
          <summary>Ergebnis je Datei</summary>
          {#if details[job.id]}
            <ul>
              {#each details[job.id].items as item (item.rel_path)}
                <li><span class="path">{item.rel_path}</span> · {item.state}{item.reason ? ` · ${item.reason}` : ''}</li>
              {/each}
            </ul>
          {/if}
        </details>
        {#if canUndo(job)}
          {#if confirming === job.id}
            <span class="confirm">
              Job #{job.id} wirklich rückgängig machen?
              <button type="button" onclick={() => undo(job.id)}>Ja, rückgängig</button>
              <button type="button" onclick={() => (confirming = null)}>Abbrechen</button>
            </span>
          {:else}
            <button type="button" onclick={() => (confirming = job.id)}>Rückgängig</button>
          {/if}
        {/if}
      </li>
    {/each}
  </ul>
</main>

<style>
  main { flex: 1; min-height: 0; overflow-y: auto; }
  .list { list-style: none; margin: 1rem 0; padding: 0; }
  .list > li { padding: 0.5rem 0; border-bottom: 1px solid #8884; }
  .head { display: flex; flex-wrap: wrap; gap: 0.25rem 1rem; }
  summary { min-height: 44px; display: flex; align-items: center; }
  .path { overflow-wrap: anywhere; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
