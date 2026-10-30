import { describe, expect, it } from 'vitest';
import {
  clear,
  emptySelection,
  isEmpty,
  isGroupFullySelected,
  isSelected,
  selectGroup,
  selectRange,
  toRequest,
  toggle,
  type PhotoRef,
} from './selection';

const photo = (id: number, takenAt: string | null): PhotoRef => ({ id, takenAt });
const a = photo(5, '2024-07-12T10:00:00Z');
const b = photo(4, '2024-07-12T09:00:00Z');
const c = photo(3, '2024-07-11T09:00:00Z');
const d = photo(2, '2024-06-30T09:00:00Z');
const e = photo(1, '2024-06-01T09:00:00Z');

describe('selection', () => {
  it('toggling a photo twice leaves nothing selected', () => {
    const s = toggle(toggle(emptySelection(), a), a);
    expect(isSelected(s, a)).toBe(false);
    expect(isEmpty(s)).toBe(true);
    expect(toRequest(s).exclude_ids).toEqual([]);
  });

  it('deselecting a photo inside a selected month becomes a single exclusion', () => {
    const s = toggle(selectGroup(emptySelection(), '2024-07'), b);
    expect(isSelected(s, b)).toBe(false);
    expect(isSelected(s, a)).toBe(true);
    expect(toRequest(s)).toEqual({
      filter: { groups: ['2024-07'], ranges: [] },
      include_ids: [],
      exclude_ids: [4],
    });
    expect(isGroupFullySelected(s, '2024-07')).toBe(false);
  });

  it('a whole month is sent as one group, not as ids', () => {
    const s = selectGroup(emptySelection(), '2024-07');
    expect(toRequest(s)).toEqual({ filter: { groups: ['2024-07'], ranges: [] }, include_ids: [], exclude_ids: [] });
    expect(isSelected(s, c)).toBe(true);
    expect(isSelected(s, d)).toBe(false);
    expect(selectGroup(s, '2024-07-12')).toBe(s);
  });

  it('selecting a group drops the exceptions inside it', () => {
    let s = toggle(toggle(emptySelection(), d), e);
    s = toggle(selectGroup(s, '2024-07'), a);
    s = selectGroup(s, '2024-07');
    expect(toRequest(s).exclude_ids).toEqual([]);
    s = selectGroup(s, '2024-06');
    expect(toRequest(s).include_ids).toEqual([]);
    expect(isGroupFullySelected(s, '2024-06')).toBe(true);
  });

  it('a range runs from the last marked photo, not from the last deselected', () => {
    let s = toggle(toggle(emptySelection(), e), c);
    s = toggle(s, e); // deselecting e must not move the anchor away from c
    s = selectRange(s, a);
    expect(toRequest(s).filter.ranges).toEqual([[3, 5]]);
    expect(isSelected(s, b)).toBe(true);
    expect(isSelected(s, d)).toBe(false);
  });

  it('clear empties the selection and leaves selection mode', () => {
    const s = clear();
    expect(s).toEqual(emptySelection());
    expect(s.active).toBe(false);
    expect(toggle(s, a).active).toBe(true);
  });
});
