import { afterEach, describe, expect, it, vi } from 'vitest';
import { loadThumb } from './thumbLoader';

const pending = () => new Response('{"state":"pending"}', { status: 202, headers: { 'Retry-After': '1' } });

function sequence(...responses: (() => Response)[]) {
  let i = 0;
  const calls: string[] = [];
  const fetchFn = (async (url: string) => {
    calls.push(url);
    return responses[Math.min(i++, responses.length - 1)]();
  }) as unknown as typeof fetch;
  return { fetchFn, calls };
}

afterEach(() => vi.restoreAllMocks());

describe('loadThumb', () => {
  it('a finished thumbnail resolves to a blob url', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:x');
    const { fetchFn } = sequence(() => new Response('img', { status: 200 }));
    expect(await loadThumb('/api/thumbs/1/s?k=a', { fetchFn })).toEqual({ state: 'loaded', blobUrl: 'blob:x' });
  });

  it('a pending thumbnail retries until it is ready', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:y');
    const { fetchFn, calls } = sequence(pending, pending, () => new Response('img', { status: 200 }));
    const delays: number[] = [];
    const sleep = async (ms: number) => void delays.push(ms);
    const result = await loadThumb('/t', { fetchFn, sleep });
    expect(result).toEqual({ state: 'loaded', blobUrl: 'blob:y' });
    expect(calls).toHaveLength(3);
    expect(delays).toEqual([1000, 1500]);
  });

  it('retry delays never exceed five seconds', async () => {
    const { fetchFn } = sequence(...Array(12).fill(pending), () => new Response('', { status: 422 }));
    const delays: number[] = [];
    await loadThumb('/t', { fetchFn, sleep: async (ms) => void delays.push(ms) });
    expect(Math.max(...delays)).toBe(5000);
  });

  it('an unrenderable file resolves to failed', async () => {
    const { fetchFn, calls } = sequence(() => new Response('{"state":"failed"}', { status: 422 }));
    expect(await loadThumb('/t', { fetchFn })).toEqual({ state: 'failed' });
    expect(calls).toHaveLength(1);
  });

  it('aborting stops the retries', async () => {
    const ctrl = new AbortController();
    const { fetchFn, calls } = sequence(pending);
    const sleep = async () => ctrl.abort();
    const result = await loadThumb('/t', { fetchFn, sleep, signal: ctrl.signal });
    expect(result).toEqual({ state: 'pending' });
    expect(calls).toHaveLength(1);
  });

  it('redirects to the login page on 401', async () => {
    const { fetchFn } = sequence(() => new Response('', { status: 401 }));
    const paths: string[] = [];
    const result = await loadThumb('/t', { fetchFn, redirect: (p) => paths.push(p) });
    expect(paths).toEqual(['/login']);
    expect(result).toEqual({ state: 'failed' });
  });
});
