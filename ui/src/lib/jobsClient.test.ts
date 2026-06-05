import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { CreateJobRequest } from '../types/generated/create-job-request';
import type { JobResponse } from '../types/generated/job-response';
import { cancelJob, getJob, isTerminalJobState, submitJob } from './jobsClient';

// Mock the shared API client so we assert the exact endpoint/method/headers
// the jobs client uses, without exercising fetch/CSRF plumbing here.
vi.mock('../api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

import { api } from '../api/client';

const sampleJob: JobResponse = {
  id: 'job-123',
  kind: 'engine-scan',
  state: 'queued',
  progress: 0,
};

describe('jobsClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('submitJob', () => {
    it('POSTs to /api/v1/jobs with the request body', async () => {
      vi.mocked(api.post).mockResolvedValueOnce(sampleJob);
      const req: CreateJobRequest = { kind: 'engine-scan' };

      const result = await submitJob(req);

      expect(api.post).toHaveBeenCalledWith('/api/v1/jobs', req, undefined);
      expect(result).toEqual(sampleJob);
    });

    it('passes an Idempotency-Key header when given', async () => {
      vi.mocked(api.post).mockResolvedValueOnce(sampleJob);
      const req: CreateJobRequest = { kind: 'speedtest' };

      await submitJob(req, 'key-abc');

      expect(api.post).toHaveBeenCalledWith('/api/v1/jobs', req, {
        headers: { 'Idempotency-Key': 'key-abc' },
      });
    });
  });

  describe('getJob', () => {
    it('GETs the id-scoped endpoint, url-encoding the id', async () => {
      vi.mocked(api.get).mockResolvedValueOnce(sampleJob);

      await getJob('a/b id');

      expect(api.get).toHaveBeenCalledWith('/api/v1/jobs/a%2Fb%20id');
    });
  });

  describe('cancelJob', () => {
    it('DELETEs the id-scoped endpoint', async () => {
      vi.mocked(api.delete).mockResolvedValueOnce({ ...sampleJob, state: 'cancelled' });

      const result = await cancelJob('job-123');

      expect(api.delete).toHaveBeenCalledWith('/api/v1/jobs/job-123');
      expect(result.state).toBe('cancelled');
    });
  });

  describe('isTerminalJobState', () => {
    it('is true for terminal states', () => {
      expect(isTerminalJobState('succeeded')).toBe(true);
      expect(isTerminalJobState('failed')).toBe(true);
      expect(isTerminalJobState('cancelled')).toBe(true);
    });

    it('is false for in-flight states', () => {
      expect(isTerminalJobState('queued')).toBe(false);
      expect(isTerminalJobState('running')).toBe(false);
      expect(isTerminalJobState('')).toBe(false);
    });
  });
});
