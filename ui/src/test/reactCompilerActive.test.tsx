/**
 * reactCompilerActive.test.tsx — proves the React Compiler runs in the test
 * environment, not just in the production build.
 *
 * vite.config.ts and vitest.config.ts declare their plugins separately, so it
 * is entirely possible to ship compiled components while testing un-compiled
 * ones. That gap is invisible: every test still passes, it just stops testing
 * what ships — a memo the compiler subsumes looks required, and a compiler
 * regression could never fail a test.
 *
 * The probe writes no memo of its own. `onPick` closes over state that never
 * changes, so a compiled build holds its identity across re-renders and the
 * counter stays at zero; un-compiled, it is a fresh closure every render and
 * the counter climbs. The count is therefore a direct read on whether the
 * compiler is in the pipeline.
 *
 * The comparison happens inside an effect against a ref rather than through a
 * dependency array, because a dependency array is exactly what a static lint
 * rule objects to here — it can see that `onPick` has no memo, and it cannot
 * see the compiler that gives it one.
 */
import { fireEvent, render, screen } from '@testing-library/react';
import { type ReactElement, useEffect, useRef, useState } from 'react';
import { describe, expect, it } from 'vitest';

function Probe({ onIdentityChange }: { onIdentityChange: () => void }): ReactElement {
  const [tick, setTick] = useState(0);
  const [items] = useState(['a', 'b']);

  const onPick = () => {
    void items;
  };

  const previous = useRef<(() => void) | null>(null);
  useEffect(() => {
    if (previous.current !== null && previous.current !== onPick) {
      onIdentityChange();
    }
    previous.current = onPick;
  });

  return (
    <button type="button" onClick={() => setTick((n) => n + 1)}>
      tick {tick}
    </button>
  );
}

describe('React Compiler', () => {
  it('is active in the test environment, so tests exercise what ships', () => {
    let identityChanges = 0;
    render(
      <Probe
        onIdentityChange={() => {
          identityChanges += 1;
        }}
      />,
    );

    fireEvent.click(screen.getByRole('button'));
    fireEvent.click(screen.getByRole('button'));

    expect(
      identityChanges,
      'a callback with no hand-written memo churned identity across re-renders — the React ' +
        'Compiler is not running in vitest, so the suite exercises un-compiled components ' +
        'while the bundle ships compiled ones',
    ).toBe(0);
  });
});
