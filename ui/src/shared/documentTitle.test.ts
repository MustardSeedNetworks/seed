import { describe, expect, it } from 'vitest';
import { documentTitle } from './documentTitle';

describe('documentTitle', () => {
  it('puts the page name first and the bare product after a middle dot', () => {
    expect(documentTitle('Devices', 'NIAC')).toBe('Devices \u00b7 NIAC');
    expect(documentTitle('History', 'Stem')).toBe('History \u00b7 Stem');
    expect(documentTitle('Surveys', 'Trellis')).toBe('Surveys \u00b7 Trellis');
    expect(documentTitle('Dashboard', 'Seed')).toBe('Dashboard \u00b7 Seed');
  });

  it('is the product alone when the page has no name of its own', () => {
    expect(documentTitle('', 'Seed')).toBe('Seed');
  });

  it('never uses a pipe, which screen readers announce', () => {
    expect(documentTitle('A', 'B')).not.toContain('|');
  });
});
