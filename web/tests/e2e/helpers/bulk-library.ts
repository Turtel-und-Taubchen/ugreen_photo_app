import fs from 'node:fs/promises';
import path from 'node:path';

/** Files in testdata/library. */
export const FIXTURE_FILES = 10;
export const BULK_FILES = 5000;
export const EXPECTED_TOTAL = FIXTURE_FILES + BULK_FILES;

const SOURCE = 'sample.jpg';
const FIRST_YEAR = 2023;
const MONTHS = 24;
const GOLDEN = 0.6180339887498949;

export interface BulkMonth {
  month: string;
  count: number;
  /** Day (YYYY-MM-DD) to number of photos. */
  days: Map<string, number>;
}

export interface BulkPlanEntry {
  rel: string;
  day: string;
}

export interface BulkPlan {
  months: BulkMonth[];
  entries: BulkPlanEntry[];
}

const pad = (n: number, width = 2) => String(n).padStart(width, '0');

/**
 * Deterministic and uneven: month sizes follow a repeating weight pattern, and a quadratic
 * skew toward the end of each month piles many photos onto the same days (more than one
 * grid row each). Months are listed newest first, like the API does.
 */
export function bulkPlan(): BulkPlan {
  const weights = Array.from({ length: MONTHS }, (_, m) => 1 + ((m * 7) % 5));
  const sum = weights.reduce((a, b) => a + b, 0);
  const counts = weights.map((w) => Math.floor((BULK_FILES * w) / sum));
  let rest = BULK_FILES - counts.reduce((a, b) => a + b, 0);
  for (let m = 0; rest > 0; m = (m + 1) % MONTHS, rest--) counts[m]++;

  const months: BulkMonth[] = [];
  const entries: BulkPlanEntry[] = [];
  for (let m = 0; m < MONTHS; m++) {
    const year = FIRST_YEAR + Math.floor(m / 12);
    const mon = (m % 12) + 1;
    const daysInMonth = new Date(Date.UTC(year, mon, 0)).getUTCDate();
    const perDay = new Map<number, number>();
    for (let i = 0; i < counts[m]; i++) {
      const u = (((i + 1) * GOLDEN) % 1) ** 2;
      const d = daysInMonth - Math.min(daysInMonth - 1, Math.floor(u * daysInMonth));
      const n = (perDay.get(d) ?? 0) + 1;
      perDay.set(d, n);
      // Starts above the fixture's WA0001 so no bulk file can overwrite it.
      const seq = 1000 + n;
      const ymd = `${year}${pad(mon)}${pad(d)}`;
      entries.push({
        rel: `${year}/${pad(mon)}/IMG-${ymd}-WA${pad(seq, 4)}.jpg`,
        day: `${year}-${pad(mon)}-${pad(d)}`,
      });
    }
    const days = new Map<string, number>();
    for (const [d, n] of [...perDay].sort((a, b) => b[0] - a[0])) days.set(`${year}-${pad(mon)}-${pad(d)}`, n);
    months.push({ month: `${year}-${pad(mon)}`, count: counts[m], days });
  }
  return { months: months.reverse(), entries };
}

/**
 * Copies the metadata-free sample into 5,000 dated files. The scanner skips files that are
 * younger than a day, so every mtime lies well in the past; fixtures get one older than all
 * bulk days, so a fixture dated by mtime cannot land in a bulk month.
 */
export async function createBulkLibrary(photosDir: string): Promise<BulkPlan> {
  const old = new Date('2022-06-15T12:00:00Z');
  const age = async (dir: string): Promise<void> => {
    for (const entry of await fs.readdir(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) await age(full);
      else await fs.utimes(full, old, old);
    }
  };
  await age(photosDir);

  const plan = bulkPlan();
  const source = path.join(photosDir, '2024', '07', SOURCE);
  for (const dir of new Set(plan.entries.map((e) => path.dirname(e.rel)))) {
    await fs.mkdir(path.join(photosDir, dir), { recursive: true });
  }
  const queue = [...plan.entries];
  const worker = async () => {
    for (let e = queue.pop(); e; e = queue.pop()) {
      const target = path.join(photosDir, e.rel);
      await fs.copyFile(source, target);
      const when = new Date(`${e.day}T12:00:00Z`);
      await fs.utimes(target, when, when);
    }
  };
  await Promise.all(Array.from({ length: 16 }, worker));
  return plan;
}
