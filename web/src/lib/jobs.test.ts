import { describe, expect, it } from 'vitest';
import { progressText, resultText, type Job } from './jobs';

const base: Job = {
  id: 1, action: 'tags.add', status: 'done', total: 3, done: 3, ok: 2, skipped: 0, failed: 1,
  created_at: '2024-07-05T09:00:00Z', finished_at: '2024-07-05T09:00:05Z',
};

describe('job texts', () => {
  it("a running job reads 'Schreibe x von y'", () => {
    const running = { ...base, status: 'running' as const, done: 1, finished_at: null };
    expect(progressText(running, 0)).toBe('Schreibe 1 von 3');
    expect(progressText(running, 2)).toBe('Schreibe 1 von 3 · 2 Jobs warten');
    expect(progressText(running, 1)).toBe('Schreibe 1 von 3 · 1 Job wartet');
  });

  it("a finished job reads 'n erfolgreich, m fehlgeschlagen'", () => {
    expect(resultText(base)).toBe('2 erfolgreich, 1 fehlgeschlagen');
    expect(resultText({ ...base, skipped: 4 })).toBe('2 erfolgreich, 1 fehlgeschlagen, 4 übersprungen');
  });
});
