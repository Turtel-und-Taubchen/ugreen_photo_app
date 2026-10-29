import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { compose, ROOT_FILE } from './compose';

const LIBRARY = fileURLToPath(new URL('../../../testdata/library', import.meta.url));

async function waitHealthy(): Promise<void> {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    try {
      const res = await fetch('http://127.0.0.1:18470/api/health');
      if (res.status === 200) return;
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  console.error(compose(['logs']));
  throw new Error('app did not become healthy within 60 s');
}

export default async function globalSetup(): Promise<void> {
  const tmp = await fs.mkdtemp(path.join(os.tmpdir(), 'photocurator-e2e-'));
  process.env.PC_E2E_ROOT = tmp;
  await fs.mkdir(path.dirname(ROOT_FILE), { recursive: true });
  await fs.writeFile(ROOT_FILE, tmp);
  compose(['down', '-v']);
  await fs.cp(LIBRARY, path.join(tmp, 'photos'), { recursive: true });
  for (const dir of ['data', 'backup']) {
    await fs.mkdir(path.join(tmp, dir));
    await fs.chmod(path.join(tmp, dir), 0o777);
  }
  compose(['up', '-d', '--build']);
  await waitHealthy();
}
