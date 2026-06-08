/**
 * Test-run orchestration store (Zustand).
 *
 * Replaces the former `window` CustomEvent bus that coordinated the
 * "Run All Tests" flow (`runAllTests` → `cardTestComplete` → `testsComplete`).
 * The event bus was global, untyped, and untestable, and its completion
 * accounting raced the listener lifecycle — the root cause behind the C2
 * false-`testsComplete` defect (seed#1568). This store makes the run a small,
 * typed, unit-testable state machine.
 *
 * Flow:
 *   1. A trigger (FAB click, link-up auto-run) calls `start()`. This bumps
 *      `startSignal` (subscribers react and kick off their work) and `runId`
 *      (scopes async backstops to a single run).
 *   2. The orchestrator (app shell) declares which card-managed tests the run
 *      must await via `awaitTests(...)`. An empty set completes the run at once.
 *   3. Each card calls `reportComplete(test)` when its poll-driven test settles.
 *      The last expected report flips the run back to `idle`.
 *   4. A backstop timeout calls `settlePartial(runId)`; a run that did not see
 *      every report is surfaced as `partial`, never as a silent `idle` (the C2
 *      correctness invariant).
 *
 * Subscribe to the start signal with {@link useTestRunSignal}; read `status`
 * for UI (e.g. the FAB) via the hook directly.
 *
 * Related: seed#1568 (C2), SEED_UI_ARCH_PLAN.md A2/H2.
 */

import { useEffect, useRef } from 'react';
import { create } from 'zustand';
import { devtools, subscribeWithSelector } from 'zustand/middleware';

export type TestRunStatus = 'idle' | 'running' | 'partial';

interface TestRunState {
  /** Current run lifecycle state. `partial` persists until the next `start()`. */
  status: TestRunStatus;
  /** Monotonic counter bumped on every `start()`; subscribers react to changes. */
  startSignal: number;
  /** Monotonic run identifier; backstops capture it to ignore stale timeouts. */
  runId: number;
  /** Card-managed tests still awaited for the active run. */
  pending: string[];
  /** Full set of card-managed tests for the active run (for "X of Y" messaging). */
  expected: string[];
  /** True once `awaitTests` has declared the set — guards early `reportComplete`. */
  awaiting: boolean;
}

interface TestRunActions {
  /** Begin a run: clear prior outcome, bump the start signal and run id. Returns the new run id. */
  start: () => number;
  /** Declare the card-managed tests to await; an empty set completes at once. */
  awaitTests: (tests: string[]) => void;
  /** Record a card-managed test as finished; the last one settles the run. */
  reportComplete: (test: string) => void;
  /** Backstop: mark the still-running run (matched by id) as partial. */
  settlePartial: (runId: number) => void;
  /** Reset to the initial idle state (used by tests and on teardown). */
  reset: () => void;
}

const initialState: TestRunState = {
  status: 'idle',
  startSignal: 0,
  runId: 0,
  pending: [],
  expected: [],
  awaiting: false,
};

// NB: the store type is inferred (not annotated) so the `subscribeWithSelector`
// augmentation to `.subscribe(selector, listener)` survives — an explicit
// `UseBoundStore<StoreApi<…>>` annotation would erase that overload, which
// `useTestRunSignal` below depends on.
export const useTestRunStore = create<TestRunState & TestRunActions>()(
  devtools(
    subscribeWithSelector((set, get) => ({
      ...initialState,

      start: (): number => {
        set(
          (s) => ({
            status: 'running',
            startSignal: s.startSignal + 1,
            runId: s.runId + 1,
            pending: [],
            expected: [],
            awaiting: false,
          }),
          false,
          'testRun/start',
        );
        return get().runId;
      },

      awaitTests: (tests: string[]) =>
        set(
          (s) => {
            if (s.status !== 'running') {
              return {};
            }
            if (tests.length === 0) {
              // Nothing to wait for — a genuine, complete run.
              return { status: 'idle', pending: [], expected: [], awaiting: false };
            }
            return { expected: tests, pending: tests, awaiting: true };
          },
          false,
          'testRun/awaitTests',
        ),

      reportComplete: (test: string) =>
        set(
          (s) => {
            if (s.status !== 'running' || !s.awaiting || !s.pending.includes(test)) {
              return {};
            }
            const pending = s.pending.filter((t) => t !== test);
            if (pending.length === 0) {
              // Every awaited card reported — a genuine, complete run.
              return { status: 'idle', pending: [], awaiting: false };
            }
            return { pending };
          },
          false,
          'testRun/reportComplete',
        ),

      settlePartial: (runId: number) =>
        set(
          (s) => {
            if (s.status !== 'running' || s.runId !== runId) {
              return {};
            }
            return { status: 'partial', awaiting: false };
          },
          false,
          'testRun/settlePartial',
        ),

      reset: () => set({ ...initialState }, false, 'testRun/reset'),
    })),
    { name: 'testRunStore' },
  ),
);

/**
 * Subscribe to run starts. `handler` is invoked once per `start()` with the
 * run's id, using the latest closure each render — replicating the
 * fire-once-per-event semantics of the former `window` `runAllTests` listener
 * without re-firing when unrelated dependencies change.
 */
export function useTestRunSignal(handler: (runId: number) => void): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(
    () =>
      useTestRunStore.subscribe(
        (s) => s.startSignal,
        (signal, previous) => {
          if (signal !== previous) {
            handlerRef.current(useTestRunStore.getState().runId);
          }
        },
      ),
    [],
  );
}
