/**
 * disabledTokens.test.ts — a disabled control is drawn with its own tokens.
 *
 * Disabled controls used to be the enabled control at opacity-50, which took
 * the label down with the layer (#1944). Owner 2026-10-14: a fill, an edge and
 * text-disabled per theme, applied by the control-disabled utility. WCAG 1.4.3
 * exempts inactive controls, so the floor is legibility, not AA: the label
 * keeps 2.5:1 on its fill.
 */
import { describe, expect, it } from 'vitest';
import { contrast, palette, rgb } from '../test/palette';

const LABEL_FLOOR = 2.5;

describe('disabled control tokens', () => {
  for (const mode of ['light', 'dark'] as const) {
    const tokens = palette(mode);
    const hex = (name: string): string => {
      const value = tokens.get(name);
      expect(value, `--color-${name} is not defined in ${mode} mode`).toBeDefined();
      return value ?? '';
    };

    it(`${mode}: text-disabled is legible on bg-disabled`, () => {
      const ratio = contrast(rgb(hex('text-disabled')), rgb(hex('bg-disabled')));
      expect(ratio).toBeGreaterThanOrEqual(LABEL_FLOOR);
    });
  }
});
