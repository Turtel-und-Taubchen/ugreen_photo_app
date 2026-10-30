import { api } from './api';

export type JobStatus = 'queued' | 'running' | 'done' | 'aborted';

export interface Job {
  id: number;
  action: string;
  status: JobStatus;
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

export const createJob = (selection: JobSelection, action: string, payload: unknown): Promise<{ id: number }> =>
  api<{ id: number }>('/api/jobs', { method: 'POST', body: { selection, action, payload } });

export const isActive = (j: Job): boolean => j.status === 'queued' || j.status === 'running';

export function progressText(job: Job, waiting: number): string {
  const text = `Schreibe ${job.done} von ${job.total}`;
  return waiting > 0 ? `${text} · ${waiting} ${waiting === 1 ? 'Job wartet' : 'Jobs warten'}` : text;
}

export function resultText(job: Job): string {
  const text = `${job.ok} erfolgreich, ${job.failed} fehlgeschlagen`;
  return job.skipped > 0 ? `${text}, ${job.skipped} übersprungen` : text;
}
