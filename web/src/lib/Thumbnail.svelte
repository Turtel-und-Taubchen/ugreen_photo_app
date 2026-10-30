<script lang="ts">
  import { loadThumb } from './thumbLoader';

  interface Props {
    id: number | string;
    mediaType: string;
    thumbKey: string;
    size?: 's' | 'l';
    /** File name, used as the alt text of the image. */
    label: string;
  }
  let { id, mediaType, thumbKey, size = 's', label }: Props = $props();

  let phase = $state<'pending' | 'loaded' | 'failed'>('pending');
  let src = $state('');
  let tile: HTMLElement;

  const url = $derived(`/api/thumbs/${id}/${size}?k=${encodeURIComponent(thumbKey)}`);

  // Loads only while the tile is on screen; leaving it aborts the request and frees the blob.
  $effect(() => {
    const target = url;
    let ctrl: AbortController | undefined;
    let blobUrl = '';
    const stop = () => {
      ctrl?.abort();
      ctrl = undefined;
      if (blobUrl) URL.revokeObjectURL(blobUrl);
      blobUrl = '';
      src = '';
      phase = 'pending';
    };
    const start = async () => {
      const own = new AbortController();
      ctrl = own;
      const result = await loadThumb(target, { signal: own.signal });
      if (own.signal.aborted) return;
      if (result.state === 'loaded') {
        blobUrl = result.blobUrl;
        src = result.blobUrl;
      }
      if (result.state !== 'pending') phase = result.state;
    };
    const observer = new IntersectionObserver((entries) => {
      const visible = entries[entries.length - 1].isIntersecting;
      if (visible && !ctrl) void start();
      else if (!visible && ctrl) stop();
    });
    observer.observe(tile);
    return () => {
      observer.disconnect();
      stop();
    };
  });
</script>

<div class="tile" bind:this={tile}>
  {#if phase === 'loaded'}
    <img data-state="loaded" {src} alt={label} />
  {:else if phase === 'failed'}
    <div class="ph failed" data-state="failed" role="img" aria-label="Vorschau nicht verfügbar">Vorschau nicht verfügbar</div>
  {:else}
    <div class="ph" data-state="pending" role="img" aria-label="Vorschau wird erzeugt"></div>
  {/if}
  {#if mediaType === 'video'}
    <span class="video" role="img" aria-label="Video">▶</span>
  {/if}
</div>

<style>
  .tile { position: relative; aspect-ratio: 1; background: #eee; overflow: hidden; }
  img, .ph { width: 100%; height: 100%; display: block; }
  img { object-fit: cover; }
  .ph { display: flex; align-items: center; justify-content: center; text-align: center; font-size: 0.75rem; color: #555; }
  .failed { color: #b00020; }
  .video { position: absolute; right: 4px; bottom: 4px; padding: 2px 6px; border-radius: 4px; background: rgba(0, 0, 0, 0.6); color: #fff; font-size: 0.875rem; }
</style>
