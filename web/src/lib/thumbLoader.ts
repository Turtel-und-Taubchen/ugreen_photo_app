export type ThumbResult = { state: 'pending' } | { state: 'loaded'; blobUrl: string } | { state: 'failed' };

export interface LoadThumbOptions {
  fetchFn?: typeof fetch;
  signal?: AbortSignal;
  sleep?: (ms: number, signal?: AbortSignal) => Promise<void>;
  redirect?: (path: string) => void;
}

const MAX_DELAY_MS = 5000;

function abortableSleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener(
      'abort',
      () => {
        clearTimeout(timer);
        resolve();
      },
      { once: true },
    );
  });
}

// Renders are queued server-side, so a 202 is polled with a growing delay instead of hammering it.
function retryDelay(response: Response, attempt: number): number {
  const seconds = Number(response.headers.get('Retry-After'));
  const base = (Number.isFinite(seconds) && seconds > 0 ? seconds : 1) * 1000;
  return Math.min(base * 1.5 ** attempt, MAX_DELAY_MS);
}

/**
 * Resolves with `loaded`/`failed`, or with `pending` when aborted before the thumbnail was
 * ready (then nothing was allocated and the caller has nothing to revoke).
 */
export async function loadThumb(url: string, opts: LoadThumbOptions = {}): Promise<ThumbResult> {
  const fetchFn = opts.fetchFn ?? fetch;
  const sleep = opts.sleep ?? abortableSleep;
  const { signal } = opts;
  for (let attempt = 0; ; attempt++) {
    if (signal?.aborted) return { state: 'pending' };
    let response: Response;
    try {
      response = await fetchFn(url, { signal, headers: { 'X-Requested-With': 'fetch' } });
    } catch {
      return signal?.aborted ? { state: 'pending' } : { state: 'failed' };
    }
    if (response.status === 200) {
      let blob: Blob;
      try {
        blob = await response.blob();
      } catch {
        return signal?.aborted ? { state: 'pending' } : { state: 'failed' };
      }
      if (signal?.aborted) return { state: 'pending' };
      return { state: 'loaded', blobUrl: URL.createObjectURL(blob) };
    }
    if (response.status === 202) {
      await sleep(retryDelay(response, attempt), signal);
      continue;
    }
    if (response.status === 401) (opts.redirect ?? ((p) => location.assign(p)))('/login');
    return { state: 'failed' };
  }
}
