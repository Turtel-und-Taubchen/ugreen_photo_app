/**
 * Pure selection model mirroring the server: selection = (groups ∪ ranges ∪ include) \ exclude.
 * All functions return new states and never mutate their input, so a reactive store can wrap them.
 */

export interface PhotoRef {
  id: number;
  takenAt: string | null;
}

export interface SelectionState {
  /** Month ("YYYY-MM") or day ("YYYY-MM-DD") keys. */
  groups: string[];
  /** Inclusive endpoints in either order. */
  ranges: [PhotoRef, PhotoRef][];
  include: PhotoRef[];
  exclude: PhotoRef[];
  /** Last photo that was marked; ranges start here. */
  anchor: PhotoRef | null;
  active: boolean;
}

export interface SelectionRequest {
  filter: { groups: string[]; ranges: [number, number][] };
  include_ids: number[];
  exclude_ids: number[];
}

export function emptySelection(): SelectionState {
  return { groups: [], ranges: [], include: [], exclude: [], anchor: null, active: false };
}

// The server groups on substr(taken_at, 1, 7|10) of the stored text, so slice the text
// instead of parsing it into a Date (which would shift keys across time zones).
const dayKey = (ref: PhotoRef): string | null => (ref.takenAt ? ref.takenAt.slice(0, 10) : null);
const monthKey = (ref: PhotoRef): string | null => (ref.takenAt ? ref.takenAt.slice(0, 7) : null);

/** Grid order: takenAt descending, then id descending, undated photos last. Negative means `a` comes first. */
export function compareGrid(a: PhotoRef, b: PhotoRef): number {
  if (a.takenAt !== b.takenAt) {
    if (a.takenAt === null) return 1;
    if (b.takenAt === null) return -1;
    return a.takenAt > b.takenAt ? -1 : 1;
  }
  return b.id - a.id;
}

function inRange(ref: PhotoRef, [a, b]: [PhotoRef, PhotoRef]): boolean {
  const [first, last] = compareGrid(a, b) <= 0 ? [a, b] : [b, a];
  return compareGrid(first, ref) <= 0 && compareGrid(ref, last) <= 0;
}

const hasRef = (list: PhotoRef[], ref: PhotoRef): boolean => list.some((r) => r.id === ref.id);
const without = (list: PhotoRef[], ref: PhotoRef): PhotoRef[] => list.filter((r) => r.id !== ref.id);

export function isSelected(state: SelectionState, ref: PhotoRef): boolean {
  if (hasRef(state.exclude, ref)) return false;
  if (hasRef(state.include, ref)) return true;
  const day = dayKey(ref);
  const month = monthKey(ref);
  if ((day && state.groups.includes(day)) || (month && state.groups.includes(month))) return true;
  return state.ranges.some((r) => inRange(ref, r));
}

export function isEmpty(state: SelectionState): boolean {
  return state.groups.length === 0 && state.ranges.length === 0 && state.include.length === 0;
}

/** Enters selection mode without marking anything. */
export function start(state: SelectionState): SelectionState {
  return state.active ? state : { ...state, active: true };
}

export function toggle(state: SelectionState, ref: PhotoRef): SelectionState {
  if (isSelected(state, ref)) {
    // Dropping the explicit inclusion may leave the photo covered by a group or range.
    let next = { ...state, include: without(state.include, ref), active: true };
    if (isSelected(next, ref)) next = { ...next, exclude: [...next.exclude, ref] };
    return next;
  }
  let next = { ...state, exclude: without(state.exclude, ref), active: true, anchor: ref };
  if (!isSelected(next, ref)) next = { ...next, include: [...next.include, ref] };
  return next;
}

export function selectRange(state: SelectionState, ref: PhotoRef): SelectionState {
  const anchor = state.anchor;
  if (!anchor) {
    return isSelected(state, ref) ? { ...state, active: true, anchor: ref } : toggle(state, ref);
  }
  const range: [PhotoRef, PhotoRef] = [anchor, ref];
  return {
    ...state,
    ranges: [...state.ranges, range],
    // Both are redundant inside the range; stale exclusions would otherwise punch holes in it.
    include: state.include.filter((r) => !inRange(r, range)),
    exclude: state.exclude.filter((r) => !inRange(r, range)),
    anchor: ref,
    active: true,
  };
}

const isMonthKey = (key: string): boolean => key.length === 7;
const inGroup = (key: string, ref: PhotoRef): boolean => !!ref.takenAt && ref.takenAt.startsWith(key);
const parentMonth = (key: string): string => key.slice(0, 7);

export function selectGroup(state: SelectionState, key: string): SelectionState {
  if (!isMonthKey(key) && state.groups.includes(parentMonth(key))) return state;
  const groups = state.groups.filter((g) => !(isMonthKey(key) && g !== key && g.startsWith(key)));
  if (!groups.includes(key)) groups.push(key);
  return {
    ...state,
    groups,
    include: state.include.filter((r) => !inGroup(key, r)),
    exclude: state.exclude.filter((r) => !inGroup(key, r)),
    active: true,
  };
}

export function isGroupFullySelected(state: SelectionState, key: string): boolean {
  const covered = state.groups.includes(key) || (!isMonthKey(key) && state.groups.includes(parentMonth(key)));
  return covered && !state.exclude.some((r) => inGroup(key, r));
}

export function clear(): SelectionState {
  return emptySelection();
}

export function toRequest(state: SelectionState): SelectionRequest {
  return {
    filter: {
      groups: [...state.groups],
      ranges: state.ranges.map(([a, b]) => [a.id, b.id] as [number, number]),
    },
    include_ids: state.include.map((r) => r.id),
    exclude_ids: state.exclude.map((r) => r.id),
  };
}
