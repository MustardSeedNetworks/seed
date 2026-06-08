/**
 * testRunStore.test.ts — unit tests for the run-all-tests orchestration store.
 *
 * The store replaces the former `window` CustomEvent bus (`runAllTests` /
 * `cardTestComplete` / `testsComplete`). These tests pin the state machine:
 * a run starts, declares the card-managed tests it must await, settles to
 * `idle` only when every awaited test reports, and settles to `partial`
 * (never a silent `idle`) when a backstop fires — the C2 correctness invariant.
 */

import { beforeEach, describe, expect, it } from 'vitest';
import { useTestRunStore } from './testRunStore';

const store = (): ReturnType<typeof useTestRunStore.getState> => useTestRunStore.getState();

describe('testRunStore', () => {
  beforeEach(() => {
    store().reset();
  });

  it('starts idle with no pending tests', () => {
    expect(store().status).toBe('idle');
    expect(store().pending).toEqual([]);
  });

  it('start() moves to running and bumps the start signal and run id', () => {
    const beforeSignal = store().startSignal;
    const beforeRun = store().runId;

    store().start();

    expect(store().status).toBe('running');
    expect(store().startSignal).toBe(beforeSignal + 1);
    expect(store().runId).toBe(beforeRun + 1);
  });

  it('awaitTests([]) settles a card-free run straight to idle', () => {
    store().start();
    store().awaitTests([]);

    expect(store().status).toBe('idle');
    expect(store().pending).toEqual([]);
  });

  it('awaitTests(tests) records the pending set while running', () => {
    store().start();
    store().awaitTests(['speedtest', 'iperf']);

    expect(store().status).toBe('running');
    expect(store().pending).toEqual(['speedtest', 'iperf']);
    expect(store().expected).toEqual(['speedtest', 'iperf']);
  });

  it('reportComplete() decrements pending and settles idle on the last one', () => {
    store().start();
    store().awaitTests(['speedtest', 'iperf']);

    store().reportComplete('speedtest');
    expect(store().status).toBe('running');
    expect(store().pending).toEqual(['iperf']);

    store().reportComplete('iperf');
    expect(store().status).toBe('idle');
    expect(store().pending).toEqual([]);
  });

  it('ignores reportComplete for a test that is not awaited', () => {
    store().start();
    store().awaitTests(['speedtest']);

    store().reportComplete('healthchecks');
    expect(store().pending).toEqual(['speedtest']);
    expect(store().status).toBe('running');
  });

  it('ignores reportComplete arriving before awaitTests (race with no listener)', () => {
    store().start();
    // A card finishes before the orchestrator declares what to await — mirrors
    // the prior behaviour where the cardTestComplete listener was attached only
    // after the fetch phase. The early report is dropped, not double-counted.
    store().reportComplete('speedtest');
    store().awaitTests(['speedtest']);

    expect(store().status).toBe('running');
    expect(store().pending).toEqual(['speedtest']);
  });

  it('settlePartial() flips a running run to partial for the matching run id', () => {
    store().start();
    const { runId } = store();
    store().awaitTests(['healthchecks']);

    store().settlePartial(runId);

    expect(store().status).toBe('partial');
    expect(store().pending).toEqual(['healthchecks']);
  });

  it('settlePartial() with a stale run id is a no-op (does not clobber a new run)', () => {
    store().start();
    const staleRunId = store().runId;
    // A fresh run supersedes the first; the first run's backstop must not fire.
    store().start();
    store().awaitTests(['speedtest']);

    store().settlePartial(staleRunId);

    expect(store().status).toBe('running');
  });

  it('settlePartial() does not override an already-completed run', () => {
    store().start();
    const { runId } = store();
    store().awaitTests([]); // completes → idle

    store().settlePartial(runId);

    expect(store().status).toBe('idle');
  });

  it('a fresh start() clears a prior partial outcome', () => {
    store().start();
    const { runId } = store();
    store().settlePartial(runId);
    expect(store().status).toBe('partial');

    store().start();
    expect(store().status).toBe('running');
  });
});
