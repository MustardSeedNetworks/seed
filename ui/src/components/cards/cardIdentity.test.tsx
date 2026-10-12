/**
 * Effects kept stable by the React Compiler rather than by useCallback
 * (UI-SEED-41 slice 7, #3066).
 *
 * These cards pass a plain function to an effect's dependency list. Compiled,
 * the function keeps its identity until what it reads changes, so the effect
 * runs once. Each case fails when the compiler is removed from the vitest
 * config: the function is new on every render, so the card refetches or
 * re-subscribes each time it re-renders.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import { createMockResponse, mockFetch } from '../../test/setup';
import type { SystemHealthResponse } from '../../types/generated/system-health-response';
import { PathDiscoveryCard } from './PathDiscoveryCard';
import { SLADashboardCard } from './SlaDashboardCard';
import { SystemHealthCard } from './SystemHealthCard';

describe('compiled cards keep their effects stable across re-renders', () => {
  it('SLADashboardCard fetches once, not again when loading settles', async () => {
    mockFetch.mockReset();
    mockFetch.mockImplementation(() => createMockResponse({ activeCount: 2 }));
    const { rerender } = render(<SLADashboardCard />);
    expect(await screen.findByText('2')).toBeInTheDocument();
    rerender(<SLADashboardCard />);
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it('SystemHealthCard fetches once, not again when its data arrives', async () => {
    mockFetch.mockReset();
    const health: SystemHealthResponse = {
      system: {
        cpuPercent: 10,
        memoryPercent: 20,
        memoryUsed: 2e9,
        memoryTotal: 1e10,
        diskPercent: 30,
        diskUsed: 3e10,
        diskTotal: 1e11,
        uptime: 3600,
        loadAvg1: 0.5,
        loadAvg5: 0.4,
        loadAvg15: 0.3,
        goroutines: 40,
        processMemory: 5e7,
        hostname: 'probe-1',
        os: 'linux',
        arch: 'amd64',
        numCpu: 4,
      },
      application: { version: '0.0.0', uptime: 60 },
      services: { discoveryService: true, linkMonitor: true, sseHub: true },
    };
    mockFetch.mockImplementation(() => createMockResponse(health));
    const { rerender } = render(<SystemHealthCard />);
    expect(await screen.findByText('probe-1')).toBeInTheDocument();
    rerender(<SystemHealthCard />);
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it('PathDiscoveryCard registers its trace handler once', async () => {
    const unregister = vi.fn();
    const register = vi.fn(() => unregister);
    const { rerender } = render(<PathDiscoveryCard onRegisterTraceHandler={register} />);
    rerender(<PathDiscoveryCard onRegisterTraceHandler={register} />);
    await waitFor(() => {
      expect(screen.getByText(i18n.t('cards:pathDiscovery.title'))).toBeInTheDocument();
    });
    expect(register).toHaveBeenCalledTimes(1);
    expect(unregister).not.toHaveBeenCalled();
  });
});
