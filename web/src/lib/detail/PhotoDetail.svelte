<script lang="ts">
  import { onMount } from 'svelte';
  import Thumbnail from '../Thumbnail.svelte';
  import { fetchPhoto, type PhotoDetail } from '../photos';
  import { swipeDirection } from './swipe';

  interface Props {
    id: number;
    onClose: () => void;
    /** Steps to a neighbouring photo; the host owns the current id. */
    onStep: (id: number) => void;
  }
  let { id, onClose, onStep }: Props = $props();

  let photo = $state<PhotoDetail | null>(null);
  let failed = $state(false);

  const STATE_LABELS: Record<string, string> = { open: 'offen', done: 'erledigt', irrelevant: 'nicht relevant' };
  const DATE_HINTS: Record<string, string> = {
    filename: 'Datum aus dem Dateinamen',
    mtime: 'Datum aus der Änderungszeit',
  };

  // Stale responses of an earlier id must not overwrite the current photo.
  $effect(() => {
    const wanted = id;
    let current = true;
    failed = false;
    fetchPhoto(wanted)
      .then((p) => {
        if (current) photo = p;
      })
      .catch(() => {
        if (current) failed = true;
      });
    return () => {
      current = false;
    };
  });

  // Taken dates are local wall-clock times; parsing the parts avoids any time zone shift.
  function formatTaken(takenAt: string | null): string {
    const m = takenAt?.match(/^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2}))?/);
    if (!m) return 'kein Datum';
    const [y, mo, d, h, mi] = m.slice(1).map((v) => (v === undefined ? 0 : Number(v)));
    const date = new Date(y, mo - 1, d, h, mi);
    return m[4] === undefined
      ? date.toLocaleDateString('de-DE', { dateStyle: 'long' })
      : date.toLocaleString('de-DE', { dateStyle: 'long', timeStyle: 'short' });
  }

  // Keys and swipes ignore the end of the list: there is nothing to step to.
  function step(target: number | null | undefined): void {
    if (target !== null && target !== undefined) onStep(target);
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === 'Escape') history.back();
    else if (e.key === 'ArrowLeft') step(photo?.prev_id);
    else if (e.key === 'ArrowRight') step(photo?.next_id);
  }

  let start: { x: number; y: number; pointerId: number } | null = null;

  function onPointerDown(e: PointerEvent): void {
    start = { x: e.clientX, y: e.clientY, pointerId: e.pointerId };
  }

  function onPointerUp(e: PointerEvent): void {
    if (!start || start.pointerId !== e.pointerId) return;
    const dir = swipeDirection(e.clientX - start.x, e.clientY - start.y);
    start = null;
    if (dir === 'left') step(photo?.next_id);
    else if (dir === 'right') step(photo?.prev_id);
  }

  // The entry pushed on open makes the browser's back gesture close the view instead of leaving the app.
  onMount(() => {
    history.pushState({ photoDetail: true }, '');
    const onPop = () => onClose();
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  });
</script>

<svelte:window onkeydown={onKey} />

<div class="overlay" role="dialog" aria-modal="true" aria-label="Foto-Details" data-testid="photo-detail">
  <button type="button" class="close" aria-label="Schließen" onclick={() => history.back()}>×</button>
  {#if failed}
    <p role="alert">Foto konnte nicht geladen werden</p>
  {:else if photo}
    <div
      class="stage" data-testid="detail-stage"
      role="presentation"
      onpointerdown={onPointerDown}
      onpointerup={onPointerUp}
      onpointercancel={() => (start = null)}
    >
      {#key photo.id}
        <Thumbnail id={photo.id} mediaType={photo.media_type} thumbKey={photo.thumb_key} size="l" label={photo.rel_path} />
      {/key}
    </div>
    <dl>
      <dt>Aufnahmedatum</dt>
      <dd data-testid="detail-taken">
        {formatTaken(photo.taken_at)}
        {#if DATE_HINTS[photo.taken_source]}<small>{DATE_HINTS[photo.taken_source]}</small>{/if}
      </dd>
      <dt>Tags</dt>
      <dd data-testid="detail-tags">
        {#if photo.tags.length === 0}keine Tags{:else}<ul>{#each photo.tags as tag (tag)}<li>{tag}</li>{/each}</ul>{/if}
      </dd>
      <dt>Ort</dt>
      <dd data-testid="detail-place">{photo.location.label || 'kein Ort'}</dd>
      <dt>Status</dt>
      <dd data-testid="detail-state">{STATE_LABELS[photo.review_state] ?? photo.review_state}</dd>
      <dt>Pfad</dt>
      <dd data-testid="detail-path">{photo.rel_path}</dd>
    </dl>
  {/if}
</div>

<style>
  .overlay { position: fixed; inset: 0; z-index: 10; display: flex; flex-direction: column; gap: 8px; padding: 8px; background: #fff; overflow-y: auto; }
  .close { position: absolute; top: 8px; right: 8px; z-index: 1; width: 44px; height: 44px; font-size: 1.5rem; }
  .stage { flex: 1 1 60%; min-height: 200px; touch-action: pan-y; }
  .stage :global(.tile) { width: 100%; height: 100%; aspect-ratio: auto; background: #111; }
  .stage :global(img) { object-fit: contain; }
  dl { margin: 0; display: grid; grid-template-columns: max-content 1fr; gap: 4px 12px; }
  dt { font-weight: 600; }
  dd { margin: 0; overflow-wrap: anywhere; }
  small { display: block; color: #555; }
  ul { margin: 0; padding-left: 1.1rem; }
</style>
