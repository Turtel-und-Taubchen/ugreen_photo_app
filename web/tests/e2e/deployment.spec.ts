import { execFileSync } from 'node:child_process';
import { expect, test } from '@playwright/test';
import { compose } from './compose';

test('the compose service reports healthy', () => {
  const ids = compose(['ps', '-q', 'photocurator']).split(/\r?\n/).filter((l) => l !== '');
  expect(ids).toHaveLength(1);
  const status = execFileSync('docker', ['inspect', '--format', '{{.State.Health.Status}}', ids[0]], {
    encoding: 'utf8',
  }).trim();
  expect(status).toBe('healthy');
});
