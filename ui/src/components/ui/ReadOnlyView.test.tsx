/**
 * ReadOnlyView tests — cover the operator-passes-through path, the
 * viewer banner + fieldset path, and the native-disable propagation
 * that's the whole reason the wrapper exists.
 */

import { render } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';
import { ReadOnlyView } from './ReadOnlyView';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

const user = (role: CurrentUser['role']): CurrentUser => ({
  username: 'u',
  role,
  isActive: true,
});

describe('<ReadOnlyView>', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('renders children unwrapped for an operator', async () => {
    mockGet.mockResolvedValueOnce(user('operator'));
    const { container, findByTestId, queryByRole } = render(
      <RoleProvider>
        <ReadOnlyView>
          <input data-testid="probe" />
        </ReadOnlyView>
      </RoleProvider>,
    );
    const input = await findByTestId('probe');
    expect((input as HTMLInputElement).disabled).toBe(false);
    expect(queryByRole('status')).toBeNull();
    expect(container.querySelector('fieldset')).toBeNull();
  });

  it('wraps children in a disabled fieldset and shows a banner for a viewer', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    const { findByTestId, findByRole, container } = render(
      <RoleProvider>
        <ReadOnlyView>
          <input data-testid="probe-input" />
          <button data-testid="probe-button" type="button">
            Save
          </button>
        </ReadOnlyView>
      </RoleProvider>,
    );
    // Banner renders with status role for assistive tech.
    await findByRole('status');
    // The fieldset is the HTML-spec lever that disables every nested
    // form control without us having to gate each one. jsdom doesn't
    // synthesize the disabled-propagation onto the input.disabled
    // property, so the structural check (input nested under disabled
    // fieldset) is what the test asserts; the actual disabled paint
    // happens in real browsers per the HTML spec.
    const fieldset = container.querySelector('fieldset');
    expect(fieldset).not.toBeNull();
    expect(fieldset?.hasAttribute('disabled')).toBe(true);
    const input = await findByTestId('probe-input');
    const button = await findByTestId('probe-button');
    expect(fieldset?.contains(input)).toBe(true);
    expect(fieldset?.contains(button)).toBe(true);
  });

  it('honors a custom notice', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    const { findByText } = render(
      <RoleProvider>
        <ReadOnlyView notice="Custom message for this panel.">
          <span />
        </ReadOnlyView>
      </RoleProvider>,
    );
    await findByText('Custom message for this panel.');
  });
});
