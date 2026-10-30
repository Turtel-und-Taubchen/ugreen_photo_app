<script lang="ts">
  import { fetchGeoStatus, geoNotice, type GeoStatus } from '../lib/geo';
  import PlacePicker from '../lib/PlacePicker.svelte';

  let status = $state<GeoStatus | null>(null);
  let loadFailed = $state(false);

  fetchGeoStatus().then(
    (s) => (status = s),
    () => (loadFailed = true),
  );

  const notice = $derived(geoNotice(status));
</script>

<main>
  <h2>Ortssuche</h2>
  {#if notice}
    <p>{notice}</p>
  {/if}
  {#if loadFailed}
    <p role="alert">Status der Ortssuche nicht abrufbar</p>
  {/if}
  <PlacePicker enabled={status?.enabled ?? false} />
</main>

<style>
  main { flex: 1; min-height: 0; overflow-y: auto; }
</style>
