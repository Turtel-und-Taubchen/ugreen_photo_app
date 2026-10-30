<script lang="ts">
  import { api, errorMessage } from '../lib/api';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      await api('/api/login', { method: 'POST', body: { username, password }, allowUnauthorized: true });
      location.assign('/');
    } catch (e) {
      error = errorMessage(e, 'Anmeldung fehlgeschlagen.');
      busy = false;
    }
  }
</script>

<main>
  <h1>PhotoCurator</h1>
  <form onsubmit={submit}>
    <label>
      Name
      <input name="username" type="text" autocapitalize="off" autocorrect="off" autocomplete="username" required bind:value={username} />
    </label>
    <label>
      Passwort
      <input name="password" type="password" autocomplete="current-password" required bind:value={password} />
    </label>
    {#if error}
      <p role="alert">{error}</p>
    {/if}
    <button type="submit" disabled={busy}>Anmelden</button>
  </form>
</main>

<style>
  form { display: flex; flex-direction: column; gap: 1rem; max-width: 24rem; }
  label { display: flex; flex-direction: column; gap: 0.25rem; }
  input, button { min-height: 44px; font-size: 1rem; box-sizing: border-box; padding: 0 0.75rem; }
</style>
