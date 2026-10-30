<script lang="ts">
  import { api } from './api';
  import { fetchHealth } from './health';
  import JobStatus from './JobStatus.svelte';

  const health = fetchHealth();
  const me = api<{ username: string }>('/api/me');

  async function logout() {
    try {
      await api('/api/logout', { method: 'POST' });
    } finally {
      location.assign('/login');
    }
  }
</script>

<header>
  <h1>PhotoCurator</h1>
  {#await health}
    <small></small>
  {:then h}
    <small>Version {h?.version}</small>
  {:catch}
    <small role="alert">Server nicht erreichbar</small>
  {/await}
  {#await me then user}
    <span data-testid="current-user">{user.username}</span>
  {:catch}
    <span></span>
  {/await}
  <nav>
    <a href="/">Fotos</a>
    <a href="/tags">Tags</a>
    <a href="/orte">Ortssuche</a>
  </nav>
  <button type="button" onclick={logout}>Abmelden</button>
  <JobStatus />
</header>

<style>
  h1 { margin: 0; font-size: 1.25rem; }
  header { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 0.5rem; }
  nav { display: flex; gap: 0.25rem; }
  a { display: inline-flex; align-items: center; min-height: 44px; min-width: 44px; padding: 0 0.5rem; box-sizing: border-box; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
