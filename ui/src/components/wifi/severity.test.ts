import { describe, expect, it } from 'vitest';
import { severityStyle } from './severity';

describe('severityStyle', () => {
  it('ranks the ladder info < warning < error < critical', () => {
    expect(severityStyle('info').rank).toBe(1);
    expect(severityStyle('warning').rank).toBe(2);
    expect(severityStyle('error').rank).toBe(3);
    expect(severityStyle('critical').rank).toBe(4);
  });

  it('gives error the orange severity-high token, distinct from critical red', () => {
    expect(severityStyle('error').badge).toContain('severity-high');
    expect(severityStyle('critical').badge).toContain('status-error');
    expect(severityStyle('error').badge).not.toEqual(severityStyle('critical').badge);
  });

  it('falls back for an unknown severity', () => {
    const fb = severityStyle('nope');
    expect(fb.rank).toBe(0);
    expect(fb.badge).toContain('surface-sunken');
  });
});
