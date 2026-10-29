import fs from 'node:fs/promises';
import { compose, ROOT_FILE } from './compose';

export default async function globalTeardown(): Promise<void> {
  const root = (await fs.readFile(ROOT_FILE, 'utf8').catch(() => '')).trim();
  process.env.PC_E2E_ROOT = root;
  compose(['down', '-v']);
  if (root) await fs.rm(root, { recursive: true, force: true });
  if (compose(['ps', '-aq']).trim() !== '') throw new Error('e2e containers remain after teardown');
}
