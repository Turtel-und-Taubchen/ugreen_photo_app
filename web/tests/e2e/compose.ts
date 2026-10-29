import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const PROJECT = 'photocurator-e2e';
export const COMPOSE_FILE = fileURLToPath(new URL('./compose.e2e.yml', import.meta.url));
export const ROOT_FILE = fileURLToPath(new URL('../../test-results/.e2e-root', import.meta.url));

export function compose(args: string[]): string {
  return execFileSync('docker', ['compose', '-p', PROJECT, '-f', COMPOSE_FILE, ...args], {
    encoding: 'utf8',
    env: { ...process.env, PC_E2E_ROOT: process.env.PC_E2E_ROOT ?? '' },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
}
