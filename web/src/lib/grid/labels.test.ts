import { describe, expect, it } from 'vitest';
import { dayLabel, monthLabel } from './labels';

describe('labels', () => {
  it('formats months and days in German', () => {
    expect(monthLabel('2023-03')).toBe('März 2023');
    expect(dayLabel('2024-07-02')).toBe('2. Juli 2024');
  });
});
