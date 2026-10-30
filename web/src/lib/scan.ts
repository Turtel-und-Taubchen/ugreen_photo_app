import { api } from './api';

export interface UnreadableFile {
  rel_path: string;
  error: string;
}

export interface ScanStatus {
  state: string;
  trigger: string;
  done: number;
  total: number;
  started_at: string | null;
  finished_at: string | null;
  last_error: string;
  files: number;
  unreadable: UnreadableFile[];
  unreadable_count: number;
}

export const fetchScan = (): Promise<ScanStatus> => api<ScanStatus>('/api/scan');

export const startScan = (): Promise<ScanStatus> => api<ScanStatus>('/api/scan', { method: 'POST' });

export const isRunning = (s: ScanStatus): boolean => s.state === 'running';

// The server sends UTC; people read their own clock.
function clock(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

export function scanStatusText(s: ScanStatus): string {
  if (isRunning(s)) return `Einlesen läuft … ${s.done} von ${s.total}`;
  if (!s.finished_at) return 'Noch nicht eingelesen';
  return `${s.files} Dateien · zuletzt eingelesen ${clock(s.finished_at)}`;
}

export function scanErrorText(s: ScanStatus): string {
  return s.last_error ? `Letzter Lauf mit Fehler: ${s.last_error}` : '';
}
