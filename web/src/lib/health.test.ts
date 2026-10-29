import { describe, expect, it } from 'vitest';
import { fetchHealth } from './health';

function fakeFetch(status: number, body: unknown): typeof fetch {
  return (async () => new Response(JSON.stringify(body), { status })) as typeof fetch;
}

describe('fetchHealth', () => {
  it('reads status and version from the health endpoint', async () => {
    const result = await fetchHealth(fakeFetch(200, { status: 'ok', version: '1.2.3' }));
    expect(result).toEqual({ status: 'ok', version: '1.2.3' });
  });

  it('rejects a non-200 health response', async () => {
    await expect(fetchHealth(fakeFetch(503, {}))).rejects.toThrow();
  });
});
