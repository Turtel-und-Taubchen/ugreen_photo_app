<script lang="ts">
  import AppHeader from './lib/AppHeader.svelte';
  import Library from './routes/Library.svelte';
  import Login from './routes/Login.svelte';
  import Places from './routes/Places.svelte';
  import Tags from './routes/Tags.svelte';

  const isLogin = location.pathname === '/login';
  const isTags = location.pathname === '/tags';
  const isPlaces = location.pathname === '/orte';
</script>

{#if isLogin}
  <Login />
{:else}
  <div class="shell">
    <AppHeader />
    {#if isTags}
      <Tags />
    {:else if isPlaces}
      <Places />
    {:else}
      <Library />
    {/if}
  </div>
{/if}

<style>
  .shell { display: flex; flex-direction: column; height: 100dvh; box-sizing: border-box; padding: 0 8px; overflow: hidden; }
</style>
