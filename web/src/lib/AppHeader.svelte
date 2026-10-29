<script lang="ts">
  import { api } from './api';

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
  <strong>PhotoCurator</strong>
  {#await me then user}
    <span data-testid="current-user">{user.username}</span>
  {:catch}
    <span></span>
  {/await}
  <button type="button" onclick={logout}>Abmelden</button>
</header>

<style>
  header { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 0.5rem; }
  button { min-height: 44px; min-width: 44px; padding: 0 0.75rem; font-size: 1rem; }
</style>
