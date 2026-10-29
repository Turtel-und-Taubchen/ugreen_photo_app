import { execFileSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');

function checkIgnore(path: string): number {
	try {
		execFileSync('git', ['check-ignore', '-q', '--no-index', path], { cwd: repoRoot, stdio: 'ignore' });
		return 0;
	} catch (e) {
		return (e as { status: number }).status;
	}
}

describe('gitignore rules', () => {
	it('git ignores build output, test artefacts, local data and secrets', () => {
		for (const p of [
			'node_modules/x',
			'web/node_modules/x',
			'web/dist/index.html',
			'web/test-results/x',
			'web/playwright-report/index.html',
			'data/index.db',
			'data/x.json',
			'foo.db',
			'.env',
			'deploy/.env'
		]) {
			expect(checkIgnore(p), p).toBe(0);
		}
	});

	it('git keeps sources, fixtures and the env example', () => {
		for (const p of [
			'deploy/.env.example',
			'testdata/library/2024/07/sample.heic',
			'web/src/lib/data/x.ts'
		]) {
			expect(checkIgnore(p), p).toBe(1);
		}
	});
});
