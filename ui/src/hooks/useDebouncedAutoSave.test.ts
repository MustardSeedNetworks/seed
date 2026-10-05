/**
 * useDebouncedAutoSave — an edit saves, a load never does, and a caller who
 * cannot write never saves (#2467).
 *
 * The drawer used to tell a load from an edit with a 500 ms window after
 * opening: an edit inside it was dropped, and a load after it was saved
 * (#2994). These cases pin the edit-driven contract with no window at all.
 */

import { act, renderHook } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useDebouncedAutoSave } from './useDebouncedAutoSave';

beforeEach(() => {
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});

interface Harness {
  edit: (value: number) => void;
  load: (value: number) => void;
  setCanWrite: (canWrite: boolean) => void;
  saved: number[];
}

function arm(canWrite: boolean): Harness {
  const saved: number[] = [];
  const view = renderHook(
    (enabled: boolean) => {
      const [value, setValue] = useState(0);
      const edit = useDebouncedAutoSave(setValue, () => void saved.push(value), enabled);
      return { edit, setValue };
    },
    { initialProps: canWrite },
  );

  return {
    edit: (value): void => act(() => view.result.current.edit(value)),
    load: (value): void => act(() => view.result.current.setValue(value)),
    setCanWrite: (enabled): void => view.rerender(enabled),
    saved,
  };
}

describe('useDebouncedAutoSave', () => {
  it('saves an edit made the moment the drawer opens', () => {
    const h = arm(true);

    h.edit(3);
    act(() => vi.advanceTimersByTime(800));

    expect(h.saved).toEqual([3]);
  });

  it('never saves a loaded value, however late it lands', () => {
    const h = arm(true);

    act(() => vi.advanceTimersByTime(5000));
    h.load(4);
    act(() => vi.advanceTimersByTime(5000));

    expect(h.saved).toEqual([]);
  });

  it('saves the edited value when a load lands during the debounce', () => {
    const h = arm(true);

    h.edit(3);
    h.load(4);
    act(() => vi.advanceTimersByTime(800));

    expect(h.saved).toEqual([3]);
  });

  it('coalesces edits inside the debounce into one save of the last', () => {
    const h = arm(true);

    h.edit(1);
    act(() => vi.advanceTimersByTime(500));
    h.edit(2);
    act(() => vi.advanceTimersByTime(799));
    expect(h.saved).toEqual([]);
    act(() => vi.advanceTimersByTime(1));

    expect(h.saved).toEqual([2]);
  });

  it('never saves when the caller cannot write', () => {
    const h = arm(false);

    h.edit(3);
    act(() => vi.advanceTimersByTime(5000));

    expect(h.saved).toEqual([]);
  });

  it('does not save on the role alone becoming writable', () => {
    // RoleProvider clears the user whenever /users/me fails and sets it again
    // on the next success, so canWrite flips false -> true with nothing the
    // user did. That must not schedule a save of values nobody edited.
    const h = arm(false);

    h.load(4);
    h.setCanWrite(true);
    act(() => vi.advanceTimersByTime(5000));

    expect(h.saved).toEqual([]);
  });
});
