/**
 * The discovery table's CVE badge opens the device's findings (seed#2640).
 *
 * `VulnerabilityDetailsModal` was rendered by NetworkDiscoveryCard behind state
 * that was only ever set to null, so nothing could open it, and it was a bare
 * overlay with no dialog role, Escape or focus trap. The badge is now the
 * opener and the modal is built on the shared `Modal`. These drive the real
 * DiscoveryModal: the badge must open a named dialog without toggling the row,
 * and Escape must close only that dialog and hand focus back to the badge.
 * Focus moving INTO the dialog needs layout (jsdom has no offsetParent), so
 * e2e/a11y-dialogs.spec.ts covers that half in real browsers.
 */

import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { DeviceVulnerabilities } from '../../types/vulnerabilities';
import { DiscoveryModal } from './DiscoveryModal';
import type { NetworkDiscoveryData } from './networkDiscoveryCardTypes';

const findings: DeviceVulnerabilities = {
  deviceIp: '10.44.10.7',
  mac: '00:1b:21:aa:bb:01',
  hostname: 'sw-core-01',
  vendor: 'Cisco',
  product: 'IOS',
  version: '15.2',
  scanTime: '2026-10-01T00:00:00Z',
  vulnerabilities: [
    {
      cveId: 'CVE-2026-0002',
      description: 'test finding',
      severity: 'CRITICAL',
      score: 9.8,
      published: '2026-01-01T00:00:00Z',
      modified: '2026-01-01T00:00:00Z',
      references: [],
      affectedCpe: '',
    },
  ],
};

const fetchDeviceVulnerabilities = vi.fn().mockResolvedValue(findings);

vi.mock('../../hooks/useVulnerabilities', () => ({
  useVulnerabilities: () => ({
    fetchDeviceVulnerabilities,
    triggerScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

const data: NetworkDiscoveryData = {
  devices: [
    {
      ip: '10.44.10.7',
      mac: '00:1b:21:aa:bb:01',
      hostname: 'sw-core-01',
      lastSeen: '2026-10-01T00:00:00Z',
      discoveryMethod: ['arp'],
      isLocal: true,
      vulnerabilities: findings,
    },
  ],
  status: {
    scanning: false,
    deviceCount: 1,
    lastScan: '2026-10-01T00:00:00Z',
    subnet: '10.44.10.0/24',
    localIP: '10.44.10.2',
    interface: 'eth0',
  },
};

function renderModal() {
  render(<DiscoveryModal isOpen onClose={() => {}} data={data} />);
  // The badge renders in the column and in the row's folded summary; the
  // column copy is the first.
  const [badge] = screen.getAllByRole('button', { name: 'Show vulnerabilities for sw-core-01' });
  if (!badge) {
    throw new Error('no vulnerability badge');
  }
  return badge;
}

describe('DiscoveryModal vulnerability details', () => {
  it('opens the findings as a named dialog without toggling the row', async () => {
    const user = userEvent.setup();
    const badge = renderModal();

    await user.click(badge);

    const dialog = await screen.findByRole('dialog', { name: 'Vulnerability Report' });
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(fetchDeviceVulnerabilities).toHaveBeenCalledWith('10.44.10.7');
    expect(await within(dialog).findByText('CVE-2026-0002')).toBeInTheDocument();
    expect(screen.getByTestId('discovery-row-toggle')).toHaveAttribute('aria-expanded', 'false');
  });

  it('closes only the findings on Escape and returns focus to the badge', async () => {
    const user = userEvent.setup();
    const badge = renderModal();

    await user.click(badge);
    await screen.findByRole('dialog', { name: 'Vulnerability Report' });

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('dialog', { name: 'Vulnerability Report' })).not.toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: 'Network Discovery' })).toBeInTheDocument();
    expect(badge).toHaveFocus();
  });
});
