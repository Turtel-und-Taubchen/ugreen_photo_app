export const LONG_PRESS_MS = 500;
export const LONG_PRESS_SLOP_PX = 10;

/**
 * Fires `callback` once a pointer of any type stays down for LONG_PRESS_MS within
 * LONG_PRESS_SLOP_PX. The click the release produces is swallowed, otherwise the
 * long press would also trigger the tile's normal tap action.
 */
export function longpress(node: HTMLElement, callback: () => void) {
  let current = callback;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let startX = 0;
  let startY = 0;
  let swallowClick = false;
  let swallowTimer: ReturnType<typeof setTimeout> | undefined;

  const cancel = () => {
    clearTimeout(timer);
    timer = undefined;
  };

  const onDown = (e: PointerEvent) => {
    clearTimeout(swallowTimer);
    swallowClick = false;
    cancel();
    startX = e.clientX;
    startY = e.clientY;
    timer = setTimeout(() => {
      timer = undefined;
      swallowClick = true;
      current();
    }, LONG_PRESS_MS);
  };
  const onMove = (e: PointerEvent) => {
    if (timer !== undefined && Math.hypot(e.clientX - startX, e.clientY - startY) > LONG_PRESS_SLOP_PX) cancel();
  };
  const onUp = () => {
    cancel();
    // A release that yields no click (e.g. the pointer left the node) must not eat a later, unrelated click.
    if (swallowClick) swallowTimer = setTimeout(() => (swallowClick = false), 300);
  };
  const onClick = (e: MouseEvent) => {
    if (!swallowClick) return;
    swallowClick = false;
    e.preventDefault();
    e.stopImmediatePropagation();
  };
  // Mobile browsers open their own context menu on a long press; the app shows its own.
  const onContextMenu = (e: Event) => e.preventDefault();

  node.addEventListener('pointerdown', onDown);
  node.addEventListener('pointermove', onMove);
  node.addEventListener('pointerup', onUp);
  node.addEventListener('pointercancel', onUp);
  node.addEventListener('click', onClick, true);
  node.addEventListener('contextmenu', onContextMenu);

  return {
    update(next: () => void) {
      current = next;
    },
    destroy() {
      cancel();
      clearTimeout(swallowTimer);
      node.removeEventListener('pointerdown', onDown);
      node.removeEventListener('pointermove', onMove);
      node.removeEventListener('pointerup', onUp);
      node.removeEventListener('pointercancel', onUp);
      node.removeEventListener('click', onClick, true);
      node.removeEventListener('contextmenu', onContextMenu);
    },
  };
}
