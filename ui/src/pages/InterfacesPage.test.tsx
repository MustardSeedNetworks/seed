/**
 * InterfacesPage (UI-SEED-21): loading, empty and no-data-yet are three
 * distinct states, and the list sorts by any rate with unrated interfaces
 * always last.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { UseInterfaceStatsResult } from '../hooks/useInterfaceStats';
import i18n from '../i18n';
import type { InterfaceStatsResponse } from '../types/generated/interface-stats-list-response';

let nextIfIndex = 1;

function iface(
  name: string,
  errors: number | null,
  extra: Partial<InterfaceStatsResponse> = {},
): InterfaceStatsResponse {
  return {
    targetId: 't1',
    targetName: 'core-sw',
    ifIndex: nextIfIndex++,
    name,
    operStatus: 'up',
    speedBps: 1_000_000_000,
    ...(errors === null
      ? {}
      : {
          rates: {
            sampledAt: '2026-10-07T12:00:00Z',
            inOctetsPerSec: 1_250_000,
            outOctetsPerSec: 125_000,
            inUtilizationPct: 1,
            outUtilizationPct: 0.1,
            inErrorsPerSec: errors,
            outErrorsPerSec: 0,
            inDiscardsPerSec: 0,
            outDiscardsPerSec: 0,
          },
        }),
    ...extra,
  };
}

const state: { current: UseInterfaceStatsResult } = {
  current: { interfaces: [], loading: false, error: null },
};

vi.mock('../hooks/useInterfaceStats', () => ({
  useInterfaceStats: (): UseInterfaceStatsResult => state.current,
}));

const { InterfacesPage } = await import('./InterfacesPage');

function names(): string[] {
  return screen
    .getAllByTestId('interface-row')
    .map((row) => within(row).getByTestId('interface-name').textContent ?? '');
}

beforeEach(async () => {
  await i18n.changeLanguage('en');
});

afterEach(() => {
  state.current = { interfaces: [], loading: false, error: null };
});

describe('InterfacesPage states', () => {
  it('shows loading until the list arrives', () => {
    state.current = { interfaces: [], loading: true, error: null };
    render(<InterfacesPage />);
    expect(screen.getByTestId('interfaces-loading')).toBeInTheDocument();
    expect(screen.queryByTestId('interfaces-empty')).not.toBeInTheDocument();
  });

  it('points an empty estate at polling targets', () => {
    render(<InterfacesPage />);
    const empty = screen.getByTestId('interfaces-empty');
    expect(within(empty).getByRole('link', { name: 'Add a polling target' })).toHaveAttribute(
      'href',
      '/polling-targets',
    );
    expect(screen.queryByTestId('interfaces-table')).not.toBeInTheDocument();
  });

  it('lists interfaces with no rate yet and says why', () => {
    state.current = { interfaces: [iface('Gi0/1', null)], loading: false, error: null };
    render(<InterfacesPage />);
    expect(screen.getByTestId('interfaces-no-data')).toBeInTheDocument();
    expect(names()).toEqual(['Gi0/1']);
    expect(screen.getByText('No rate yet')).toBeInTheDocument();
  });

  it('drops the no-data notice once any interface is rated', () => {
    state.current = {
      interfaces: [iface('Gi0/1', null), iface('Gi0/2', 0)],
      loading: false,
      error: null,
    };
    render(<InterfacesPage />);
    expect(screen.queryByTestId('interfaces-no-data')).not.toBeInTheDocument();
    expect(screen.getByText('10 Mbps / 1 Mbps')).toBeInTheDocument();
  });

  it('shows a read failure', () => {
    state.current = {
      interfaces: [],
      loading: false,
      error: 'Failed to read interface statistics',
    };
    render(<InterfacesPage />);
    expect(screen.getByTestId('interfaces-error')).toHaveTextContent(
      'Failed to read interface statistics',
    );
  });

  it('spells out a status other than up', () => {
    state.current = {
      interfaces: [iface('Gi0/9', 0, { operStatus: 'lowerLayerDown' })],
      loading: false,
      error: null,
    };
    render(<InterfacesPage />);
    expect(screen.getByText('Lower layer down')).toBeVisible();
  });
});

describe('InterfacesPage sorting', () => {
  beforeEach(() => {
    state.current = {
      interfaces: [
        iface('ge-1', 0.5),
        iface('ge-22', null),
        iface('ge-333', 4),
        iface('ge-4444', 0),
      ],
      loading: false,
      error: null,
    };
  });

  it('starts on errors, highest first, with unrated interfaces last', () => {
    render(<InterfacesPage />);
    expect(names()).toEqual(['ge-333', 'ge-1', 'ge-4444', 'ge-22']);
    expect(screen.getByTestId('interfaces-sort-errors').closest('th')).toHaveAttribute(
      'aria-sort',
      'descending',
    );
  });

  it('reverses on a second click and keeps unrated interfaces last', async () => {
    render(<InterfacesPage />);
    await userEvent.click(screen.getByTestId('interfaces-sort-errors'));
    expect(names()).toEqual(['ge-4444', 'ge-1', 'ge-333', 'ge-22']);
    expect(screen.getByTestId('interfaces-sort-errors').closest('th')).toHaveAttribute(
      'aria-sort',
      'ascending',
    );
  });

  it('sorts by interface name ascending, numbers in order', async () => {
    state.current = {
      interfaces: [iface('Gi0/10', 0), iface('Gi0/2', 0), iface('Gi0/1', null)],
      loading: false,
      error: null,
    };
    render(<InterfacesPage />);
    await userEvent.click(screen.getByTestId('interfaces-sort-interface'));
    expect(names()).toEqual(['Gi0/1', 'Gi0/2', 'Gi0/10']);
    expect(screen.getByTestId('interfaces-sort-errors').closest('th')).toHaveAttribute(
      'aria-sort',
      'none',
    );
  });
});

describe('InterfacesPage locale', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('renders Spanish copy', async () => {
    await i18n.changeLanguage('es');
    state.current = { interfaces: [iface('Gi0/1', null)], loading: false, error: null };
    render(<InterfacesPage />);
    expect(screen.getByRole('button', { name: 'Errores' })).toBeInTheDocument();
    expect(screen.getByText('Sin tasa todavía')).toBeInTheDocument();
  });
});
