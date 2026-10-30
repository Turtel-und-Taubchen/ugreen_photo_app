import { fetchPhotoPage, type PhotoItem, type PhotoPage } from '../photos';

/** Months whose pages are being fetched at the same time. */
export const MAX_PARALLEL_MONTHS = 2;
const DEFAULT_PAGE_SIZE = 200;

export interface PhotoLoaderOptions {
  fetchFn?: typeof fetch;
  pageSize?: number;
  /** Called whenever a month gained photos or a request for it failed. */
  onChange?: (month: string) => void;
}

export interface PhotoLoader {
  /** Resolves once `uptoIndex` is loaded, the month is exhausted, or a request failed. Never rejects. */
  ensure(month: string, uptoIndex: number): Promise<void>;
  /** The photo at `index` within the month, undefined while it is still a placeholder. */
  get(month: string, index: number): PhotoItem | undefined;
  /** Number of photos loaded so far; they always form the prefix 0..loaded-1. */
  loaded(month: string): number;
  /** True while the last request for the month failed and no later one succeeded. */
  failed(month: string): boolean;
  /** Drops every cached page; responses still in flight are discarded. */
  reset(): void;
}

interface MonthState {
  items: PhotoItem[];
  ids: Set<number>;
  cursor: string | null;
  done: boolean;
  failed: boolean;
  wanted: number;
  task: Promise<void> | null;
}

interface Generation {
  months: Map<string, MonthState>;
  queue: (() => void)[];
  active: number;
}

const newGeneration = (): Generation => ({ months: new Map(), queue: [], active: 0 });

export function createPhotoLoader(opts: PhotoLoaderOptions = {}): PhotoLoader {
  const pageSize = opts.pageSize ?? DEFAULT_PAGE_SIZE;
  let gen = newGeneration();

  const stateOf = (month: string): MonthState => {
    let s = gen.months.get(month);
    if (!s) {
      s = { items: [], ids: new Set(), cursor: null, done: false, failed: false, wanted: -1, task: null };
      gen.months.set(month, s);
    }
    return s;
  };

  // Newest request first: while scrolling fast, the month the user just reached matters
  // more than the ones already scrolled past.
  const pump = (g: Generation) => {
    while (g.active < MAX_PARALLEL_MONTHS && g.queue.length > 0) g.queue.pop()!();
  };

  // Pages of one month are fetched strictly one after another: each cursor comes from the
  // previous page, and `wanted` is re-read per page so later `ensure` calls extend the run.
  async function drain(g: Generation, month: string, s: MonthState): Promise<void> {
    while (g === gen && !s.done && s.items.length <= s.wanted) {
      let page: PhotoPage;
      try {
        page = await fetchPhotoPage(month, s.cursor, pageSize, { fetchFn: opts.fetchFn });
      } catch {
        if (g !== gen) return;
        s.failed = true;
        opts.onChange?.(month);
        return;
      }
      if (g !== gen) return;
      const before = s.items.length;
      for (const item of page.items) {
        if (s.ids.has(item.id)) continue;
        s.ids.add(item.id);
        s.items.push(item);
      }
      s.failed = false;
      // A page without a single new photo would otherwise be requested again forever.
      s.done = page.next === null || s.items.length === before;
      s.cursor = page.next;
      opts.onChange?.(month);
    }
  }

  function schedule(g: Generation, month: string, s: MonthState): Promise<void> {
    return new Promise<void>((resolve) => {
      g.queue.push(async () => {
        g.active++;
        try {
          await drain(g, month, s);
        } finally {
          g.active--;
          s.task = null;
          resolve();
          pump(g);
        }
      });
      pump(g);
    });
  }

  return {
    ensure(month, uptoIndex) {
      const s = stateOf(month);
      s.wanted = Math.max(s.wanted, uptoIndex);
      if (s.task) return s.task;
      if (s.done || s.items.length > uptoIndex) return Promise.resolve();
      s.task = schedule(gen, month, s);
      return s.task;
    },
    get: (month, index) => gen.months.get(month)?.items[index],
    loaded: (month) => gen.months.get(month)?.items.length ?? 0,
    failed: (month) => gen.months.get(month)?.failed ?? false,
    reset() {
      gen = newGeneration();
    },
  };
}
