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
  import { errorMessage } from './api';
  import PlaceMap from './map/PlaceMap.svelte';
  import { createFavorite, deleteFavorite, listFavorites, renameFavorite, sortFavorites, type Favorite } from './places';
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

  let favorites = $state<Favorite[]>([]);
  let favoritesLoaded = $state(false);
  let favoritesError = $state('');
  let favoriteError = $state('');
  let renaming = $state<number | null>(null);
  let renameValue = $state('');
  let deleting = $state<number | null>(null);
  let saving = $state(false);
  let saveName = $state('');

  $effect(() => {
    let alive = true;
    listFavorites().then(
      (list) => {
        if (!alive) return;
        favorites = list;
        favoritesLoaded = true;
      },
      (e) => {
        if (alive) favoritesError = errorMessage(e);
      },
    );
    return () => {
      alive = false;
    };
  });

  const point = $derived.by(() => {
    const la = parseCoordinate(lat);
    const lo = parseCoordinate(lon);
    return la !== null && lo !== null && Math.abs(la) <= 90 && Math.abs(lo) <= 180 ? { lat: la, lon: lo } : null;
  });

  function emit() {
    onchange?.(point ? { ...point, name, parts: { ...parts } } : null);
  }

  async function favoriteAction(run: () => Promise<void>): Promise<boolean> {
    favoriteError = '';
    try {
      await run();
      return true;
    } catch (e) {
      favoriteError = errorMessage(e);
      return false;
    }
  }

  function startRename(f: Favorite) {
    renaming = f.id;
    renameValue = f.name;
    deleting = null;
    favoriteError = '';
  }

  async function commitRename(event: SubmitEvent, f: Favorite) {
    event.preventDefault();
    const ok = await favoriteAction(async () => {
      const saved = await renameFavorite(f.id, renameValue);
      favorites = sortFavorites(favorites.map((x) => (x.id === saved.id ? saved : x)));
    });
    if (ok) renaming = null;
  }

  async function confirmDelete(f: Favorite) {
    const ok = await favoriteAction(async () => {
      await deleteFavorite(f.id);
      favorites = favorites.filter((x) => x.id !== f.id);
    });
    if (ok) deleting = null;
  }

  function startSave() {
    saveName = name;
    saving = true;
    favoriteError = '';
  }

  async function commitSave(event: SubmitEvent) {
    event.preventDefault();
    if (!point) return;
    const target = point;
    const ok = await favoriteAction(async () => {
      const saved = await createFavorite({ name: saveName, lat: target.lat, lon: target.lon, parts: { ...parts } });
      favorites = sortFavorites([...favorites, saved]);
    });
    if (ok) saving = false;
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
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }

  function apply(next: PickedPlace) {
    lat = String(next.lat);
    lon = String(next.lon);
    name = next.name;
    parts = { ...next.parts };
    emit();
  }

  // A map click only moves the point; name and parts describe the previous place until the user decides.
  function pick(p: { lat: number; lon: number }) {
    lat = String(p.lat);
    lon = String(p.lon);
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
      error = errorMessage(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="picker">
  <section aria-labelledby="favorites-title">
    <h3 id="favorites-title">Lieblingsorte</h3>
    {#if favoritesError}
      <p role="alert">{favoritesError}</p>
    {/if}
    {#if favoriteError}
      <p role="alert">{favoriteError}</p>
    {/if}
    {#if favoritesLoaded && favorites.length === 0}
      <p>Noch keine Lieblingsorte.</p>
    {/if}
    <ul role="list" class="favorites" aria-label="Lieblingsorte">
      {#each favorites as f (f.id)}
        <li>
          {#if renaming === f.id}
            <form class="field" onsubmit={(e) => commitRename(e, f)}>
              <label class="sr-label" for="favorite-rename-{f.id}">Neuer Name</label>
              <input id="favorite-rename-{f.id}" type="text" bind:value={renameValue} autocomplete="off" />
              <button type="submit">Speichern</button>
              <button type="button" onclick={() => (renaming = null)}>Abbrechen</button>
            </form>
          {:else}
            <div class="favorite">
              <button type="button" class="hit" onclick={() => apply({ lat: f.lat, lon: f.lon, name: f.name, parts: f.parts })}>
                <strong>{f.name}</strong>
                {#if f.parts.city}<small>{f.parts.city}</small>{/if}
              </button>
              {#if deleting === f.id}
                <button type="button" aria-label="„{f.name}“ endgültig löschen" onclick={() => confirmDelete(f)}>Wirklich löschen</button>
                <button type="button" onclick={() => (deleting = null)}>Abbrechen</button>
              {:else}
                <button type="button" aria-label="„{f.name}“ umbenennen" onclick={() => startRename(f)}>Umbenennen</button>
                <button type="button" aria-label="„{f.name}“ löschen" onclick={() => { deleting = f.id; favoriteError = ''; }}>Löschen</button>
              {/if}
            </div>
          {/if}
        </li>
      {/each}
    </ul>
  </section>

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
        <button type="button" class="hit" onclick={() => apply({ lat: place.lat, lon: place.lon, name: place.label, parts: place.parts })}>
          <strong>{place.name}</strong>
          <span>{place.address}</span>
          <small>{place.kind}</small>
        </button>
      </li>
    {/each}
  </ul>

  <section aria-labelledby="chosen-title">
    <h3 id="chosen-title">Gewählter Ort</h3>
    <PlaceMap {point} onpick={pick} />
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
    {#if saving}
      <form class="field" onsubmit={commitSave}>
        <label class="sr-label" for="favorite-name">Name des Lieblingsorts</label>
        <input id="favorite-name" type="text" bind:value={saveName} autocomplete="off" />
        <button type="submit">Speichern</button>
        <button type="button" onclick={() => (saving = false)}>Abbrechen</button>
      </form>
    {:else}
      <button type="button" onclick={startSave} disabled={!point}>Als Lieblingsort speichern</button>
    {/if}
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
  .favorites { list-style: none; margin: 0.5rem 0; padding: 0; display: flex; flex-direction: column; gap: 0.25rem; }
  .favorite { display: flex; flex-wrap: wrap; gap: 0.25rem; align-items: center; }
  .favorite .hit { flex: 1 1 10rem; width: auto; }
  .sr-label { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 0.25rem 0.75rem; margin: 0.75rem 0; }
  dd { margin: 0; }
</style>
