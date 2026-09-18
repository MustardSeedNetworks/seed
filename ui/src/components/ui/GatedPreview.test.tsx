/**
 * GatedPreview over the real LicenseContext — seed#2669.
 *
 * Driven through the provider and the real `/api/v1/license` payload for the
 * same reason RequireFeature.licence.test.tsx is: mocking useLicense keeps
 * passing while the wire answer carries no `features` at all, which is how
 * paying users ended up looking at the Free gate (#2688).
 */

import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider } from '../../contexts/LicenseContext';
import type { LicenseStatusResponse } from '../../types/generated/license-status-response';
import { GatedPreview } from './GatedPreview';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

function payload(features: string[]): LicenseStatusResponse {
  return {
    tier: features.length > 0 ? 'Pro' : 'Free',
    tierValue: features.length > 0 ? 2 : 0,
    isTrialMode: false,
    canMintTokens: false,
    activated: features.length > 0,
    features,
  };
}

function renderGate(features: string[]): void {
  mockGet.mockResolvedValue(payload(features));
  render(
    <LicenseProvider isAuthenticated>
      <GatedPreview feature="path_analysis" preview={<button type="button">sample hop</button>}>
        <span>live path analysis</span>
      </GatedPreview>
    </LicenseProvider>,
  );
}

describe('GatedPreview', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('shows the pitch and a non-empty sample when the feature is not licensed', async () => {
    renderGate([]);
    await waitFor(() => expect(screen.getByTestId('gated-pitch')).toBeVisible());
    expect(screen.getByTestId('gated-preview')).toHaveAttribute('data-feature', 'path_analysis');
    expect(screen.getByText('sample hop')).toBeVisible();
    expect(screen.queryByText('live path analysis')).toBeNull();
  });

  it('names the tier the licence policy grants the feature in', async () => {
    renderGate([]);

    await waitFor(() => expect(screen.getByText('Path Analysis is a Pro feature')).toBeVisible());
  });

  it('takes the sample out of the tab order and the accessibility tree', async () => {
    renderGate([]);

    await waitFor(() => expect(screen.getByText('sample hop')).toBeVisible());
    // A sample the keyboard can reach is a page whose controls do nothing.
    const sample = screen.getByText('sample hop').parentElement;
    expect(sample).toHaveAttribute('inert');
    expect(sample).toHaveAttribute('aria-hidden', 'true');
  });

  it('renders the real body, and no pitch, when the feature is licensed', async () => {
    renderGate(['path_analysis']);

    await waitFor(() => expect(screen.getByText('live path analysis')).toBeVisible());
    expect(screen.queryByTestId('gated-preview')).toBeNull();
    expect(screen.queryByText('sample hop')).toBeNull();
  });

  it('renders neither half while the licence fetch is in flight', () => {
    mockGet.mockReturnValue(new Promise(() => undefined));
    render(
      <LicenseProvider isAuthenticated>
        <GatedPreview feature="path_analysis" preview={<span>sample hop</span>}>
          <span>live path analysis</span>
        </GatedPreview>
      </LicenseProvider>,
    );

    // Children would fire the gated fetches; the pitch would flash at a Pro user.
    expect(screen.queryByText('live path analysis')).toBeNull();
    expect(screen.queryByTestId('gated-preview')).toBeNull();
  });
});
