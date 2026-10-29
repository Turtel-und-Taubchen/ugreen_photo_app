<script lang="ts">
  import { fetchHealth } from './lib/health';

  const health = fetchHealth();
</script>

<main>
  <h1>PhotoCurator</h1>
  {#await health}
    <p></p>
  {:then h}
    <p>Version {h.version}</p>
  {:catch}
    <p role="alert">Server nicht erreichbar</p>
  {/await}
</main>
