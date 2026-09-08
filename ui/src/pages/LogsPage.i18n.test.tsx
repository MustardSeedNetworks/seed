/**
 * LogsPage.i18n.test.tsx — the logs page renders real locale copy.
 *
 * S1-14b. The page is two cards and owns no copy, so the suite renders the
 * real cards through it. Writing it found seven English sentences hardcoded
 * in `SystemHealthCard` — the `Tip:` label and all six resource suggestions,
 * which are exactly what an operator reads when a resource crosses 75% — plus
 * the platform line's `Unknown` fallbacks.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../i18n';

vi.mock('../hooks/useLogs', () => ({
  useLogs: () => ({
    logs: [],
    allLogs: [],
    filters: {},
    setFilters: vi.fn(),
    resetFilters: vi.fn(),
    setIsStreaming: vi.fn(),
    fetchLogs: vi.fn(),
    fetchStats: vi.fn(),
    clearLogs: vi.fn(),
    addLog: vi.fn(),
    stats: { totalCount: 12, byLevel: { ERROR: 2, WARN: 1 } },
    isStreaming: true,
    isLoading: false,
    error: null,
  }),
}));

const health = {
  hostname: 'seed-lab',
  os: 'linux',
  arch: 'amd64',
  numCpu: 4,
  cpuPercent: 92,
  memoryPercent: 80,
  memoryUsed: 8 * 1024 ** 3,
  memoryTotal: 16 * 1024 ** 3,
  diskPercent: 12,
  diskUsed: 10 * 1024 ** 3,
  diskTotal: 100 * 1024 ** 3,
  uptime: 90_061,
  loadAvg1: 1.5,
  goroutines: 42,
  processMemory: 64 * 1024 ** 2,
};

const { LogsPage } = await import('./LogsPage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(<LogsPage />);
  await waitFor(() => expect(screen.getByText('seed-lab')).toBeVisible());
}

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ system: health }),
      }),
    ),
  );
});

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('LogsPage — real locale copy', () => {
  it('labels the system health card and its resources in English', async () => {
    await renderIn('en');

    expect(screen.getByText('System Health')).toBeVisible();
    expect(screen.getByText('CPU')).toBeVisible();
    expect(screen.getByText('Memory')).toBeVisible();
    expect(screen.getByText('Disk')).toBeVisible();
    expect(screen.getByText('Uptime')).toBeVisible();
    expect(screen.getByText('Load (1m)')).toBeVisible();
    expect(screen.getByText('linux/amd64 - 4 CPUs')).toBeVisible();
  });

  it('advises in English on the resource that is actually under pressure', async () => {
    await renderIn('en');

    // CPU is at 92%: the critical wording, not the merely-high one.
    expect(screen.getAllByText('Tip:').length).toBe(2);
    expect(
      screen.getByText('Check for runaway processes or consider upgrading CPU resources'),
    ).toBeVisible();
    // Memory is at 80%: over the 75% line but under critical.
    expect(
      screen.getByText(
        'Consider increasing system memory or closing memory-intensive applications',
      ),
    ).toBeVisible();
    // Disk is at 12%: no advice at all.
    expect(
      screen.queryByText('Clear temporary files, remove unused applications, or archive old data'),
    ).toBeNull();
  });

  it('labels the log viewer card in English', async () => {
    await renderIn('en');

    expect(screen.getByText('System Logs')).toBeVisible();
    expect(screen.getByText('Live')).toBeVisible();
    expect(screen.getByText('logs')).toBeVisible();
    expect(screen.getByText('errors')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Full Screen' })).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    for (const english of [
      'System Health',
      'Memory',
      'Disk',
      'Uptime',
      'Load (1m)',
      'Tip:',
      'Check for runaway processes or consider upgrading CPU resources',
      'Consider increasing system memory or closing memory-intensive applications',
      'System Logs',
      'Live',
      'errors',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }

    expect(screen.getByText('Salud del sistema')).toBeVisible();
    expect(screen.getByText('Memoria')).toBeVisible();
    expect(screen.getByText('Disco')).toBeVisible();
    expect(screen.getAllByText('Consejo:').length).toBe(2);
    expect(
      screen.getByText('Busque procesos desbocados o considere ampliar los recursos de CPU'),
    ).toBeVisible();
    expect(screen.getByText('linux/amd64 - 4 CPU')).toBeVisible();
    expect(screen.getByText('Registros del sistema')).toBeVisible();
    expect(screen.getByText('En vivo')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Pantalla Completa' })).toBeVisible();
  });

  it('leaves the kernel’s own words and CPU alone under es', async () => {
    await renderIn('es');

    // `linux`, `amd64` and the hostname come off the wire; CPU is the unit.
    expect(screen.getByText('seed-lab')).toBeVisible();
    expect(screen.getByText('CPU')).toBeVisible();
    expect(screen.getByText(/linux\/amd64/)).toBeVisible();
  });
});
