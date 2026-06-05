import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { useEnginePhase } from './useEnginePhase';

// Controllable EventSource fake (same shape as the useJobEvents suite).
class FakeEventSource {
  static instances: FakeEventSource[] = [];
  url: string;
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
  }

  emit(type: string, ev: unknown): void {
    for (const fn of this.listeners[type] ?? []) {
      fn(ev);
    }
  }
}

function progressFrame(phase: string, fraction: number): { data: string } {
  return { data: JSON.stringify({ type: 'scan.progress', payload: { phase, fraction } }) };
}

describe('useEnginePhase', () => {
  let original: typeof EventSource;

  beforeEach(() => {
    FakeEventSource.instances = [];
    original = global.EventSource;
    global.EventSource = FakeEventSource as unknown as typeof EventSource;
  });

  afterEach(() => {
    global.EventSource = original;
  });

  it('opens the engine events stream', () => {
    renderHook(() => useEnginePhase());
    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.instances[0].url).toContain('/api/v1/discovery/engine/events');
  });

  it('tracks the latest scan.progress phase', () => {
    const { result } = renderHook(() => useEnginePhase());
    expect(result.current.phase).toBe('');

    act(() => FakeEventSource.instances[0].emit('scan.progress', progressFrame('discovery', 0.2)));
    expect(result.current.phase).toBe('discovery');

    act(() => FakeEventSource.instances[0].emit('scan.progress', progressFrame('enrichment', 0.8)));
    expect(result.current.phase).toBe('enrichment');
  });

  it('resets the phase on scan.started', () => {
    const { result } = renderHook(() => useEnginePhase());
    act(() => FakeEventSource.instances[0].emit('scan.progress', progressFrame('assessment', 1)));
    expect(result.current.phase).toBe('assessment');

    act(() => FakeEventSource.instances[0].emit('scan.started', { data: '{}' }));
    expect(result.current.phase).toBe('');
  });

  it('ignores a malformed frame without throwing', () => {
    const { result } = renderHook(() => useEnginePhase());
    expect(() =>
      act(() => FakeEventSource.instances[0].emit('scan.progress', { data: 'not json' })),
    ).not.toThrow();
    expect(result.current.phase).toBe('');
  });

  it('closes the stream on unmount', () => {
    const { unmount } = renderHook(() => useEnginePhase());
    const source = FakeEventSource.instances[0];
    unmount();
    expect(source.closed).toBe(true);
  });
});
