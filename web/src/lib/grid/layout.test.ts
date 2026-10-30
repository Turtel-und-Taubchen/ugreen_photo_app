import { describe, expect, it } from 'vitest';
import type { MonthGroup } from '../photos';
import {
  DAY_HEADER_HEIGHT,
  MAX_TILES,
  MONTH_HEADER_HEIGHT,
  buildRows,
  columnsFor,
  monthForFolder,
  offsetOfMonth,
  tileSizeFor,
  windowAt,
  type PhotoRow,
  type Row,
} from './layout';

function month(name: string, ...dayCounts: number[]): MonthGroup {
  const days = dayCounts.map((count, i) => ({ day: `${name}-${String(28 - i).padStart(2, '0')}`, count }));
  return { month: name, count: dayCounts.reduce((a, b) => a + b, 0), days };
}

const photoRows = (rows: Row[]): PhotoRow[] => rows.filter((r): r is PhotoRow => r.kind === 'photos');
const ranges = (rows: Row[]) => photoRows(rows).map((r) => [r.day, r.start, r.end]);

function grid(width: number, height: number, months: MonthGroup[]) {
  const columns = columnsFor(width, height);
  return { columns, ...buildRows({ months }, columns, tileSizeFor(width, columns)) };
}

// Tiles in every row at least partly inside [scrollTop, scrollTop + height).
const tilesOnScreen = (rows: Row[], scrollTop: number, height: number) =>
  photoRows(rows)
    .filter((r) => r.top < scrollTop + height && r.top + r.height > scrollTop)
    .reduce((sum, r) => sum + r.end - r.start, 0);

describe('columnsFor', () => {
  it('a phone gets at least 3 columns and a desktop at least 6', () => {
    expect(columnsFor(390, 844)).toBeGreaterThanOrEqual(3);
    expect(columnsFor(1440, 900)).toBeGreaterThanOrEqual(6);
    expect(columnsFor(3840, 2160)).toBeGreaterThanOrEqual(6);
    expect(columnsFor(200, 5000)).toBe(3);
  });

  it('a screenful never shows more than 200 tiles, at any scroll position', () => {
    for (const [width, height] of [
      [390, 844],
      [1440, 900],
      [2560, 1440],
      [3840, 2160],
      [7680, 4320],
    ]) {
      const { rows, height: total } = grid(width, height, [month('2024-07', 5000)]);
      for (let top = 0; top + height <= total; top += 37) {
        expect(tilesOnScreen(rows, top, height)).toBeLessThanOrEqual(200);
      }
    }
  });
});

describe('buildRows', () => {
  it('every day starts a new row and its last partial row ends at the day boundary', () => {
    const { rows, height } = buildRows({ months: [month('2024-07', 5, 3, 4)] }, 4, 100);
    expect(rows.map((r) => r.kind)).toEqual(['month', 'day', 'photos', 'photos', 'day', 'photos', 'day', 'photos']);
    expect(ranges(rows)).toEqual([
      ['2024-07-28', 0, 4],
      ['2024-07-28', 4, 5],
      ['2024-07-27', 5, 8],
      ['2024-07-26', 8, 12],
    ]);
    expect(height).toBe(MONTH_HEADER_HEIGHT + 3 * DAY_HEADER_HEIGHT + 4 * 100);
  });

  it('a day filling its rows exactly gets no empty trailing row', () => {
    const { rows } = buildRows({ months: [month('2024-07', 8, 1)] }, 4, 100);
    expect(ranges(rows)).toEqual([
      ['2024-07-28', 0, 4],
      ['2024-07-28', 4, 8],
      ['2024-07-27', 8, 9],
    ]);
  });

  it('index ranges restart in each month and stack rows without gaps', () => {
    const { rows, height } = buildRows({ months: [month('2024-07', 3), month('2024-06', 2, 0, 7)] }, 3, 50);
    expect(photoRows(rows).map((r) => [r.month, r.start, r.end])).toEqual([
      ['2024-07', 0, 3],
      ['2024-06', 0, 2],
      ['2024-06', 2, 5],
      ['2024-06', 5, 8],
      ['2024-06', 8, 9],
    ]);
    rows.forEach((r, i) => expect(r.top).toBe(i === 0 ? 0 : rows[i - 1].top + rows[i - 1].height));
    expect(height).toBe(rows.at(-1)!.top + rows.at(-1)!.height);
  });

  it('every photo of a month lies in exactly one row, in order', () => {
    const months = [month('2024-07', 1, 17, 6, 12), month('2024-05', 25)];
    for (const columns of [3, 5, 6, 12]) {
      const { rows } = buildRows({ months }, columns, 100);
      for (const m of months) {
        const own = photoRows(rows).filter((r) => r.month === m.month);
        expect(own[0].start).toBe(0);
        own.forEach((r, i) => {
          expect(r.end - r.start).toBeGreaterThan(0);
          expect(r.end - r.start).toBeLessThanOrEqual(columns);
          if (i > 0) expect(r.start).toBe(own[i - 1].end);
        });
        expect(own.at(-1)!.end).toBe(m.count);
      }
    }
  });
});

describe('windowAt', () => {
  it('covers the viewport plus half a screen per side, row boundaries exclusive', () => {
    // 10 photo rows of 100 px follow the headers at 0..84.
    const { rows } = buildRows({ months: [month('2024-07', 40)] }, 4, 100);
    const scrollTop = MONTH_HEADER_HEIGHT + DAY_HEADER_HEIGHT + 400;
    const w = windowAt(rows, scrollTop, 200);
    // Overscan spans 100 px per side; the rows ending or starting exactly on its edges are out.
    expect(photoRows(w.rows).map((r) => r.start)).toEqual([12, 16, 20, 24]);
    expect(w.tiles).toBe(16);
    expect(w.rows).toEqual(rows.slice(w.start, w.end));
  });

  it('includes the headers at the very top and stops at the end of the grid', () => {
    const { rows, height } = buildRows({ months: [month('2024-07', 6)] }, 3, 100);
    expect(windowAt(rows, 0, 1000).rows).toEqual(rows);
    expect(windowAt(rows, height + 5000, 1000).rows).toEqual([]);
  });

  it('no window ever holds more than 300 tiles, on any screen size', () => {
    const months = [month('2024-07', 4000, 1, 2, 3, 900), month('2024-06', 7, 7, 7, 2000)];
    for (const [width, height] of [
      [390, 844],
      [1440, 900],
      [3840, 2160],
      [7680, 4320],
      [390, 20000],
      [200, 100000],
    ]) {
      const { rows, height: total } = grid(width, height, months);
      for (let top = -height; top < total + height; top += 211) {
        const w = windowAt(rows, top, height);
        const tiles = photoRows(w.rows).reduce((sum, r) => sum + r.end - r.start, 0);
        expect(tiles).toBe(w.tiles);
        expect(tiles).toBeLessThanOrEqual(MAX_TILES);
        const onScreen = tilesOnScreen(rows, top, height);
        if (onScreen > 0) expect(tiles).toBeGreaterThan(0);
        // Wherever the screen fits the budget, everything on screen is in the window.
        if (onScreen <= MAX_TILES) {
          const visible = rows.filter((r) => r.top < top + height && r.top + r.height > top);
          for (const r of visible) expect(w.rows).toContain(r);
        }
      }
    }
  });
});

describe('jump targets', () => {
  const months = [month('2024-07', 3), month('2023-05', 1), month('2023-01', 2)];

  it('a folder jumps to its month, or the next older month when that month is empty', () => {
    expect(monthForFolder({ months }, '2023/05')).toBe('2023-05');
    expect(monthForFolder({ months }, '2023/03')).toBe('2023-01');
    expect(monthForFolder({ months }, '2025/01')).toBe('2024-07');
    expect(monthForFolder({ months }, '2022/12')).toBeNull();
    expect(monthForFolder({ months }, 'Urlaub')).toBeNull();
  });

  it('a month offset is the top of its header row', () => {
    const { rows } = buildRows({ months }, 3, 100);
    expect(offsetOfMonth(rows, '2024-07')).toBe(0);
    expect(offsetOfMonth(rows, '2023-05')).toBe(MONTH_HEADER_HEIGHT + DAY_HEADER_HEIGHT + 100);
    expect(offsetOfMonth(rows, '2023-03')).toBeNull();
  });
});
