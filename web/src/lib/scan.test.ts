import { describe, expect, it } from 'vitest';
import { scanStatusText, type ScanStatus } from './scan';

const base: ScanStatus = {
  state: 'idle', trigger: 'start', done: 0, total: 0, started_at: null, finished_at: null,
  last_error: '', files: 0, unreadable: [], unreadable_count: 0,
};

describe('scanStatusText', () => {
  it('a running scan reads as progress x of y, a finished one as file count and time', () => {
    expect(scanStatusText({ ...base, state: 'running', done: 3, total: 9 })).toBe('Einlesen läuft … 3 von 9');
    const finished = new Date(2024, 6, 5, 9, 7).toISOString(); // local wall clock 09:07
    expect(scanStatusText({ ...base, files: 9, finished_at: finished })).toBe('9 Dateien · zuletzt eingelesen 09:07');
    expect(scanStatusText(base)).toBe('Noch nicht eingelesen');
  });
});
