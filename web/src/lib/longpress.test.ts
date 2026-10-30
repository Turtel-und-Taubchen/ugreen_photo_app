import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest';
import { longpress } from './longpress';

// The vitest environment is node, so the element is a minimal stand-in that only dispatches to registered listeners.
class FakeNode {
  private listeners = new Map<string, ((e: unknown) => void)[]>();
  addEventListener(type: string, fn: (e: unknown) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), fn]);
  }
  removeEventListener() {}
  emit(type: string, x = 0, y = 0) {
    const event = { type, clientX: x, clientY: y, preventDefault: vi.fn(), stopImmediatePropagation: vi.fn() };
    for (const fn of this.listeners.get(type) ?? []) fn(event);
    return event;
  }
}

describe('longpress', () => {
  let node: FakeNode;
  let fired: Mock<() => void>;

  beforeEach(() => {
    vi.useFakeTimers();
    node = new FakeNode();
    fired = vi.fn<() => void>();
    longpress(node as unknown as HTMLElement, fired);
  });
  afterEach(() => vi.useRealTimers());

  it('a press held 500 ms fires once and swallows the following click', () => {
    node.emit('pointerdown', 10, 10);
    vi.advanceTimersByTime(499);
    expect(fired).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(fired).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1000);
    node.emit('pointerup', 10, 10);
    expect(node.emit('click').stopImmediatePropagation).toHaveBeenCalled();
    expect(fired).toHaveBeenCalledTimes(1);
    // Only the click directly after the press is swallowed.
    expect(node.emit('click').stopImmediatePropagation).not.toHaveBeenCalled();
  });

  it('moving more than 10 px cancels the long press', () => {
    node.emit('pointerdown', 10, 10);
    node.emit('pointermove', 10, 21);
    vi.advanceTimersByTime(700);
    node.emit('pointerup', 10, 21);
    expect(fired).not.toHaveBeenCalled();
    expect(node.emit('click').stopImmediatePropagation).not.toHaveBeenCalled();
  });

  it('a short press leaves the click alone', () => {
    node.emit('pointerdown', 10, 10);
    vi.advanceTimersByTime(200);
    node.emit('pointerup', 10, 10);
    expect(fired).not.toHaveBeenCalled();
    expect(node.emit('click').stopImmediatePropagation).not.toHaveBeenCalled();
  });

  it('the context menu of the browser is suppressed', () => {
    expect(node.emit('contextmenu').preventDefault).toHaveBeenCalled();
  });
});
