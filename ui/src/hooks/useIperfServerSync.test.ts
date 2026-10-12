/**
 * useIperfServerSync reports the listener's state from the start/stop answer.
 *
 * POST /api/v1/telemetry/iperf/server answers the listener's resulting
 * ServerStatus, so the hook hands that to onStatus instead of reading
 * /server/status a second time.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { ServerStatus } from '../types/generated/iperf-server-status';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();

vi.mock('../api', () => ({
  api: {
    get: (p: string): Promise<unknown> => mockGet(p),
    post: (p: string, b?: unknown): Promise<unknown> => mockPost(p, b),
  },
}));
vi.mock('../contexts/RoleContext', () => ({ useRole: () => ({ canWrite: true }) }));

const { useIperfServerSync } = await import('./useIperfServerSync');

describe('useIperfServerSync', () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it('reports the start answer without reading the status again', async () => {
    const started: ServerStatus = { running: true, port: 5202, pid: 4242 };
    mockGet.mockResolvedValue({ running: false, port: 0, pid: 0 });
    mockPost.mockResolvedValue(started);
    const onStatus = vi.fn<(status: ServerStatus) => void>();

    renderHook(() =>
      useIperfServerSync({ installed: true, enableServer: true, serverPort: 5202, onStatus }),
    );

    await waitFor(() => {
      expect(onStatus).toHaveBeenCalledWith(started);
    });
    expect(mockPost).toHaveBeenCalledWith('/api/v1/telemetry/iperf/server', {
      action: 'start',
      port: 5202,
    });
    expect(mockGet).toHaveBeenCalledTimes(1);
  });
});
