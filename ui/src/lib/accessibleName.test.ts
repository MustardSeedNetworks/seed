import { describe, expect, it } from 'vitest';
import { accessibleNameIncluding } from './accessibleName';

describe('accessibleNameIncluding', () => {
  it('leaves a name that already contains the visible words in order', () => {
    expect(accessibleNameIncluding('Open help', 'Help')).toBe('Open help');
    expect(accessibleNameIncluding('Start network scan', 'Scan')).toBe('Start network scan');
  });

  it('appends a visible label the description does not contain', () => {
    expect(accessibleNameIncluding('Start network discovery scan', 'Start Scan')).toBe(
      'Start network discovery scan Start Scan',
    );
    expect(accessibleNameIncluding('Select device', '192.168.1.1')).toBe(
      'Select device 192.168.1.1',
    );
  });

  it('does not treat a longer word as containing a shorter one', () => {
    expect(accessibleNameIncluding('Scanning network', 'Scan')).toBe('Scanning network Scan');
  });

  it('ignores parenthetical text the way axe does', () => {
    expect(accessibleNameIncluding('Select interface eth0 (1000 Mbps)', 'eth0 (1000 Mbps)')).toBe(
      'Select interface eth0 (1000 Mbps)',
    );
  });

  it('returns the description when nothing is visible', () => {
    expect(accessibleNameIncluding('Select profile', '   ')).toBe('Select profile');
  });
});
