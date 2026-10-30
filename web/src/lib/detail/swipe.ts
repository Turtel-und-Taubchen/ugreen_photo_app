export const SWIPE_MIN_PX = 50;

/** A swipe counts when it moves at least 50 px and is mostly horizontal; moving left means "left". */
export function swipeDirection(dx: number, dy: number): 'left' | 'right' | null {
  if (Math.abs(dx) < SWIPE_MIN_PX || Math.abs(dx) <= Math.abs(dy)) return null;
  return dx < 0 ? 'left' : 'right';
}
