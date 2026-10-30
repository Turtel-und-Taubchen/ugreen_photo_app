import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { compose, ROOT_FILE } from './compose';
import { createBulkLibrary } from './helpers/bulk-library';
import { startNominatimMock } from './nominatim-mock';

const LIBRARY = fileURLToPath(new URL('../../../testdata/library', import.meta.url));

export default async function globalSetup(): Promise<void> {
  const tmp = await fs.mkdtemp(path.join(os.tmpdir(), 'photocurator-e2e-'));
  process.env.PC_E2E_ROOT = tmp;
  await fs.mkdir(path.dirname(ROOT_FILE), { recursive: true });
  await fs.writeFile(ROOT_FILE, tmp);
  compose(['down', '-v']);
  await startNominatimMock();
  await fs.cp(LIBRARY, path.join(tmp, 'photos'), { recursive: true });
  await createBulkLibrary(path.join(tmp, 'photos'));
  try {
    // The app refuses to start unless all three mounts are writable for PUID:PGID: copy the
    // fixtures into named volumes and chown them (a host bind mount cannot do this on Windows).
    compose(['--profile', 'seed', 'run', '--rm', '--no-deps', 'seed']);
    // --wait returns only once the image's own healthcheck reports healthy.
    compose(['up', '-d', '--pull', 'never', '--wait', '--wait-timeout', '90']);
  } catch (err) {
    try {
      console.error(compose(['logs']));
    } catch {
      // logs are a diagnostic only
    }
    throw err;
  }
}
