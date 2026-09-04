/**
 * LicenseContext tests — cover the fetch-on-authenticated contract and the
 * fail-closed defaults, plus the auth gating that stops the login screen
 * (and any other unauthenticated render) from ever calling GET /license.
 *
 * Companion to RoleContext.test.tsx's "auth gating" suite: both providers
 * were fetching unconditionally on mount before authentication settled,
 * which is what produced the spurious 401 storm behind seed#2422.
 */

import { render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider, type LicenseStatus, useLicense } from './LicenseContext';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

const status = (overrides: Partial<LicenseStatus> = {}): LicenseStatus => ({
  tier: 'free',
  tierValue: 0,
  isTrialMode: false,
  canMintTokens: false,
  activated: false,
  ...overrides,
});

function Consumer(): React.ReactElement {
  const { status: licenseStatus, loading } = useLicense();
  return <span>{loading ? 'loading' : (licenseStatus?.tier ?? 'none')}</span>;
}

describe('LicenseContext / useLicense', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('throws if useLicense is called outside LicenseProvider', () => {
    const orig = console.error;
    console.error = (): void => {}; // silence React's expected error boundary log
    try {
      expect(() => render(<Consumer />)).toThrow(/inside <LicenseProvider>/);
    } finally {
      console.error = orig;
    }
  });
});

describe('LicenseProvider auth gating', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('does not fetch /license while unauthenticated', () => {
    render(
      <LicenseProvider isAuthenticated={false}>
        <span>child</span>
      </LicenseProvider>,
    );
    expect(mockGet).not.toHaveBeenCalled();
  });

  it('fetches /license exactly once when rendered authenticated', async () => {
    mockGet.mockResolvedValueOnce(status({ tier: 'pro' }));
    render(
      <LicenseProvider isAuthenticated={true}>
        <span>child</span>
      </LicenseProvider>,
    );
    await waitFor(() => expect(mockGet).toHaveBeenCalledTimes(1));
    expect(mockGet).toHaveBeenCalledWith('/api/v1/license');
  });

  it('fetches exactly once when isAuthenticated flips from false to true', async () => {
    mockGet.mockResolvedValueOnce(status({ tier: 'pro' }));
    const { rerender } = render(
      <LicenseProvider isAuthenticated={false}>
        <span>child</span>
      </LicenseProvider>,
    );
    expect(mockGet).not.toHaveBeenCalled();

    rerender(
      <LicenseProvider isAuthenticated={true}>
        <span>child</span>
      </LicenseProvider>,
    );
    await waitFor(() => expect(mockGet).toHaveBeenCalledTimes(1));
  });
});
