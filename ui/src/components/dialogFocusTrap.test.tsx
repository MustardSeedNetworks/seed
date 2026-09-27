/**
 * dialogFocusTrap.test.tsx — every `aria-modal` dialog keeps Tab inside itself.
 *
 * `aria-modal="true"` tells a screen reader the page behind is inert. Without a
 * trap that is a lie: Tab walks out of the dialog into controls the operator
 * cannot see (seed#2648). These dialogs are hand-rolled rather than built on
 * <Modal>, so each one is checked here, including the two dialogs
 * ProfileManagement opens on top of itself.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

import { DiscoveryModal } from './cards/DiscoveryModal';
import { LogViewerModal } from './cards/LogViewerModal';
import { ProfileManagement } from './profiles/ProfileManagement';

vi.mock('../hooks/useLogs', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../hooks/useLogs')>()),
  useLogs: () => ({
    logs: [],
    allLogs: [],
    filters: { levels: [], layers: [], components: [], search: '' },
    setFilters: vi.fn(),
    resetFilters: vi.fn(),
    stats: null,
    isStreaming: false,
    setIsStreaming: vi.fn(),
    isLoading: false,
    error: null,
    clearLogs: vi.fn(),
  }),
}));

vi.mock('../contexts/RoleContext', () => ({
  useRole: () => ({ canWrite: true }),
}));

vi.mock('../contexts/profileContext', () => ({
  useProfileContext: () => ({
    profiles: [
      {
        id: 'p1',
        name: 'Clinic',
        description: '',
        isDefault: false,
        config: {},
        createdAt: '2026-09-01T00:00:00Z',
        updatedAt: '2026-09-01T00:00:00Z',
      },
    ],
    activeProfile: null,
    isLoading: false,
    error: null,
    createProfile: vi.fn(),
    updateProfile: vi.fn(),
    deleteProfile: vi.fn(),
    switchProfile: vi.fn(),
    duplicateProfile: vi.fn(),
    downloadProfiles: vi.fn(),
  }),
}));

// jsdom does no layout, so offsetParent is always null and the trap would see
// no focusable element at all. Any attached element counts as rendered here.
const offsetParent = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetParent');
beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
    configurable: true,
    get(this: HTMLElement) {
      return this.parentNode;
    },
  });
});
afterAll(() => {
  if (offsetParent) {
    Object.defineProperty(HTMLElement.prototype, 'offsetParent', offsetParent);
  }
});

function withQuery(node: ReactNode): ReactNode {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

function focusables(dialog: HTMLElement): HTMLElement[] {
  return Array.from(
    dialog.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => !el.hasAttribute('aria-hidden'));
}

function expectTabStaysInside(dialog: HTMLElement): void {
  const items = focusables(dialog);
  const first = items.at(0);
  const last = items.at(-1);
  if (!(first && last && first !== last)) {
    throw new Error('dialog needs at least two focusable controls');
  }

  last.focus();
  fireEvent.keyDown(last, { key: 'Tab' });
  expect(document.activeElement).toBe(first);

  fireEvent.keyDown(first, { key: 'Tab', shiftKey: true });
  expect(document.activeElement).toBe(last);
}

describe('dialog focus traps', () => {
  it('LogViewerModal keeps Tab inside and closes on Escape', () => {
    const onClose = vi.fn();
    render(<LogViewerModal isOpen={true} onClose={onClose} />);
    const dialog = screen.getByRole('dialog');

    expectTabStaysInside(dialog);
    fireEvent.keyDown(document.activeElement ?? dialog, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('DiscoveryModal keeps Tab inside and closes on Escape', () => {
    const onClose = vi.fn();
    render(withQuery(<DiscoveryModal isOpen={true} onClose={onClose} data={null} />));
    const dialog = screen.getByRole('dialog');

    expectTabStaysInside(dialog);
    fireEvent.keyDown(document.activeElement ?? dialog, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('ProfileManagement keeps Tab inside and focuses its close button on open', async () => {
    const onClose = vi.fn();
    render(<ProfileManagement onClose={onClose} />);
    const dialog = screen.getByRole('dialog');

    await waitFor(() => {
      expect(document.activeElement).toBe(screen.getByTestId('profile-modal-close'));
    });
    expectTabStaysInside(dialog);
    // From the close button: a focused control with a tooltip takes the first
    // Escape to dismiss its bubble, which is the intended order.
    const close = screen.getByTestId('profile-modal-close');
    act(() => {
      close.focus();
    });
    fireEvent.keyDown(close, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('hands focus to the profile editor, and Escape closes only the editor', async () => {
    const onClose = vi.fn();
    render(<ProfileManagement onClose={onClose} />);
    fireEvent.click(screen.getByTestId('profile-create'));

    const editor = await screen.findByRole('dialog', { name: 'Create profile' });
    expectTabStaysInside(editor);

    fireEvent.keyDown(document.activeElement ?? editor, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Create profile' })).toBeNull();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('traps focus in the delete confirmation', async () => {
    render(<ProfileManagement onClose={vi.fn()} />);
    fireEvent.click(screen.getByTestId('profile-delete-p1'));

    const confirm = await screen.findByRole('dialog', { name: 'Delete profile?' });
    expectTabStaysInside(confirm);
  });
});
