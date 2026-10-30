import { describe, expect, it } from 'vitest';
import { actionLabel, canUndo, progressText, resultSummary, resultText, type Job } from './jobs';

const base: Job = {
  id: 1, action: 'tags.add', status: 'done', user: 'anna', undo_of: null, undone_by: null, total: 3, done: 3, ok: 2, skipped: 0, failed: 1,
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

describe('job list helpers', () => {
  const undo: Job = { ...base, id: 2, action: 'undo', undo_of: 1 };

  it('actions read as German labels, unknown ones stay raw', () => {
    expect(actionLabel(base, [base])).toBe('Tags hinzufügen');
    expect(actionLabel({ ...base, action: 'foo.bar' }, [])).toBe('foo.bar');
  });

  it('an undo job names the action of its source job, with a fallback', () => {
    expect(actionLabel(undo, [undo, base])).toBe('Rückgängig: Tags hinzufügen');
    expect(actionLabel(undo, [undo])).toBe('Rückgängig');
  });

  it("the result reads 'n erfolgreich · m übersprungen · k fehlgeschlagen'", () => {
    expect(resultSummary({ ...base, ok: 2, skipped: 1, failed: 0 })).toBe('2 erfolgreich · 1 übersprungen · 0 fehlgeschlagen');
  });

  it('only finished jobs that are not undone can be undone', () => {
    expect(canUndo(base)).toBe(true);
    expect(canUndo({ ...base, status: 'running' })).toBe(false);
    expect(canUndo({ ...base, status: 'queued' })).toBe(false);
    expect(canUndo({ ...base, undone_by: 5 })).toBe(false);
    expect(canUndo({ ...undo, status: 'running' })).toBe(false);
  });
});
