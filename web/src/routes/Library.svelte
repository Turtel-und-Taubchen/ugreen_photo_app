<script lang="ts">
  import ScanStatus from '../lib/ScanStatus.svelte';
  import PhotoDetail from '../lib/detail/PhotoDetail.svelte';
  import PhotoGrid from '../lib/grid/PhotoGrid.svelte';
  import { isRunning } from '../lib/scan';

  let scanRunning = $state(false);
  let openId = $state<number | null>(null);
</script>

<ScanStatus onStatus={(s) => (scanRunning = isRunning(s))} />
<PhotoGrid onOpen={(id) => (openId = id)} {scanRunning} />
{#if openId !== null}
  <PhotoDetail id={openId} onClose={() => (openId = null)} onStep={(id) => (openId = id)} />
{/if}
