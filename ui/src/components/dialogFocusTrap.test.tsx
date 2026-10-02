/**
 * dialogFocusTrap.test.tsx — every `aria-modal` dialog keeps Tab inside itself.
 *
 * `aria-modal="true"` tells a screen reader the page behind is inert. Without a
 * trap that is a lie: Tab walks out of the dialog into controls the operator
 * cannot see (seed#2648). Most of these dialogs are hand-rolled rather than
 * built on <Modal>, so each one is checked here, including the two dialogs
 * ProfileManagement opens on top of itself.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

import { useFocusTrap } from '../hooks/useFocusTrap';
import { DiscoveryModal } from './cards/DiscoveryModal';
import { LogViewerModal } from './cards/LogViewerModal';
import { ProfileManagement } from './profiles/ProfileManagement';
import { Modal } from './ui/Modal';

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
    // Focused before the manager mounts, as the account button is in the app.
    const opener = document.body.appendChild(document.createElement('button'));
    opener.focus();
    render(<ProfileManagement onClose={onClose} />);
    const create = screen.getByTestId('profile-create');
    act(() => {
      create.focus();
    });
    fireEvent.click(create);

    const editor = await screen.findByRole('dialog', { name: 'Create profile' });
    expectTabStaysInside(editor);

    fireEvent.keyDown(document.activeElement ?? editor, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Create profile' })).toBeNull();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    // The manager's trap stayed open underneath, so it does not take focus
    // back to its own first control: the editor returns it to its opener.
    expect(document.activeElement).toBe(create);
    opener.remove();
  });

  // The trap listens on document. Modal used to stop Escape on its content so
  // it never got there, and the Bluetooth device table could not be closed
  // from the keyboard at all (#388).
  it('Modal closes on Escape pressed inside it', () => {
    const onClose = vi.fn();
    render(
      <Modal isOpen={true} onClose={onClose} title="Devices">
        <button type="button">inside</button>
      </Modal>,
    );

    fireEvent.keyDown(screen.getByRole('button', { name: 'inside' }), { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('traps focus in the delete confirmation', async () => {
    render(<ProfileManagement onClose={vi.fn()} />);
    fireEvent.click(screen.getByTestId('profile-delete-p1'));

    const confirm = await screen.findByRole('dialog', { name: 'Delete profile?' });
    expectTabStaysInside(confirm);
  });
});

// A parent re-render must not move focus inside an open dialog. Callers pass
// `onEscape` as an inline arrow, a new function on every render of the owner.
// The trap's effect listed it as a dependency, so each re-render tore the trap
// down, restoring focus to the control that opened the dialog, and set it up
// again, focusing the dialog's first control. The discovery modal's owner
// re-renders on every device poll, so a keyboard user walking the device table
// was thrown back to the Rescan button (#461).
function Dialog({ onEscape }: { onEscape: () => void }) {
  const ref = useFocusTrap<HTMLDivElement>({ isActive: true, onEscape });
  return (
    <div ref={ref} role="dialog">
      <button type="button">first</button>
      <button type="button">second</button>
    </div>
  );
}

function Owner({ onEscape }: { onEscape: (tick: number) => void }) {
  const [tick, setTick] = useState(0);
  return (
    <>
      <button type="button" onClick={() => setTick(tick + 1)}>
        opener {tick}
      </button>
      {/* Captures `tick`, so it is a new function on every render even under
          the React Compiler. */}
      <Dialog onEscape={() => onEscape(tick)} />
    </>
  );
}

function nextFrame(): Promise<void> {
  return act(() => new Promise<void>((resolve) => requestAnimationFrame(() => resolve())));
}

describe('a focus trap across re-renders of its owner', () => {
  it('keeps focus where it is when the owner re-renders with a new onEscape', async () => {
    const onEscape = vi.fn();
    render(<Owner onEscape={onEscape} />);
    await nextFrame(); // the trap's own autofocus on open
    const second = screen.getByRole('button', { name: 'second' });
    second.focus();

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: /opener/ }));
    });
    await nextFrame();

    expect(screen.getByRole('button', { name: 'opener 1' })).toBeInTheDocument();
    expect(document.activeElement).toBe(second);
  });

  // The trap used to move its initial focus a frame after opening. Under load
  // that frame came late -- after the operator had already focused a sort
  // header in the discovery modal -- and moved focus to the first control, so
  // the header's second Enter went to the CSV button instead (#2922).
  it('leaves focus the operator placed inside the dialog before its first frame', async () => {
    render(<Dialog onEscape={vi.fn()} />);
    const second = screen.getByRole('button', { name: 'second' });
    second.focus();

    await nextFrame();

    expect(document.activeElement).toBe(second);
  });

  it('focuses the first control when the dialog opens', () => {
    render(<Dialog onEscape={vi.fn()} />);
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'first' }));
  });

  it('calls the onEscape from the latest render', () => {
    const first = vi.fn();
    const latest = vi.fn();
    const { rerender } = render(<Dialog onEscape={first} />);
    rerender(<Dialog onEscape={latest} />);

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(latest).toHaveBeenCalledTimes(1);
    expect(first).not.toHaveBeenCalled();
  });
});
