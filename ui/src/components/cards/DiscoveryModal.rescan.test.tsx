/**
 * Rescan stays in the tab order while its scan runs (seed#2996).
 *
 * It used to take `disabled` once the scan began, so the browser dropped the
 * keyboard focus that had just pressed it onto the page body. The focus trap
 * pulled it back a frame later, but to the first control, not to Rescan.
 * jsdom has no focus fixup for disabled controls, so the focus half is in
 * e2e/a11y-dialogs.spec.ts; this pins the contract that keeps it focusable.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { DiscoveryModal } from './DiscoveryModal';
import type { NetworkDiscoveryData } from './networkDiscoveryCardTypes';

vi.mock('../../hooks/useVulnerabilities', () => ({
  useVulnerabilities: () => ({
    fetchDeviceVulnerabilities: vi.fn(),
    triggerScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

function discovery(scanning: boolean): NetworkDiscoveryData {
  return {
    devices: [],
    status: {
      scanning,
      deviceCount: 0,
      lastScan: '2026-10-01T00:00:00Z',
      subnet: '10.44.10.0/24',
      localIP: '10.44.10.2',
      interface: 'eth0',
    },
  };
}

describe('DiscoveryModal rescan', () => {
  it('starts a scan when idle', async () => {
    const user = userEvent.setup();
    const onScan = vi.fn();
    render(<DiscoveryModal isOpen onClose={() => {}} data={discovery(false)} onScan={onScan} />);

    const rescan = screen.getByTestId('discovery-rescan');
    expect(rescan).not.toHaveAttribute('aria-disabled', 'true');
    await user.click(rescan);

    expect(onScan).toHaveBeenCalledTimes(1);
  });

  it('stays focusable but inert while a scan runs', async () => {
    const user = userEvent.setup();
    const onScan = vi.fn();
    const { rerender } = render(
      <DiscoveryModal isOpen onClose={() => {}} data={discovery(false)} onScan={onScan} />,
    );
    const rescan = screen.getByTestId('discovery-rescan');
    rescan.focus();

    rerender(<DiscoveryModal isOpen onClose={() => {}} data={discovery(true)} onScan={onScan} />);

    expect(rescan).toBeEnabled();
    expect(rescan).toHaveAttribute('aria-disabled', 'true');
    expect(rescan).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(onScan).not.toHaveBeenCalled();
  });
});
