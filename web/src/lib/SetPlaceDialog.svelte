<script lang="ts">
  import { errorMessage } from './api';
  import Dialog from './Dialog.svelte';
  import { fetchGeoStatus, geoNotice, type GeoStatus } from './geo';
  import { createJob, type LocationPayload } from './jobs';
  import PlacePicker, { type PickedPlace } from './PlacePicker.svelte';
  import { toRequest } from './selection';
  import { countSelection, selection } from './selectionStore.svelte';

  let { onClose }: { onClose: () => void } = $props();

  let status = $state<GeoStatus | null>(null);
  let loadFailed = $state(false);
  let place = $state<PickedPlace | null>(null);
  let overwrite = $state<{ count: number; located: number } | null>(null);
  let error = $state('');
  let busy = $state(false);

  fetchGeoStatus().then(
    (s) => (status = s),
    () => (loadFailed = true),
  );

  const notice = $derived(geoNotice(status));
  const ready = $derived(place !== null && place.name.trim() !== '');

  // Editing the place invalidates a pending overwrite question that was asked for the old one.
  function changed(next: PickedPlace | null) {
    place = next;
    overwrite = null;
  }

  async function run(mode: LocationPayload['mode']) {
    if (!place) return;
    const payload: LocationPayload = {
      location: { lat: place.lat, lon: place.lon, label: place.name.trim(), ...place.parts },
      mode,
    };
    await createJob(toRequest(selection.state), 'location.set', payload);
    onClose();
  }

  async function guarded(work: () => Promise<void>) {
    error = '';
    busy = true;
    try {
      await work();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }

  const apply = () =>
    guarded(async () => {
      const counts = await countSelection();
      if (counts.located > 0) overwrite = counts;
      else await run('all');
    });
</script>

<Dialog title="Ort setzen" {onClose} wide>
  {#if notice}
    <p>{notice}</p>
  {/if}
  {#if loadFailed}
    <p role="alert">Status der Ortssuche nicht abrufbar</p>
  {/if}
  <PlacePicker enabled={status?.enabled ?? false} onchange={changed} />
  {#if overwrite}
    <p data-testid="overwrite-prompt">{overwrite.located} von {overwrite.count} ausgewählten Fotos haben bereits einen Ort.</p>
  {/if}
  {#if error}
    <p role="alert">{error}</p>
  {/if}
  {#snippet actions(close)}
    <button type="button" onclick={close}>Abbrechen</button>
    {#if overwrite}
      <button type="button" disabled={busy} onclick={() => guarded(() => run('missing'))}>Nur Fotos ohne Ort</button>
      <button type="button" disabled={busy} onclick={() => guarded(() => run('all'))}>Alle überschreiben</button>
    {:else}
      <button type="button" disabled={!ready || busy} onclick={apply}>Übernehmen</button>
    {/if}
  {/snippet}
</Dialog>
