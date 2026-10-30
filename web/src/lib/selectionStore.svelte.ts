import { api } from './api';
import * as model from './selection';
import type { PhotoRef, SelectionState } from './selection';

const DEBOUNCE_MS = 150;

// Module-level on purpose: tiles are rebuilt by virtualization and the detail view
// mounts/unmounts, but the selection must outlive both.
let state = $state.raw<SelectionState>(model.emptySelection());
let count = $state(0);
let timer: ReturnType<typeof setTimeout> | undefined;
// Responses of any request but the latest are stale; a slow earlier answer must not overwrite a newer count.
let latest = 0;

function refreshCount(): void {
  clearTimeout(timer);
  const mine = ++latest;
  if (model.isEmpty(state)) {
    count = 0;
    return;
  }
  timer = setTimeout(async () => {
    try {
      const res = await api<{ count: number }>('/api/selection/count', { method: 'POST', body: model.toRequest(state) });
      if (mine === latest) count = res.count;
    } catch {
      // Keep the last known count; the next change asks again.
    }
  }, DEBOUNCE_MS);
}

function commit(next: SelectionState): void {
  state = next;
  refreshCount();
}

/** Actions are exposed individually so any tile handler (click, long press) can drive the selection. */
export const selection = {
  get active(): boolean {
    return state.active;
  },
  get count(): number {
    return count;
  },
  get state(): SelectionState {
    return state;
  },
  isSelected: (ref: PhotoRef): boolean => model.isSelected(state, ref),
  isGroupFullySelected: (key: string): boolean => model.isGroupFullySelected(state, key),
  start: (): void => commit(model.start(state)),
  toggle: (ref: PhotoRef): void => commit(model.toggle(state, ref)),
  selectRange: (ref: PhotoRef): void => commit(model.selectRange(state, ref)),
  selectGroup: (key: string): void => commit(model.selectGroup(state, key)),
  clear: (): void => commit(model.clear()),
};
