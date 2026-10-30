import type { PhotoGroups } from '../photos';

export const MONTH_HEADER_HEIGHT = 48;
export const DAY_HEADER_HEIGHT = 44;
/** Upper bound of tiles in one window, overscan included, whatever the screen size. */
export const MAX_TILES = 300;
const MIN_COLUMNS = 3;
const MIN_TILE_WIDTH = 120;
const MAX_TILES_PER_SCREEN = 200;

interface RowBase {
  top: number;
  height: number;
  month: string;
}

export interface MonthRow extends RowBase {
  kind: 'month';
  count: number;
}

export interface DayRow extends RowBase {
  kind: 'day';
  day: string;
  count: number;
}

/** Photos `start` (inclusive) to `end` (exclusive), indexed within `month` newest first. */
export interface PhotoRow extends RowBase {
  kind: 'photos';
  day: string;
  start: number;
  end: number;
}

export type Row = MonthRow | DayRow | PhotoRow;

export interface GridLayout {
  rows: Row[];
  height: number;
}

export interface GridWindow {
  /** Index of the first row in the window; `end` is exclusive. */
  start: number;
  end: number;
  rows: Row[];
  tiles: number;
}

/** Square tiles fill the width; `columnsFor` relies on exactly this pitch. */
export const tileSizeFor = (width: number, columns: number): number => width / columns;

const tilesOf = (row: Row): number => (row.kind === 'photos' ? row.end - row.start : 0);

// A viewport not aligned to the row grid cuts into one extra row, hence the +1.
const tilesPerScreen = (columns: number, width: number, viewportHeight: number): number =>
  columns * (Math.ceil(viewportHeight / tileSizeFor(width, columns)) + 1);

/**
 * One tile per 120 px, at least 3; on large screens the count is lowered until a screenful
 * holds at most 200 tiles, so the tiles grow instead of multiplying.
 */
export function columnsFor(width: number, viewportHeight: number): number {
  let columns = Math.max(MIN_COLUMNS, Math.floor(width / MIN_TILE_WIDTH));
  while (columns > MIN_COLUMNS && tilesPerScreen(columns, width, viewportHeight) > MAX_TILES_PER_SCREEN) {
    columns--;
  }
  return columns;
}

/**
 * Month header, then per day a day header and its photo rows. Every day starts a new row,
 * so a row's index range never spans two days.
 */
export function buildRows(groups: Pick<PhotoGroups, 'months'>, columns: number, tileSize: number): GridLayout {
  const rows: Row[] = [];
  let top = 0;
  const push = (row: Row) => {
    rows.push(row);
    top += row.height;
  };
  for (const m of groups.months) {
    const days = m.days.filter((d) => d.count > 0);
    if (days.length === 0) continue;
    const count = days.reduce((sum, d) => sum + d.count, 0);
    push({ kind: 'month', top, height: MONTH_HEADER_HEIGHT, month: m.month, count });
    let index = 0;
    for (const d of days) {
      push({ kind: 'day', top, height: DAY_HEADER_HEIGHT, month: m.month, day: d.day, count: d.count });
      const dayEnd = index + d.count;
      for (let start = index; start < dayEnd; start += columns) {
        const end = Math.min(start + columns, dayEnd);
        push({ kind: 'photos', top, height: tileSize, month: m.month, day: d.day, start, end });
      }
      index = dayEnd;
    }
  }
  return { rows, height: top };
}

/** First row whose bottom lies below `y`. */
function rowAt(rows: Row[], y: number): number {
  let lo = 0;
  let hi = rows.length;
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (rows[mid].top + rows[mid].height > y) hi = mid;
    else lo = mid + 1;
  }
  return lo;
}

/** First row starting at or below `y`. */
function rowFrom(rows: Row[], y: number): number {
  let lo = 0;
  let hi = rows.length;
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (rows[mid].top >= y) hi = mid;
    else lo = mid + 1;
  }
  return lo;
}

/**
 * Rows intersecting the viewport plus up to half a screen of overscan per side. Visible rows
 * are taken first (top down), overscan is then added alternately below and above; whatever
 * would push the window past `MAX_TILES` is left out, so the window stays contiguous.
 */
export function windowAt(rows: Row[], scrollTop: number, viewportHeight: number): GridWindow {
  const overscan = viewportHeight / 2;
  const visibleEnd = rowFrom(rows, scrollTop + viewportHeight);
  const overStart = rowAt(rows, scrollTop - overscan);
  const overEnd = rowFrom(rows, scrollTop + viewportHeight + overscan);
  const start0 = Math.min(rowAt(rows, scrollTop), visibleEnd);
  let start = start0;
  let end = start0;
  let tiles = 0;
  const fits = (row: Row) => tiles + tilesOf(row) <= MAX_TILES;

  while (end < visibleEnd && fits(rows[end])) tiles += tilesOf(rows[end++]);
  if (end === visibleEnd) {
    for (let grew = true; grew; ) {
      grew = false;
      if (end < overEnd && fits(rows[end])) {
        tiles += tilesOf(rows[end++]);
        grew = true;
      }
      if (start > overStart && fits(rows[start - 1])) {
        tiles += tilesOf(rows[--start]);
        grew = true;
      }
    }
  }
  return { start, end, rows: rows.slice(start, end), tiles };
}

/** Top of the month's header row, or null when the month has no row. */
export function offsetOfMonth(rows: Row[], month: string): number | null {
  const row = rows.find((r) => r.kind === 'month' && r.month === month);
  return row ? row.top : null;
}

/**
 * Jump target for a `YYYY/MM` folder: that month when it has photos, else the next older
 * month that has some, else null.
 */
export function monthForFolder(groups: Pick<PhotoGroups, 'months'>, folder: string): string | null {
  const match = /^(\d{4})\/(\d{2})$/.exec(folder);
  if (!match) return null;
  const target = `${match[1]}-${match[2]}`;
  let best: string | null = null;
  for (const m of groups.months) {
    if (m.count > 0 && m.month <= target && (best === null || m.month > best)) best = m.month;
  }
  return best;
}
