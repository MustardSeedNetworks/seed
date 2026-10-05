/**
 * Effects kept stable by the React Compiler rather than by useCallback
 * (UI-SEED-41 slice 8, #3066).
 *
 * These sections pass a plain function to an effect's dependency list.
 * Compiled, the function keeps its identity until what it reads changes, so
 * the mount fetch runs once. Each case fails when the compiler is removed from
 * the vitest config: the function is new on every render, so the section
 * refetches each time it re-renders.
 */

import { render, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../../contexts/RoleContext';
import { createMockResponse, mockFetch } from '../../test/setup';
import type { CableTestSettings as CableTestSettingsType } from '../../types/settings';
import { CableTestSettings } from './sections/CableTestSettings';
import { ConfigBackupsSection } from './sections/ConfigBackupsSection';

const cableSettings: CableTestSettingsType = { enabled: true };

describe('compiled settings sections keep their effects stable across re-renders', () => {
  it('CableTestSettings checks TDR support once', async () => {
    mockFetch.mockReset();
    mockFetch.mockImplementation(() => createMockResponse({ capabilities: [] }));
    const setCableTestSettings = vi.fn();
    const section = (
      <RoleProvider isAuthenticated={false}>
        <CableTestSettings
          cableTestSettings={cableSettings}
          setCableTestSettings={setCableTestSettings}
          cableTestStatus="idle"
        />
      </RoleProvider>
    );
    const { rerender } = render(section);
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
    rerender(section);
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it('ConfigBackupsSection loads backups and version once', async () => {
    mockFetch.mockReset();
    mockFetch.mockImplementation(() =>
      createMockResponse({ backups: [], current: 3, latest: 3, needsMigration: false }),
    );
    const section = (
      <RoleProvider isAuthenticated={false}>
        <ConfigBackupsSection />
      </RoleProvider>
    );
    const { rerender } = render(section);
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2));
    rerender(section);
    expect(mockFetch).toHaveBeenCalledTimes(2);
  });
});
