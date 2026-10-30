<script lang="ts">
  import AppHeader from './lib/AppHeader.svelte';
  import Library from './routes/Library.svelte';
  import Login from './routes/Login.svelte';

  const isLogin = location.pathname === '/login';
</script>

{#if isLogin}
  <Login />
{:else}
  <div class="shell">
    <AppHeader />
    <Library />
  </div>
{/if}

<style>
  .shell { display: flex; flex-direction: column; height: 100dvh; box-sizing: border-box; padding: 0 8px; overflow: hidden; }
</style>
