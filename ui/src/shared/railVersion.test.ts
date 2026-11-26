import { describe, expect, it } from 'vitest';
import { buildDetail, formatVersion } from './railVersion';

describe('formatVersion', () => {
  it.each([
    ['0.285.3', 'v0.285.3'],
    ['v0.285.3', 'v0.285.3'],
    ['v0.285.3-31-gf4194afb-dirty', 'v0.285.3'],
    ['v0.285.3-31-gf4194afb', 'v0.285.3'],
    ['1.2.3+build.5', 'v1.2.3'],
    ['v1.0.0-rc.1', 'v1.0.0-rc.1'],
    ['v1.0.0-rc.1-4-gabc1234-dirty', 'v1.0.0-rc.1'],
    ['dev', 'dev'],
    ['unknown', 'unknown'],
  ])('%s is shown as %s', (raw, shown) => {
    expect(formatVersion(raw)).toBe(shown);
  });

  it('never shows a dirty flag or a commit', () => {
    expect(formatVersion('v0.285.3-31-gf4194afb-dirty')).not.toMatch(/dirty|-g[0-9a-f]{7}/);
  });
});

describe('buildDetail', () => {
  it('is the raw string when it says more than the visible line', () => {
    expect(buildDetail('v0.285.3-31-gf4194afb-dirty')).toBe('v0.285.3-31-gf4194afb-dirty');
  });

  it('is nothing for a clean release', () => {
    expect(buildDetail('v0.285.3')).toBeUndefined();
    expect(buildDetail('dev')).toBeUndefined();
  });
});
