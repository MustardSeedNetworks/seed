/**
 * NetworkDiscoveryCard error-surfacing tests (seed#2394).
 *
 * A failed device scan used to log, clear `scanning`, and spread `...prev`
 * -- the card silently reverted to its previous contents, indistinguishable
 * from a scan that ran and found nothing. This asserts the card's own half
 * of the fix (useDeviceScan.test.ts covers the hook half): given
 * `scanError`, it renders a role="alert" banner by testid and leaves the
 * scan button enabled to retry; given no error, it renders neither.
 *
 * Exercised directly rather than through e2e: until #2674 the card's only
 * mount point was the /path route, gated behind the Pro-tier `path_analysis`
 * feature, and the E2E suite runs unlicensed (Free) -- same blocker
 * documented in e2e/reports-page.spec.ts for export_csv_json. See the
 * fixme comments in e2e/error-scenarios.spec.ts. It now also mounts on
 * /network and /security, which the page suites cover.
 *
 * useEngineScan and useEnginePhase are mocked out (rather than relying on
 * the global EventSource polyfill in src/test/setup.ts) because that
 * polyfill's mock implementation is an arrow function, which the DOM
 * EventSource contract requires to be constructable via `new` -- arrow
 * functions never are, so any component that mounts them directly throws
 * "is not a constructor". No prior card test rendered a useJobEvents
 * consumer directly, so this had not surfaced before.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { NetworkDiscoveryCard } from './NetworkDiscoveryCard';
import type { NetworkDiscoveryData } from './networkDiscoveryCardTypes';

vi.mock('../../hooks/useEngineScan', () => ({
  useEngineScan: () => ({
    running: false,
    status: { state: 'idle', jobId: '', percentComplete: 0, error: null },
    startScan: vi.fn().mockResolvedValue(undefined),
    cancelScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('../../hooks/useEnginePhase', () => ({
  useEnginePhase: () => ({ phase: '' }),
}));

vi.mock('../../hooks/useNetworkDiscoveryAutoScan', () => ({
  useNetworkDiscoveryAutoScan: () => ({
    handleDeepScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

const populatedData: NetworkDiscoveryData = {
  devices: [
    {
      ip: '192.168.1.10',
      mac: '00:11:22:33:44:55',
      hostname: 'test-device',
      vendor: 'Test Vendor',
      lastSeen: '2026-09-04T00:00:00Z',
      discoveryMethod: ['arp'],
      isLocal: true,
    },
  ],
  status: {
    scanning: false,
    deviceCount: 1,
    lastScan: '2026-09-04T00:00:00Z',
    subnet: '192.168.1.0/24',
    localIP: '192.168.1.5',
    interface: 'eth0',
  },
};

describe('NetworkDiscoveryCard scan error banner', () => {
  it('renders the failed-scan alert and keeps the scan button enabled to retry', () => {
    render(<NetworkDiscoveryCard data={populatedData} scanError={true} onScan={() => {}} />);

    const banner = screen.getByTestId('discovery-scan-error');
    expect(banner).toHaveAttribute('role', 'alert');
    expect(screen.getByTestId('discovery-scan-button')).toBeEnabled();
  });

  it('renders no banner when the last scan did not fail', () => {
    render(<NetworkDiscoveryCard data={populatedData} scanError={false} onScan={() => {}} />);

    expect(screen.queryByTestId('discovery-scan-error')).not.toBeInTheDocument();
  });

  it('renders the failed-scan alert in the no-data state too', () => {
    render(<NetworkDiscoveryCard data={null} scanError={true} onScan={() => {}} />);

    const banner = screen.getByTestId('discovery-scan-error');
    expect(banner).toHaveAttribute('role', 'alert');
    expect(screen.getByTestId('discovery-scan-button')).toBeEnabled();
  });

  it('renders no banner in the no-data state when there is no error', () => {
    render(<NetworkDiscoveryCard data={null} scanError={false} onScan={() => {}} />);

    expect(screen.queryByTestId('discovery-scan-error')).not.toBeInTheDocument();
  });
});

const ZERO_TIME = '0001-01-01T00:00:00Z';

/** emptyData is a status with no devices; `lastScan` says why (seed#2674). */
function emptyData(lastScan: string): NetworkDiscoveryData {
  return {
    devices: [],
    status: {
      scanning: false,
      deviceCount: 0,
      lastScan,
      subnet: '192.168.1.0/24',
      localIP: '192.168.1.5',
      interface: 'eth0',
    },
  };
}

/**
 * The owner's seed#2674 report: a fresh install shows nothing and no page says
 * why. All three causes of "nothing" used to render the same sentence, so an
 * install still on its first sweep was indistinguishable from an empty network
 * and from discovery being switched off entirely.
 */
describe('NetworkDiscoveryCard empty states', () => {
  it('says it is still discovering before the first sweep finishes', () => {
    render(<NetworkDiscoveryCard data={emptyData(ZERO_TIME)} onScan={() => {}} />);

    expect(screen.getByTestId('discovery-empty-discovering')).toBeInTheDocument();
    expect(screen.queryByTestId('discovery-empty-none')).not.toBeInTheDocument();
  });

  it('says the sweep found nothing once one has run, and links to the options', async () => {
    const onOpenSettings = vi.fn();
    render(
      <NetworkDiscoveryCard
        data={emptyData('2026-09-16T00:00:00Z')}
        onScan={() => {}}
        onOpenSettings={onOpenSettings}
      />,
    );

    expect(screen.getByTestId('discovery-empty-none')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('discovery-open-options'));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it('says discovery is off rather than reporting an empty network', async () => {
    const onOpenSettings = vi.fn();
    render(
      <NetworkDiscoveryCard
        data={emptyData('2026-09-16T00:00:00Z')}
        discoveryEnabled={false}
        onScan={() => {}}
        onOpenSettings={onOpenSettings}
      />,
    );

    expect(screen.getByTestId('discovery-empty-off')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('discovery-open-options'));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it('says discovery is off in the no-data state too', () => {
    render(<NetworkDiscoveryCard data={null} discoveryEnabled={false} onScan={() => {}} />);

    expect(screen.getByTestId('discovery-empty-off')).toBeInTheDocument();
  });

  it('renders no empty state once devices are found', () => {
    render(<NetworkDiscoveryCard data={populatedData} onScan={() => {}} />);

    expect(screen.queryByTestId('discovery-empty-discovering')).not.toBeInTheDocument();
    expect(screen.queryByTestId('discovery-empty-none')).not.toBeInTheDocument();
    expect(screen.queryByTestId('discovery-empty-off')).not.toBeInTheDocument();
  });
});
