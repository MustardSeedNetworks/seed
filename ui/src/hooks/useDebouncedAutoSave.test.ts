/**
 * useDebouncedAutoSave — the auto-save must not fire for a caller who cannot
 * write (#2467).
 *
 * The drawer arms eight of these. Before the read-only Settings work they were
 * unconditional, so a viewer whose drawer merely normalised a loaded value
 * issued the operator-gated PUT behind it and got a 403 no click had asked for
 * — the same mount-time write shape #2464 fixed in PerformanceCard.
 */

import { renderHook } from '@testing-library/react';
import { useRef } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useDebouncedAutoSave } from './useDebouncedAutoSave';

beforeEach(() => {
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});

function arm(enabled: boolean, saveFn: () => void): { rerender: (props: boolean) => void } {
  const view = renderHook(
    (canWrite: boolean) => {
      const isInit = useRef(false);
      const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

      useDebouncedAutoSave(saveFn, isInit, timer, canWrite);
    },
    { initialProps: enabled },
  );

  return { rerender: view.rerender };
}

describe('useDebouncedAutoSave', () => {
  it('never saves when the caller cannot write', () => {
    const saveFn = vi.fn();
    arm(false, saveFn);

    vi.advanceTimersByTime(5000);

    expect(saveFn).not.toHaveBeenCalled();
  });

  it('does not save on the role alone becoming writable', () => {
    // RoleProvider clears the user whenever /users/me fails and sets it again
    // on the next success, so canWrite flips false -> true with nothing the
    // user did. That must not schedule a save of values nobody edited.
    const saveFn = vi.fn();
    const view = arm(false, saveFn);

    view.rerender(true);
    vi.advanceTimersByTime(5000);

    expect(saveFn).not.toHaveBeenCalled();
  });

  it('saves once the debounce elapses when the caller can write', () => {
    const saveFn = vi.fn();
    arm(true, saveFn);

    vi.advanceTimersByTime(5000);

    expect(saveFn).toHaveBeenCalledTimes(1);
  });
});
