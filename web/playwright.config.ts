import { defineConfig, devices } from '@playwright/test';
import { fileURLToPath } from 'node:url';

const here = (p: string) => fileURLToPath(new URL(p, import.meta.url));

export default defineConfig({
  testDir: 'tests/e2e',
  workers: 1,
  outputDir: 'test-results/artifacts',
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  globalSetup: here('tests/e2e/global-setup.ts'),
  globalTeardown: here('tests/e2e/global-teardown.ts'),
  use: { baseURL: 'http://127.0.0.1:18470' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
