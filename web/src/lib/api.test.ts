import { describe, expect, it } from 'vitest';
import { api, ApiError } from './api';

describe('api', () => {
  it('sends the X-Requested-With header on every request', async () => {
    let seen: Headers | undefined;
    const fetchFn = (async (_: unknown, init: RequestInit) => {
      seen = new Headers(init.headers);
      return new Response('{"username":"anna"}', { status: 200 });
    }) as unknown as typeof fetch;
    await api('/api/me', { fetchFn });
    expect(seen?.get('X-Requested-With')).toBe('fetch');
  });

  it('redirects to the login page on 401 outside the login call', async () => {
    const fetchFn = (async () => new Response('', { status: 401 })) as typeof fetch;
    const paths: string[] = [];
    await expect(api('/api/me', { fetchFn, redirect: (p) => paths.push(p) })).rejects.toBeInstanceOf(ApiError);
    expect(paths).toEqual(['/login']);
  });

  it('does not redirect on 401 when unauthorized is allowed', async () => {
    const fetchFn = (async () => new Response('{"error":"x"}', { status: 401 })) as typeof fetch;
    const paths: string[] = [];
    await expect(
      api('/api/login', { fetchFn, allowUnauthorized: true, redirect: (p) => paths.push(p) }),
    ).rejects.toMatchObject({ body: { error: 'x' } });
    expect(paths).toEqual([]);
  });
});
