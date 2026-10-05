/**
 * Render work skipped by the React Compiler rather than by useMemo and
 * useCallback (UI-SEED-41 slice 9, #3066).
 *
 * Compiled, ProfileManagement re-rendered with an unchanged context and
 * props returns the profile cards it cached, so their tooltips are not
 * rendered again. Fails when the compiler is removed from the vitest config.
 */

import { render } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';

import type { Profile } from '../../types/profile';
import { ProfileManagement } from './ProfileManagement';

const { tooltipRenders, context } = vi.hoisted(() => {
  const profiles: Profile[] = [
    {
      id: 'default',
      name: 'Default',
      description: 'Default profile',
      config: {},
      isDefault: true,
      createdAt: '2026-10-05T00:00:00Z',
      updatedAt: '2026-10-05T00:00:00Z',
    },
  ];
  return {
    tooltipRenders: vi.fn(),
    context: {
      profiles,
      activeProfile: profiles[0],
      isLoading: false,
      error: null,
      createProfile: vi.fn(),
      updateProfile: vi.fn(),
      deleteProfile: vi.fn(),
      switchProfile: vi.fn(),
      duplicateProfile: vi.fn(),
      downloadProfiles: vi.fn(),
    },
  };
});

vi.mock('../ui/Tooltip', () => ({
  Tooltip: ({ children }: { children: ReactNode }): ReactNode => {
    tooltipRenders();
    return children;
  },
}));
vi.mock('../../contexts/profileContext', () => ({ useProfileContext: () => context }));
vi.mock('../../contexts/RoleContext', () => ({ useRole: () => ({ canWrite: true }) }));

describe('compiled profile management skips work on an unchanged re-render', () => {
  it('ProfileManagement does not re-render its cards', () => {
    const onClose = vi.fn();
    const { rerender } = render(<ProfileManagement onClose={onClose} />);
    const first = tooltipRenders.mock.calls.length;
    rerender(<ProfileManagement onClose={onClose} />);
    expect(first).toBeGreaterThan(0);
    expect(tooltipRenders).toHaveBeenCalledTimes(first);
  });
});
