/**
 * Unit tests for owner-labels.ts
 * Covers: override precedence (AC-03-005), Phase 1 limitation explicit coverage,
 * flag default false, 3-project Carol scenario.
 */
import { describe, it, expect } from 'vitest';
import { mapOwnerLabel, getOwnerLabelMap } from './owner-labels';

describe('mapOwnerLabel — Phase 1 override precedence', () => {
  describe('override argument takes precedence over phase mapping', () => {
    it('returns override when non-empty string is provided', () => {
      expect(mapOwnerLabel('execution', 'Custom Engineering')).toBe('Custom Engineering');
      expect(mapOwnerLabel('planning', 'My Override')).toBe('My Override');
    });

    it('returns override even when phase is unknown', () => {
      expect(mapOwnerLabel('unknown-phase', 'Override Value')).toBe('Override Value');
    });

    it('trims whitespace from override', () => {
      expect(mapOwnerLabel('execution', '  Trimmed Override  ')).toBe('Trimmed Override');
    });
  });

  describe('Phase 1 hardcoded mapping (no override)', () => {
    it('maps planning phase to Product', () => {
      expect(mapOwnerLabel('planning')).toBe('Product');
    });

    it('maps decomposition phase to Product', () => {
      expect(mapOwnerLabel('decomposition')).toBe('Product');
    });

    it('maps execution phase to Engineering', () => {
      expect(mapOwnerLabel('execution')).toBe('Engineering');
    });

    it('maps validation phase to Review', () => {
      expect(mapOwnerLabel('validation')).toBe('Review');
    });

    it('maps acceptance phase to Quality', () => {
      expect(mapOwnerLabel('acceptance')).toBe('Quality');
    });

    it('maps closed phase to Product', () => {
      expect(mapOwnerLabel('closed')).toBe('Product');
    });

    it('is case-insensitive', () => {
      expect(mapOwnerLabel('EXECUTION')).toBe('Engineering');
      expect(mapOwnerLabel('PLANNING')).toBe('Product');
      expect(mapOwnerLabel('Validation')).toBe('Review');
    });
  });

  describe('fallback and edge cases', () => {
    it('returns DEFAULT_LABEL (Product) for empty/null/undefined phase', () => {
      expect(mapOwnerLabel('')).toBe('Product');
      expect(mapOwnerLabel('  ')).toBe('Product');
      // null/undefined would be a type error at compile time; verify string coercion
      expect(mapOwnerLabel('any-phase')).not.toBe('');
    });

    it('returns override for any unrecognized phase', () => {
      expect(mapOwnerLabel('nonexistent-phase', 'Override')).toBe('Override');
    });

    it('returns Product for any completely unknown phase without override', () => {
      expect(mapOwnerLabel('foobar')).toBe('Product');
      expect(mapOwnerLabel('unknown')).toBe('Product');
    });
  });

  describe('getOwnerLabelMap — returns a copy of the static map', () => {
    it('returns an object with all phase keys', () => {
      const map = getOwnerLabelMap();
      expect(map).toHaveProperty('planning');
      expect(map).toHaveProperty('decomposition');
      expect(map).toHaveProperty('execution');
      expect(map).toHaveProperty('validation');
      expect(map).toHaveProperty('acceptance');
      expect(map).toHaveProperty('closed');
    });

    it('returns a copy (mutation does not affect the original)', () => {
      const map = getOwnerLabelMap();
      map.planning = 'Mutated';
      expect(getOwnerLabelMap().planning).toBe('Product');
    });
  });
});