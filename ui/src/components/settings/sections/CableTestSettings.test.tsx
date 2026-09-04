/**
 * CableTestSettings TDR-support tests (seed#2326).
 *
 * The component asked GET /api/v1/telemetry/cable/support for the answer. That
 * route has never been registered — the only cable route is
 * /api/v1/telemetry/cable, which runs a test rather than reporting whether one
 * is possible — so the request 404'd and TDR support read as unavailable on
 * every host, including the Linux hosts that do support it. Verified against a
 * running daemon: authenticated, that path answers 404 while /api/v1/status
 * answers 200 with the capability report.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { CableTestSettings as CableTestSettingsType } from '../../../types/settings';
import { CableTestSettings } from './CableTestSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();

vi.mock('../../../api', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

const settings: CableTestSettingsType = { enabled: true };

function renderWithCapability(entry: unknown): void {
  mockGet.mockImplementation((path: string) => {
    if (path === '/api/v1/status') {
      return Promise.resolve({ capabilities: entry === null ? [] : [entry] });
    }

    return Promise.reject(new Error(`unexpected request to ${path}`));
  });

  render(
    <CableTestSettings
      cableTestSettings={settings}
      setCableTestSettings={vi.fn()}
      cableTestStatus="idle"
    />,
  );
}

/** The section ships collapsed, so its body is not rendered until it is opened. */
async function openSection(): Promise<void> {
  await userEvent.click(screen.getByRole('button', { name: /cable/i }));
}

afterEach(() => {
  vi.clearAllMocks();
});

describe('CableTestSettings TDR support', () => {
  it('reads support from the platform capability report, not a route that 404s', async () => {
    renderWithCapability({
      capability: 'cable_diagnostics',
      title: 'Cable diagnostics (TDR)',
      level: 'full',
    });

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledWith('/api/v1/status');
    });
    expect(mockGet).not.toHaveBeenCalledWith('/api/v1/telemetry/cable/support');
  });

  it("surfaces the platform's own note when TDR is unsupported", async () => {
    renderWithCapability({
      capability: 'cable_diagnostics',
      title: 'Cable diagnostics (TDR)',
      level: 'none',
      note: 'No macOS API exposes TDR.',
    });
    await openSection();

    // The note is the operator-facing half: "unsupported" alone does not say
    // why, and the old 404 path could only ever say "Unable to check support".
    expect(await screen.findByText('No macOS API exposes TDR.')).toBeInTheDocument();
  });
});
