import { execFileSync } from 'node:child_process';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const PROJECT = 'photocurator-e2e';
export const E2E_PORT = 18470;
export const NOMINATIM_MOCK_PORT = 18471;
/** The production compose file; the e2e run only overlays it, so it deploys what users deploy. */
export const BASE_FILE = fileURLToPath(new URL('../../../deploy/docker-compose.yml', import.meta.url));
export const OVERRIDE_FILE = fileURLToPath(new URL('./compose.e2e.yml', import.meta.url));
export const USERS_FILE = fileURLToPath(new URL('../../../testdata/e2e/users.conf', import.meta.url));
export const ROOT_FILE = fileURLToPath(new URL('../../test-results/.e2e-root', import.meta.url));

/**
 * Interpolation values for the base file's `${VAR}`s. Every compose call needs all of
 * them, including `down`/`ps`/`exec`: an empty PHOTOS_PATH turns the volume into `:/photos`,
 * which compose rejects before doing anything. Without a known root a placeholder path is
 * used — no command other than `up` touches the host directories.
 * Explicit values also win over a developer's deploy/.env, which compose reads too.
 */
export function composeEnv(root: string): Record<string, string> {
  const base = root || path.join(os.tmpdir(), 'photocurator-e2e-unknown');
  return {
    PHOTOS_PATH: path.join(base, 'photos'),
    DATA_PATH: path.join(base, 'data'),
    BACKUP_PATH: path.join(base, 'backup'),
    // Without getuid (Windows host) the bind mount presents every file as root-owned and
    // writable for root only, so root is the one uid that can write the fixture photos.
    PUID: String(process.getuid?.() ?? 0),
    PGID: String(process.getgid?.() ?? 0),
    BIND_IP: '127.0.0.1',
    PORT: String(E2E_PORT),
    TZ: 'Europe/Vienna',
    NOMINATIM_CONTACT: 'e2e@example.invalid',
    PC_E2E_USERS_FILE: USERS_FILE,
  };
}

export function compose(args: string[]): string {
  return execFileSync(
    'docker',
    ['compose', '-p', PROJECT, '-f', BASE_FILE, '-f', OVERRIDE_FILE, ...args],
    {
      encoding: 'utf8',
      env: { ...process.env, ...composeEnv(process.env.PC_E2E_ROOT ?? '') },
      stdio: ['ignore', 'pipe', 'pipe'],
    },
  );
}
