import { api, type ApiOptions } from './api';

export interface DayCount {
  day: string;
  count: number;
}

export interface MonthGroup {
  month: string;
  count: number;
  days: DayCount[];
}

export interface FolderCount {
  folder: string;
  count: number;
}

/** `total` also counts undated files, which appear in no month. Months and days are newest first. */
export interface PhotoGroups {
  total: number;
  months: MonthGroup[];
  folders: FolderCount[];
}

export interface PhotoItem {
  id: number;
  relPath: string;
  mediaType: string;
  takenAt: string | null;
  thumbKey: string;
}

export interface PhotoPage {
  items: PhotoItem[];
  next: string | null;
}

export interface PhotoLocation {
  label: string;
  sublocation: string;
  city: string;
  state: string;
  country: string;
}

/** `prev_id` is the next newer photo, `next_id` the next older one; null at either end. */
export interface PhotoDetail {
  id: number;
  rel_path: string;
  folder: string;
  media_type: string;
  format: string;
  taken_at: string | null;
  taken_source: string;
  thumb_key: string;
  tags: string[];
  location: PhotoLocation;
  review_state: string;
  read_error: string | null;
  prev_id: number | null;
  next_id: number | null;
}

type CallOptions = Pick<ApiOptions, 'fetchFn' | 'redirect'>;

export const fetchGroups = (opts: CallOptions = {}): Promise<PhotoGroups> =>
  api<PhotoGroups>('/api/photos/groups', opts);

/** One keyset page of a month, newest first; pass the previous page's `next` as `after`. */
export function fetchPhotoPage(
  month: string,
  after: string | null,
  limit?: number,
  opts: CallOptions = {},
): Promise<PhotoPage> {
  const q = new URLSearchParams({ month });
  if (after !== null) q.set('after', after);
  if (limit !== undefined) q.set('limit', String(limit));
  return api<PhotoPage>(`/api/photos?${q}`, opts);
}

export const fetchPhoto = (id: number, opts: CallOptions = {}): Promise<PhotoDetail> =>
  api<PhotoDetail>(`/api/photos/${id}`, opts);
