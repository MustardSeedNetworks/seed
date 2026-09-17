/**
 * DiscoveryEmptyState tests (seed#2674).
 *
 * The owner's report was that a fresh install shows nothing and no page says
 * why. "Nothing" has three causes and they need different words: discovery is
 * switched off, discovery is running and has not finished a sweep yet, or a
 * sweep finished and the segment really is empty. The old copy — "No devices
 * discovered. Click Scan" — was the third for all three, which is why an
 * install that had never scanned looked like an empty network.
 *
 * lastScan is the discriminator: it is Go's zero time over the wire until a
 * sweep completes (pinned in devices_status_test.go).
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { DiscoveryEmptyState, discoveryPhase } from './DiscoveryEmptyState';
import type { DiscoveryStatus } from './networkDiscoveryCardTypes';

const ZERO_TIME = '0001-01-01T00:00:00Z';

function status(overrides: Partial<DiscoveryStatus> = {}): DiscoveryStatus {
  return {
    scanning: false,
    deviceCount: 0,
    lastScan: '2026-09-16T00:00:00Z',
    subnet: '192.168.1.0/24',
    localIP: '192.168.1.5',
    interface: 'eth0',
    ...overrides,
  };
}

describe('discoveryPhase', () => {
  it('is off when discovery is disabled, whatever the status says', () => {
    expect(discoveryPhase(false, status())).toBe('off');
    expect(discoveryPhase(false, status({ scanning: true }))).toBe('off');
  });

  it('is discovering while a sweep is running', () => {
    expect(discoveryPhase(true, status({ scanning: true }))).toBe('discovering');
  });

  it('is discovering before the first sweep has finished', () => {
    expect(discoveryPhase(true, status({ lastScan: ZERO_TIME }))).toBe('discovering');
    expect(discoveryPhase(true, status({ lastScan: '' }))).toBe('discovering');
    expect(discoveryPhase(true, null)).toBe('discovering');
  });

  it('is empty once a sweep has finished and found nothing', () => {
    expect(discoveryPhase(true, status())).toBe('empty');
  });
});

describe('DiscoveryEmptyState', () => {
  it('says discovery is off and offers the options', async () => {
    const onOpenSettings = vi.fn();
    render(<DiscoveryEmptyState phase="off" onOpenSettings={onOpenSettings} />);

    expect(screen.getByTestId('discovery-empty-off')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('discovery-open-options'));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it('says it is still discovering, and offers no options button for a state that will pass', () => {
    render(<DiscoveryEmptyState phase="discovering" onOpenSettings={vi.fn()} />);

    expect(screen.getByTestId('discovery-empty-discovering')).toBeInTheDocument();
    expect(screen.queryByTestId('discovery-open-options')).not.toBeInTheDocument();
  });

  it('says the sweep found nothing and offers the options', async () => {
    const onOpenSettings = vi.fn();
    render(<DiscoveryEmptyState phase="empty" onOpenSettings={onOpenSettings} />);

    expect(screen.getByTestId('discovery-empty-none')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('discovery-open-options'));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it('renders no options button when the caller cannot open them', () => {
    render(<DiscoveryEmptyState phase="empty" />);

    expect(screen.getByTestId('discovery-empty-none')).toBeInTheDocument();
    expect(screen.queryByTestId('discovery-open-options')).not.toBeInTheDocument();
  });
});
