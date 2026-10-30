import fs from 'node:fs/promises';
import http from 'node:http';
import { fileURLToPath } from 'node:url';
import { NOMINATIM_MOCK_PORT } from './compose';

const FIXTURES = fileURLToPath(new URL('../../../testdata/nominatim/', import.meta.url));

export interface MockRequest {
  path: string;
  query: Record<string, string>;
  userAgent: string;
  at: number;
}

const SEARCH_FIXTURES: Record<string, string> = {
  'ritzenhofer saalfelden': 'search-ritzenhofer-saalfelden.json',
  saalfelden: 'search-saalfelden.json',
};

let server: http.Server | undefined;

function send(res: http.ServerResponse, status: number, body: string): void {
  res.writeHead(status, { 'content-type': 'application/json' });
  res.end(body);
}

/** Fixtures are read per request so a test run never needs a restart to pick up edits. */
async function fixture(name: string | undefined): Promise<string> {
  return name ? fs.readFile(FIXTURES + name, 'utf8') : '[]';
}

export async function startNominatimMock(): Promise<void> {
  let mode: 'ok' | 'error' = 'ok';
  let log: MockRequest[] = [];

  const srv = http.createServer(async (req, res) => {
    try {
      const url = new URL(req.url ?? '/', 'http://mock');
      const method = req.method ?? 'GET';
      if (method === 'POST' && url.pathname === '/__mock/reset') {
        log = [];
        mode = 'ok';
        return send(res, 200, '{}');
      }
      if (method === 'POST' && url.pathname === '/__mock/mode') {
        const chunks: Buffer[] = [];
        for await (const c of req) chunks.push(c as Buffer);
        const next = (JSON.parse(Buffer.concat(chunks).toString('utf8')) as { mode?: string }).mode;
        if (next !== 'ok' && next !== 'error') return send(res, 400, '{}');
        mode = next;
        return send(res, 200, '{}');
      }
      if (method === 'GET' && url.pathname === '/__mock/requests') {
        return send(res, 200, JSON.stringify(log));
      }
      if (method === 'GET' && (url.pathname === '/search' || url.pathname === '/reverse')) {
        const query = Object.fromEntries(url.searchParams);
        log.push({ path: url.pathname, query, userAgent: String(req.headers['user-agent'] ?? ''), at: Date.now() });
        if (mode === 'error') return send(res, 503, '{"error":"unavailable"}');
        if (url.pathname === '/reverse') return send(res, 200, await fixture('reverse-saalfelden.json'));
        const q = (query.q ?? '').trim().toLowerCase().replace(/\s+/g, ' ');
        return send(res, 200, await fixture(SEARCH_FIXTURES[q]));
      }
      send(res, 404, '{}');
    } catch (err) {
      send(res, 500, JSON.stringify({ error: String(err) }));
    }
  });

  // 0.0.0.0, not loopback: the container reaches the host through the docker bridge gateway.
  await new Promise<void>((resolve, reject) => {
    srv.once('error', reject);
    srv.listen(NOMINATIM_MOCK_PORT, '0.0.0.0', resolve);
  });
  server = srv;
}

export async function stopNominatimMock(): Promise<void> {
  const srv = server;
  server = undefined;
  if (!srv) return;
  // Keep-alive connections from the app would otherwise hold close() open.
  srv.closeAllConnections();
  await new Promise<void>((resolve) => srv.close(() => resolve()));
}
