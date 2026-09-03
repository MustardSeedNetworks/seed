/**
 * FeatureUnavailable / LimitedFeature tests (#750).
 *
 * The requirement these exist to meet is that a platform gap stops being a
 * silent failure. The assertions are therefore about the *explanation* reaching
 * the screen, and about the platform axis staying distinct from the licence one.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { FeatureUnavailable } from './FeatureUnavailable';
import { LimitedFeature } from './LimitedFeature';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));
vi.mock('../../api', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));

function withCapabilities(entries: unknown[]): void {
  mockGet.mockResolvedValue({ capabilities: entries });
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('FeatureUnavailable', () => {
  it('replaces the feature and gives the backend’s reason', async () => {
    withCapabilities([
      {
        capability: 'cable_diagnostics',
        title: 'Cable diagnostics (TDR)',
        level: 'none',
        note: 'No macOS API exposes TDR.',
      },
    ]);
    render(
      <FeatureUnavailable capability="cable_diagnostics">
        <p>Cable test</p>
      </FeatureUnavailable>,
    );

    expect(
      await screen.findByText('Cable diagnostics (TDR) is not available on this platform'),
    ).toBeInTheDocument();
    // The note comes from internal/capabilities, so the API, the banner and
    // HARDWARE.md all say the same words.
    expect(screen.getByText('No macOS API exposes TDR.')).toBeInTheDocument();
    expect(screen.queryByText('Cable test')).not.toBeInTheDocument();
  });

  it('renders the feature when the platform supports it', async () => {
    withCapabilities([
      { capability: 'cable_diagnostics', title: 'Cable diagnostics (TDR)', level: 'full' },
    ]);
    render(
      <FeatureUnavailable capability="cable_diagnostics">
        <p>Cable test</p>
      </FeatureUnavailable>,
    );

    await waitFor(() => {
      expect(screen.getByText('Cable test')).toBeInTheDocument();
    });
    expect(screen.queryByTestId('feature-unavailable')).not.toBeInTheDocument();
  });

  it('does not flash "unavailable" while the report is loading', () => {
    mockGet.mockReturnValue(new Promise(() => undefined));
    render(
      <FeatureUnavailable capability="cable_diagnostics">
        <p>Cable test</p>
      </FeatureUnavailable>,
    );

    // The report describes the OS, which cannot change under us — guessing
    // "unavailable" first would be wrong more often than right.
    expect(screen.getByText('Cable test')).toBeInTheDocument();
    expect(screen.queryByTestId('feature-unavailable')).not.toBeInTheDocument();
  });
});

describe('LimitedFeature', () => {
  it('keeps the feature and says what is reduced', async () => {
    withCapabilities([
      {
        capability: 'speed_duplex',
        title: 'Speed/duplex detection',
        level: 'partial',
        note: 'Reports negotiated speed; duplex is not exposed.',
      },
    ]);
    render(
      <LimitedFeature capability="speed_duplex">
        <p>Speed and duplex</p>
      </LimitedFeature>,
    );

    expect(
      await screen.findByText('Reports negotiated speed; duplex is not exposed.'),
    ).toBeInTheDocument();
    // Unlike FeatureUnavailable, the feature itself stays on screen.
    expect(screen.getByText('Speed and duplex')).toBeInTheDocument();
  });

  it('annotates limited the same as partial', async () => {
    withCapabilities([
      {
        capability: 'vlan_detection',
        title: 'VLAN detection',
        level: 'limited',
        note: "Depends on the NIC vendor's driver exposing tagged interfaces.",
      },
    ]);
    render(
      <LimitedFeature capability="vlan_detection">
        <p>VLANs</p>
      </LimitedFeature>,
    );

    expect(await screen.findByTestId('limited-feature')).toBeInTheDocument();
  });

  it('adds nothing when the platform supports the feature fully', async () => {
    withCapabilities([
      { capability: 'speed_duplex', title: 'Speed/duplex detection', level: 'full' },
    ]);
    render(
      <LimitedFeature capability="speed_duplex">
        <p>Speed and duplex</p>
      </LimitedFeature>,
    );

    await waitFor(() => {
      expect(screen.getByText('Speed and duplex')).toBeInTheDocument();
    });
    expect(screen.queryByTestId('limited-feature')).not.toBeInTheDocument();
  });

  it('treats an unknown capability as unsupported rather than fine', async () => {
    withCapabilities([]);
    render(
      <FeatureUnavailable capability="something_the_backend_never_described">
        <p>Mystery feature</p>
      </FeatureUnavailable>,
    );

    expect(await screen.findByTestId('feature-unavailable')).toBeInTheDocument();
  });
});
