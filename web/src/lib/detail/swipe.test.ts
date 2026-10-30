import { describe, expect, it } from 'vitest';
import { swipeDirection } from './swipe';

describe('swipeDirection', () => {
  it('needs 50 px and a mostly horizontal move', () => {
    expect(swipeDirection(-50, 0)).toBe('left');
    expect(swipeDirection(80, 20)).toBe('right');
    expect(swipeDirection(49, 0)).toBeNull();
    expect(swipeDirection(60, 70)).toBeNull();
  });
});
