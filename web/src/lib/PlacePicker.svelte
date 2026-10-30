<script lang="ts" module>
  import type { PlaceParts } from './geo';

  export interface PickedPlace {
    lat: number;
    lon: number;
    name: string;
    parts: PlaceParts;
  }
</script>

<script lang="ts">
  import { ApiError } from './api';
  import { emptyParts, parseCoordinate, reversePlace, searchPlaces, type Place } from './geo';

  let {
    enabled = true,
    onchange,
  }: { enabled?: boolean; onchange?: (place: PickedPlace | null) => void } = $props();

  let query = $state('');
  let results = $state<Place[]>([]);
  let searched = $state(false);
  let busy = $state(false);
  let error = $state('');
  let lat = $state('');
  let lon = $state('');
  let name = $state('');
  let parts = $state<PlaceParts>({ ...emptyParts });

  const point = $derived.by(() => {
    const la = parseCoordinate(lat);
    const lo = parseCoordinate(lon);
    return la !== null && lo !== null && Math.abs(la) <= 90 && Math.abs(lo) <= 180 ? { lat: la, lon: lo } : null;
  });

  function emit() {
    onchange?.(point ? { ...point, name, parts: { ...parts } } : null);
  }

  function message(e: unknown): string {
    return e instanceof ApiError && e.body?.error ? e.body.error : 'Anfrage fehlgeschlagen';
  }

  async function search(event: SubmitEvent) {
    event.preventDefault();
    if (!enabled || busy || query.trim() === '') return;
    busy = true;
    error = '';
    try {
      results = (await searchPlaces(query)).slice(0, 10);
      searched = true;
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }

  function choose(place: Place) {
    lat = String(place.lat);
    lon = String(place.lon);
    name = place.label;
    parts = { ...place.parts };
    emit();
  }

  async function reverse() {
    if (!enabled || busy || !point) return;
    busy = true;
    error = '';
    try {
      const place = await reversePlace(point.lat, point.lon);
      if (place) {
        name = place.label;
        parts = { ...place.parts };
      } else {
        error = 'Zu diesem Punkt wurde kein Ort gefunden.';
      }
      emit();
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="picker">
  <form onsubmit={search}>
    <label for="place-query">Ort suchen</label>
    <div class="field">
      <input id="place-query" type="text" bind:value={query} autocomplete="off" disabled={!enabled} />
      <button type="submit" disabled={!enabled || busy}>Suchen</button>
    </div>
  </form>

  {#if error}
    <p role="alert">{error}</p>
  {/if}

  {#if searched && results.length === 0}
    <p>Keine Treffer.</p>
  {/if}
  <ul role="list" class="results" aria-label="Treffer">
    {#each results as place, i (i)}
      <li>
        <button type="button" class="hit" onclick={() => choose(place)}>
          <strong>{place.name}</strong>
          <span>{place.address}</span>
          <small>{place.kind}</small>
        </button>
      </li>
    {/each}
  </ul>

  <section aria-labelledby="chosen-title">
    <h3 id="chosen-title">Gewählter Ort</h3>
    <div class="coords">
      <div>
        <label for="place-lat">Breite</label>
        <input id="place-lat" type="text" inputmode="decimal" bind:value={lat} oninput={emit} autocomplete="off" />
      </div>
      <div>
        <label for="place-lon">Länge</label>
        <input id="place-lon" type="text" inputmode="decimal" bind:value={lon} oninput={emit} autocomplete="off" />
      </div>
    </div>
    <button type="button" onclick={reverse} disabled={!enabled || busy || !point}>Ort zum Punkt ermitteln</button>
    <label for="place-name">Ortsname</label>
    <input id="place-name" type="text" bind:value={name} oninput={emit} autocomplete="off" />
    <dl>
      <dt>Sublocation</dt><dd>{parts.sublocation}</dd>
      <dt>City</dt><dd>{parts.city}</dd>
      <dt>State</dt><dd>{parts.state}</dd>
      <dt>Country</dt><dd>{parts.country}</dd>
    </dl>
    {#if point}
      <p data-testid="chosen-summary">{name || 'Ohne Namen'} ({point.lat}, {point.lon})</p>
    {/if}
  </section>
</div>

<style>
  .field, .coords { display: flex; gap: 0.5rem; }
  .coords > div { flex: 1; min-width: 0; }
  input { width: 100%; min-width: 0; min-height: 44px; padding: 0 0.5rem; font-size: 1rem; box-sizing: border-box; }
  .field input { flex: 1; width: auto; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
  label { display: block; margin: 0.5rem 0 0.25rem; }
  .results { list-style: none; margin: 0.5rem 0; padding: 0; display: flex; flex-direction: column; gap: 0.25rem; }
  .hit { display: flex; flex-direction: column; align-items: flex-start; width: 100%; text-align: left; }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 0.25rem 0.75rem; margin: 0.75rem 0; }
  dd { margin: 0; }
</style>
