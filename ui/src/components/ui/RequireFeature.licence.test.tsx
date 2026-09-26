/**
 * RequireFeature over the real LicenseContext — seed#2688.
 *
 * The gate is only as good as the wire answer behind it. Mocking
 * useLicense would have kept passing while `/api/v1/license` sent no
 * `features` at all, which is exactly how Pro and Trial users ended up
 * looking at the Free gate on Path Analysis and Reports. These tests
 * drive the provider with the real endpoint payload instead.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider } from '../../contexts/LicenseContext';
import type { LicenseStatusResponse } from '../../types/generated/license-status-response';
import { RequireFeature } from './RequireFeature';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

// The Pro/Trial slice of internal/license.proFeatures() the pages gate on.
const proFeatures = [
  'export_csv_json',
  'path_analysis',
  'wifi_analysis',
  'wifi_association_forensics',
];

function payload(overrides: Partial<LicenseStatusResponse> = {}): LicenseStatusResponse {
  return {
    tier: 'Free',
    tierValue: 0,
    isTrialMode: false,
    canMintTokens: false,
    activated: false,
    features: [],
    ...overrides,
  };
}

function renderGate(status: LicenseStatusResponse): void {
  mockGet.mockResolvedValue(status);
  render(
    <LicenseProvider isAuthenticated>
      <RequireFeature feature="path_analysis" fallback={<span>upgrade</span>}>
        <span>path analysis</span>
      </RequireFeature>
    </LicenseProvider>,
  );
}

describe('RequireFeature over GET /api/v1/license', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('unlocks the surface on Pro', async () => {
    renderGate(payload({ tier: 'Pro', tierValue: 2, activated: true, features: proFeatures }));
    expect(await screen.findByText('path analysis')).toBeInTheDocument();
    expect(screen.queryByText('upgrade')).not.toBeInTheDocument();
  });

  it('unlocks the surface during a trial', async () => {
    renderGate(
      payload({
        tier: 'Trial',
        tierValue: 2,
        isTrialMode: true,
        trialDaysLeft: 12,
        activated: true,
        features: proFeatures,
      }),
    );
    expect(await screen.findByText('path analysis')).toBeInTheDocument();
  });

  it('keeps the gate on Free', async () => {
    renderGate(payload());
    expect(await screen.findByText('upgrade')).toBeInTheDocument();
    expect(screen.queryByText('path analysis')).not.toBeInTheDocument();
  });

  it('keeps the gate when the endpoint sends no features at all', async () => {
    // The pre-fix response shape. Pinned so a regression that drops the
    // field again fails closed here rather than silently in production.
    const { features: _omitted, ...withoutFeatures } = payload({
      tier: 'Pro',
      tierValue: 2,
      activated: true,
    });
    mockGet.mockResolvedValue(withoutFeatures);
    render(
      <LicenseProvider isAuthenticated>
        <RequireFeature feature="path_analysis" fallback={<span>upgrade</span>}>
          <span>path analysis</span>
        </RequireFeature>
      </LicenseProvider>,
    );
    await waitFor(() => {
      expect(screen.getByText('upgrade')).toBeInTheDocument();
    });
  });
});
