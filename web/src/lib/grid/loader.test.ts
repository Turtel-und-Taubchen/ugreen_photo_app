import { describe, expect, it } from 'vitest';
import { createPhotoLoader } from './loader';

const idOf = (month: string, index: number) => Number(month.replace('-', '')) * 100000 + index;

interface FakeServerOptions {
  /** Requests wait until `release()` lets them through, oldest first. */
  gated?: boolean;
  fail?: (month: string, after: string | null) => boolean;
  /** Each page repeats the last photo of the previous one, as a shifted keyset would. */
  overlap?: boolean;
}

// A fake `/api/photos`: photo `i` of a month has the cursor `String(i)`.
function fakeServer(counts: Record<string, number>, opts: FakeServerOptions = {}) {
  const calls: { month: string; after: string | null; limit: number }[] = [];
  const gates: (() => void)[] = [];
  const respond = (status: number, body: unknown) =>
    ({ status, ok: status < 400, text: async () => JSON.stringify(body) }) as unknown as Response;
  const fetchFn = (async (path: string) => {
    const url = new URL(path, 'http://test');
    const month = url.searchParams.get('month')!;
    const after = url.searchParams.get('after');
    const limit = Number(url.searchParams.get('limit'));
    calls.push({ month, after, limit });
    if (opts.gated) await new Promise<void>((resolve) => gates.push(resolve));
    if (opts.fail?.(month, after)) return respond(500, { error: 'boom' });
    const from = after === null ? 0 : Number(after) + (opts.overlap ? 0 : 1);
    const to = Math.min(from + limit, counts[month]);
    const items = [];
    for (let i = from; i < to; i++) {
      items.push({ id: idOf(month, i), relPath: `${month}/${i}.jpg`, mediaType: 'image', takenAt: null, thumbKey: 'k' });
    }
    return respond(200, { items, next: to < counts[month] ? String(to - 1) : null });
  }) as unknown as typeof fetch;
  const release = () => gates.shift()!();
  return { fetchFn, calls, release };
}

// Settles every pending promise chain without touching the clock.
const flush = async () => {
  for (let i = 0; i < 50; i++) await Promise.resolve();
};

describe('photo loader', () => {
  it('scrolling deep into a month loads its pages in cursor order, once each', async () => {
    const server = fakeServer({ '2024-07': 450 });
    const loader = createPhotoLoader({ fetchFn: server.fetchFn });
    await Promise.all([loader.ensure('2024-07', 10), loader.ensure('2024-07', 420), loader.ensure('2024-07', 300)]);
    expect(server.calls).toEqual([
      { month: '2024-07', after: null, limit: 200 },
      { month: '2024-07', after: '199', limit: 200 },
      { month: '2024-07', after: '399', limit: 200 },
    ]);
    expect(loader.loaded('2024-07')).toBe(450);
    expect(loader.get('2024-07', 0)?.id).toBe(idOf('2024-07', 0));
    expect(loader.get('2024-07', 420)?.id).toBe(idOf('2024-07', 420));
    expect(loader.get('2024-07', 449)?.id).toBe(idOf('2024-07', 449));

    await loader.ensure('2024-07', 449);
    await loader.ensure('2024-07', 10_000);
    expect(server.calls).toHaveLength(3);
  });

  it('an index already loaded needs no request; the next one fetches exactly one page', async () => {
    const server = fakeServer({ '2024-07': 450 });
    const loader = createPhotoLoader({ fetchFn: server.fetchFn });
    await loader.ensure('2024-07', 0);
    await loader.ensure('2024-07', 199);
    expect(server.calls).toHaveLength(1);
    await loader.ensure('2024-07', 200);
    expect(server.calls.map((c) => c.after)).toEqual([null, '199']);
    expect(loader.loaded('2024-07')).toBe(400);
  });

  it('a failed page leaves placeholders and is retried on the next ensure', async () => {
    let failures = 1;
    const fail = (_: string, after: string | null) => after === '199' && failures-- > 0;
    const server = fakeServer({ '2024-07': 450 }, { fail });
    const changes: string[] = [];
    const loader = createPhotoLoader({ fetchFn: server.fetchFn, onChange: (m) => changes.push(m) });

    await expect(loader.ensure('2024-07', 250)).resolves.toBeUndefined();
    expect(loader.loaded('2024-07')).toBe(200);
    expect(loader.get('2024-07', 250)).toBeUndefined();
    expect(loader.failed('2024-07')).toBe(true);
    expect(changes).toHaveLength(2);

    await loader.ensure('2024-07', 250);
    expect(server.calls.map((c) => c.after)).toEqual([null, '199', '199']);
    expect(loader.loaded('2024-07')).toBe(400);
    expect(loader.failed('2024-07')).toBe(false);
  });

  it('at most two months load in parallel, with one request in flight per month', async () => {
    const server = fakeServer({ '2024-07': 10, '2024-06': 10, '2024-05': 10, '2024-04': 10 }, { gated: true });
    const loader = createPhotoLoader({ fetchFn: server.fetchFn });
    const done = [
      loader.ensure('2024-07', 5),
      loader.ensure('2024-07', 9),
      loader.ensure('2024-06', 5),
      loader.ensure('2024-05', 5),
      loader.ensure('2024-04', 5),
    ];
    await flush();
    expect(server.calls.map((c) => c.month)).toEqual(['2024-07', '2024-06']);

    // The month requested last is what the user looks at now, so it takes the freed slot.
    server.release();
    await flush();
    expect(server.calls.map((c) => c.month)).toEqual(['2024-07', '2024-06', '2024-04']);

    server.release();
    await flush();
    expect(server.calls.map((c) => c.month)).toEqual(['2024-07', '2024-06', '2024-04', '2024-05']);
    server.release();
    server.release();
    await Promise.all(done);
    for (const m of ['2024-07', '2024-06', '2024-05', '2024-04']) expect(loader.loaded(m)).toBe(10);
  });

  it('pages overlapping by a photo do not duplicate it', async () => {
    const server = fakeServer({ '2024-07': 250 }, { overlap: true });
    const loader = createPhotoLoader({ fetchFn: server.fetchFn });
    await loader.ensure('2024-07', 249);
    expect(loader.loaded('2024-07')).toBe(250);
    const ids = Array.from({ length: 250 }, (_, i) => loader.get('2024-07', i)?.id);
    expect(ids).toEqual(Array.from({ length: 250 }, (_, i) => idOf('2024-07', i)));
  });

  it('reset drops the cache and discards responses still in flight', async () => {
    const server = fakeServer({ '2024-07': 10 }, { gated: true });
    const loader = createPhotoLoader({ fetchFn: server.fetchFn });
    const stale = loader.ensure('2024-07', 5);
    await flush();
    loader.reset();
    server.release();
    await stale;
    expect(loader.loaded('2024-07')).toBe(0);

    const fresh = loader.ensure('2024-07', 5);
    await flush();
    server.release();
    await fresh;
    expect(server.calls).toHaveLength(2);
    expect(loader.loaded('2024-07')).toBe(10);
  });
});
