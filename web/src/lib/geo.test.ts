import { describe, expect, it } from 'vitest';
import { geoNotice, parseCoordinate } from './geo';

describe('geo', () => {
  it('the place search notice explains why the search is disabled', () => {
    expect(geoNotice({ enabled: true })).toBeNull();
    expect(geoNotice(null)).toBeNull();
    const text = geoNotice({ enabled: false, reason: 'Setze NOMINATIM_CONTACT, um die Ortssuche zu nutzen.' });
    expect(text).toContain('NOMINATIM_CONTACT');
  });

  it('coordinates accept a decimal comma and reject empty or non-numeric input', () => {
    expect(parseCoordinate(' 47,4262 ')).toBe(47.4262);
    expect(parseCoordinate('')).toBeNull();
    expect(parseCoordinate('abc')).toBeNull();
  });
});
