<script lang="ts">
  import { onDestroy, onMount } from 'svelte';

  interface LatLon {
    lat: number;
    lon: number;
  }
  // Declared locally: importing from ./leaflet, even as a type, must not couple this file to the library.
  interface MapHandle {
    setMarker(p: LatLon | null): void;
    focus(p: LatLon, zoom: number): void;
    remove(): void;
  }

  let {
    point,
    onpick,
    height = 240,
  }: { point: LatLon | null; onpick?: (p: LatLon) => void; height?: number } = $props();

  let el: HTMLDivElement;
  let handle: MapHandle | null = null;
  let destroyed = false;
  // The last position the user picked on the map; echoing it back must not recentre the view.
  let lastPick: LatLon | null = null;

  onMount(async () => {
    const mod = await import('./leaflet');
    if (destroyed) return;
    handle = mod.createMap(el, {
      point,
      onpick: onpick
        ? (p) => {
            lastPick = p;
            onpick?.(p);
          }
        : undefined,
    });
  });

  $effect(() => {
    const p = point;
    if (!handle) return;
    if (!p) {
      handle.setMarker(null);
      return;
    }
    const own = lastPick && lastPick.lat === p.lat && lastPick.lon === p.lon;
    // Only the point change caused by the own click is skipped; any later equal point is external.
    lastPick = null;
    if (own) return;
    handle.focus(p, 15);
  });

  onDestroy(() => {
    destroyed = true;
    handle?.remove();
    handle = null;
  });
</script>

<div class="wrap">
  <div bind:this={el} class="map" role="region" aria-label="Karte" style:height="{height}px"></div>
  <p class="attribution">
    <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer"
      >© OpenStreetMap-Mitwirkende</a
    >
  </p>
</div>

<style>
  .map { width: 100%; z-index: 0; }
  .attribution { margin: 0; font-size: 0.875rem; }
  .attribution a { display: inline-flex; align-items: center; min-height: 44px; color: #0b57d0; }
  .map :global(.pc-marker) {
    width: 24px;
    height: 24px;
    box-sizing: border-box;
    border-radius: 50%;
    background: #d93025;
    border: 3px solid #fff;
    box-shadow: 0 0 0 1px #5f0f0a;
  }
</style>
