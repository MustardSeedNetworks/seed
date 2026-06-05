import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { JobResponse } from '../types/generated/job-response';
import { useJobEvents } from './useJobEvents';

/**
 * Controllable EventSource fake: records listeners by event name and lets
 * the test emit frames and drive readyState. Replaces the coarse global
 * mock from test/setup.ts for this suite (per its "mock further per-suite"
 * note).
 */
class FakeEventSource {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;
  static instances: FakeEventSource[] = [];

  readonly CONNECTING = 0;
  readonly OPEN = 1;
  readonly CLOSED = 2;

  url: string;
  readyState = 0;
  closed = false;
  private listeners: Record<string, ((ev: unknown) => void)[]> = {};

  constructor(url: string) {
    this.url = url;
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, fn: (ev: unknown) => void): void {
    const existing = this.listeners[type] ?? [];
    existing.push(fn);
    this.listeners[type] = existing;
  }

  removeEventListener(): void {}

  close(): void {
    this.closed = true;
    this.readyState = FakeEventSource.CLOSED;
  }

  emit(type: string, ev: unknown): void {
    for (const fn of this.listeners[type] ?? []) {
      fn(ev);
    }
  }
}

const runningJob: JobResponse = {
  id: 'job-1',
  kind: 'engine-scan',
  state: 'running',
  progress: 0.5,
};

describe('useJobEvents', () => {
  let originalEventSource: typeof EventSource;

  beforeEach(() => {
    FakeEventSource.instances = [];
    originalEventSource = global.EventSource;
    global.EventSource = FakeEventSource as unknown as typeof EventSource;
  });

  afterEach(() => {
    global.EventSource = originalEventSource;
  });

  it('opens one stream at the jobs events path', () => {
    renderHook(() => useJobEvents(vi.fn()));

    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.instances[0].url).toContain('/api/v1/jobs/events');
  });

  it('invokes onJob with the parsed job for each `job` frame', () => {
    const onJob = vi.fn();
    renderHook(() => useJobEvents(onJob));

    FakeEventSource.instances[0].emit('job', { data: JSON.stringify(runningJob) });

    expect(onJob).toHaveBeenCalledWith(runningJob);
  });

  it('reports open status once the stream opens', () => {
    const { result } = renderHook(() => useJobEvents(vi.fn()));

    expect(result.current.status).toBe('connecting');
    act(() => FakeEventSource.instances[0].emit('open', {}));
    expect(result.current.status).toBe('open');
  });

  it('ignores a malformed frame without throwing or calling onJob', () => {
    const onJob = vi.fn();
    renderHook(() => useJobEvents(onJob));

    expect(() => FakeEventSource.instances[0].emit('job', { data: 'not json' })).not.toThrow();
    expect(onJob).not.toHaveBeenCalled();
  });

  it('closes the stream on unmount', () => {
    const { unmount } = renderHook(() => useJobEvents(vi.fn()));
    const source = FakeEventSource.instances[0];

    unmount();

    expect(source.closed).toBe(true);
  });

  it('does not re-open the stream when only the callback identity changes', () => {
    const { rerender } = renderHook(({ cb }) => useJobEvents(cb), {
      initialProps: { cb: vi.fn() },
    });

    rerender({ cb: vi.fn() });

    expect(FakeEventSource.instances).toHaveLength(1);
  });
});
