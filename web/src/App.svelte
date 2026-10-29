<script lang="ts">
  import { fetchHealth } from './lib/health';
  import AppHeader from './lib/AppHeader.svelte';
  import Login from './routes/Login.svelte';

  const isLogin = location.pathname === '/login';
  const health = isLogin ? null : fetchHealth();
</script>

{#if isLogin}
  <Login />
{:else}
  <AppHeader />
  <main>
    <h1>PhotoCurator</h1>
    {#await health}
      <p></p>
    {:then h}
      <p>Version {h?.version}</p>
    {:catch}
      <p role="alert">Server nicht erreichbar</p>
    {/await}
  </main>
{/if}
