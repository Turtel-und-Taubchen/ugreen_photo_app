import { api } from './api';

export type JobStatus = 'queued' | 'running' | 'done' | 'aborted';

export interface Job {
  id: number;
  action: string;
  status: JobStatus;
  user: string;
  undo_of: number | null;
  undone_by: number | null;
  total: number;
  done: number;
  ok: number;
  skipped: number;
  failed: number;
  created_at: string;
  finished_at: string | null;
}

export interface JobItem {
  rel_path: string;
  state: string;
  reason: string;
}

export interface JobDetail extends Job {
  items: JobItem[];
}

export interface JobSelection {
  filter: { groups: string[]; ranges: [number, number][] };
  include_ids: number[];
  exclude_ids: number[];
}

export const fetchJobs = (): Promise<Job[]> => api<Job[]>('/api/jobs');

export const fetchJob = (id: number): Promise<JobDetail> => api<JobDetail>(`/api/jobs/${id}`);

/** Fired after a job was queued so the progress display polls now instead of at its idle interval. */
export const JOB_CREATED_EVENT = 'photocurator:job-created';

export async function createJob(selection: JobSelection, action: string, payload: unknown): Promise<{ id: number }> {
  const created = await api<{ id: number }>('/api/jobs', { method: 'POST', body: { selection, action, payload } });
  if (typeof window !== 'undefined') window.dispatchEvent(new Event(JOB_CREATED_EVENT));
  return created;
}

export const isActive = (j: Job): boolean => j.status === 'queued' || j.status === 'running';

export function progressText(job: Job, waiting: number): string {
  const text = `Schreibe ${job.done} von ${job.total}`;
  return waiting > 0 ? `${text} · ${waiting} ${waiting === 1 ? 'Job wartet' : 'Jobs warten'}` : text;
}

export const undoJob = (id: number): Promise<{ id: number }> => api<{ id: number }>(`/api/jobs/${id}/undo`, { method: 'POST' });

const ACTION_LABELS: Record<string, string> = {
  'tags.add': 'Tags hinzufügen',
  'tags.remove': 'Tags entfernen',
  'location.set': 'Ort setzen',
  'location.remove': 'Ort entfernen',
};

export interface LocationPayload {
  location: {
    lat: number;
    lon: number;
    label: string;
    sublocation: string;
    city: string;
    state: string;
    country: string;
    countryCode: string;
  };
  /** "missing" leaves photos that already have a place untouched. */
  mode: 'all' | 'missing';
}

export function actionLabel(job: Job, all: Job[]): string {
  if (job.action !== 'undo') return ACTION_LABELS[job.action] ?? job.action;
  const source = all.find((j) => j.id === job.undo_of);
  return source ? `Rückgängig: ${actionLabel(source, all)}` : 'Rückgängig';
}

export function resultSummary(job: Job): string {
  return `${job.ok} erfolgreich · ${job.skipped} übersprungen · ${job.failed} fehlgeschlagen`;
}

// Active jobs are still changing files; an already undone one has nothing left to revert.
export const canUndo = (job: Job): boolean => !isActive(job) && job.undone_by === null;

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' });
}

export function resultText(job: Job): string {
  const text = `${job.ok} erfolgreich, ${job.failed} fehlgeschlagen`;
  return job.skipped > 0 ? `${text}, ${job.skipped} übersprungen` : text;
}
