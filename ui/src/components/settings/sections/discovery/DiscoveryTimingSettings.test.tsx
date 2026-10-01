/**
 * DiscoveryTimingSettings (seed#491).
 *
 * The drawer used to offer eight discovery timers in raw milliseconds, and the
 * daemon read two of them. What is left is those two, in the unit an operator
 * thinks in, with the backend default quoted as the recommendation.
 */

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type React from 'react';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { clearDefaultsCache } from '../../../../hooks/useDefaults';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DEFAULT_NETWORK_DISCOVERY_SETTINGS } from '../../../../types/settings';
import { DiscoveryTimingSettings } from './DiscoveryTimingSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();

vi.mock('../../../../api', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

function serveDefaults(rescanIntervalMs: number, scanTimeoutMs: number): void {
  mockGet.mockImplementation((path: string) =>
    path === '/api/v1/settings/defaults'
      ? Promise.resolve({ networkDiscovery: { scanTimeoutMs, timing: { rescanIntervalMs } } })
      : Promise.reject(new Error(`unexpected request to ${path}`)),
  );
}

function Harness({
  initial,
  onChange,
}: {
  initial: NetworkDiscoverySettings;
  onChange: (next: NetworkDiscoverySettings) => void;
}): React.JSX.Element {
  const [settings, setSettings] = useState(initial);
  return (
    <DiscoveryTimingSettings
      settings={settings}
      onSettingsChange={(update): void =>
        setSettings((prev) => {
          const next = typeof update === 'function' ? update(prev) : update;
          onChange(next);
          return next;
        })
      }
    />
  );
}

function renderTiming(rescanIntervalMs: number, scanTimeoutMs: number) {
  const onChange = vi.fn<(next: NetworkDiscoverySettings) => void>();
  render(
    <Harness
      initial={{
        ...DEFAULT_NETWORK_DISCOVERY_SETTINGS,
        scanTimeoutMs,
        timing: { rescanIntervalMs },
      }}
      onChange={onChange}
    />,
  );
  return {
    onChange,
    lastSaved: (): NetworkDiscoverySettings | undefined => onChange.mock.lastCall?.[0],
  };
}

async function openAdvanced(): Promise<void> {
  await userEvent.click(screen.getByRole('button', { name: 'Advanced' }));
}

afterEach(() => {
  clearDefaultsCache();
  vi.clearAllMocks();
});

describe('DiscoveryTimingSettings', () => {
  it('shows only the two timers the daemon reads, in minutes and seconds', async () => {
    serveDefaults(60_000, 30_000);
    renderTiming(300_000, 45_000);

    expect(screen.getByLabelText('Rescan interval (minutes)')).toHaveValue(5);
    expect(screen.queryByLabelText('Scan time limit (seconds)')).toBeNull();

    await openAdvanced();
    expect(screen.getByLabelText('Scan time limit (seconds)')).toHaveValue(45);
    expect(screen.getAllByRole('spinbutton')).toHaveLength(2);
  });

  it('quotes the backend default as the recommendation', async () => {
    serveDefaults(120_000, 20_000);
    renderTiming(60_000, 30_000);

    expect(await screen.findByText(/Recommended: 2 minutes\./)).toBeInTheDocument();
    await openAdvanced();
    expect(screen.getByText(/Recommended: 20 seconds\./)).toBeInTheDocument();
  });

  it('saves a typed value in milliseconds', async () => {
    serveDefaults(60_000, 30_000);
    const { lastSaved } = renderTiming(60_000, 30_000);

    const rescan = screen.getByLabelText('Rescan interval (minutes)');
    await userEvent.clear(rescan);
    await userEvent.type(rescan, '15');
    expect(lastSaved()?.timing.rescanIntervalMs).toBe(900_000);

    await openAdvanced();
    const limit = screen.getByLabelText('Scan time limit (seconds)');
    await userEvent.clear(limit);
    await userEvent.type(limit, '90');
    expect(lastSaved()?.scanTimeoutMs).toBe(90_000);
    expect(lastSaved()?.timing.rescanIntervalMs).toBe(900_000);
  });

  it.each([
    ['empty', ''],
    ['zero', '0'],
    ['above the maximum', '61'],
    ['a fraction', '1.5'],
  ])('keeps the stored value when the input is %s, and restores it on blur', async (_, typed) => {
    serveDefaults(60_000, 30_000);
    const { onChange } = renderTiming(300_000, 30_000);

    const rescan = screen.getByLabelText('Rescan interval (minutes)');
    fireEvent.change(rescan, { target: { value: typed } });
    expect(onChange).not.toHaveBeenCalled();

    fireEvent.blur(rescan);
    expect(rescan).toHaveValue(5);
  });

  it('leaves a value that is not a whole unit untouched until it is edited', async () => {
    serveDefaults(60_000, 30_000);
    const { onChange } = renderTiming(90_000, 7_500);

    expect(screen.getByLabelText('Rescan interval (minutes)')).toHaveValue(1.5);
    await openAdvanced();
    expect(screen.getByLabelText('Scan time limit (seconds)')).toHaveValue(7.5);
    expect(onChange).not.toHaveBeenCalled();
  });

  it('reaches both fields and opens Advanced from the keyboard', async () => {
    serveDefaults(60_000, 30_000);
    renderTiming(60_000, 30_000);

    await userEvent.tab();
    expect(screen.getByLabelText('Rescan interval (minutes)')).toHaveFocus();
    await userEvent.tab();
    const advanced = screen.getByRole('button', { name: 'Advanced' });
    expect(advanced).toHaveFocus();
    expect(advanced).toHaveAttribute('aria-expanded', 'false');
    await userEvent.keyboard('{Enter}');
    expect(advanced).toHaveAttribute('aria-expanded', 'true');
    await userEvent.tab();
    expect(screen.getByLabelText('Scan time limit (seconds)')).toHaveFocus();
  });
});
