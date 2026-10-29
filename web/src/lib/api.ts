export class ApiError extends Error {
  constructor(
    public status: number,
    public body: { error?: string } | null,
  ) {
    super(`request failed with status ${status}`);
  }
}

export interface ApiOptions {
  method?: string;
  body?: unknown;
  /** Login requests handle their own 401 instead of redirecting. */
  allowUnauthorized?: boolean;
  fetchFn?: typeof fetch;
  redirect?: (path: string) => void;
}

export async function api<T = unknown>(path: string, opts: ApiOptions = {}): Promise<T> {
  const fetchFn = opts.fetchFn ?? fetch;
  const headers: Record<string, string> = { 'X-Requested-With': 'fetch' };
  const init: RequestInit = { method: opts.method ?? 'GET', headers };
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }
  const response = await fetchFn(path, init);
  if (response.status === 401 && !opts.allowUnauthorized) {
    (opts.redirect ?? ((p) => location.assign(p)))('/login');
  }
  const text = await response.text();
  let body: unknown;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = null;
  }
  if (!response.ok) {
    throw new ApiError(response.status, body as { error?: string } | null);
  }
  return body as T;
}
