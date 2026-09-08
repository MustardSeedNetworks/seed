/**
 * SecurityPage.i18n.test.tsx — the security page renders real locale copy.
 *
 * S1-14b. The page owns no copy of its own: it is four cards in a grid, and
 * the copy a user reads belongs to them. So this suite renders the real cards
 * through the page, which is the surface an operator actually sees, rather
 * than each card in isolation.
 *
 * Writing it found the Guest Network Audit button rendering the raw
 * interpolation placeholder — `Run audit ({{count}} targets)` — in both
 * locales, because the call passed a default-value string where i18next
 * expects the options object carrying `count`.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../contexts/RoleContext';
import i18n from '../i18n';

const state = {
  targets: [{ label: 'EMR', address: '10.0.0.5' }] as { label: string; address: string }[],
};

vi.mock('../hooks/useGuestNetworkAudit', () => ({
  useGuestNetworkAudit: () => ({
    settings: { enabled: true, targets: state.targets },
    setSettings: vi.fn(),
    saveSettings: vi.fn(),
    runAudit: vi.fn(),
    report: null,
    loading: false,
    running: false,
    error: null,
  }),
}));

vi.mock('../hooks/useInsecurePortScan', () => ({
  useInsecurePortScan: () => ({
    result: null,
    scanning: false,
    error: null,
    scan: vi.fn(),
    reset: vi.fn(),
  }),
}));

vi.mock('../hooks/useBluetoothScan', () => ({
  useBluetoothScan: () => ({
    status: { state: 'idle', jobId: null, percentComplete: 0, error: null },
    running: false,
    result: null,
    devices: [],
    stats: null,
    startScan: vi.fn(),
    cancelScan: vi.fn(),
  }),
}));

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path), post: vi.fn() },
}));

const { SecurityPage } = await import('./SecurityPage');

async function renderIn(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(
    <RoleProvider isAuthenticated={true}>
      <SecurityPage />
    </RoleProvider>,
  );
  // The MFA status line only appears once the status request resolves.
  await waitFor(() => expect(screen.getAllByRole('button').length).toBeGreaterThan(2));
}

beforeEach(() => {
  mockGet.mockReset();
  state.targets = [{ label: 'EMR', address: '10.0.0.5' }];
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role: 'admin', isActive: true })
      : Promise.resolve({ totpEnabled: false, webauthnEnabled: false, webauthnCredentialCount: 0 }),
  );
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('SecurityPage — real locale copy', () => {
  it('renders the four card titles and their controls in English', async () => {
    await renderIn('en');

    expect(screen.getByText('Multi-factor authentication')).toBeVisible();
    expect(screen.getByText('Guest Network Audit')).toBeVisible();
    expect(screen.getByText('Insecure Port Scan')).toBeVisible();
    expect(screen.getByText('Bluetooth')).toBeVisible();

    expect(screen.getByText('No second factor enrolled')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Set up TOTP' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Add passkey' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Scan' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Scan for devices' })).toBeVisible();
  });

  it('counts the audit targets in the button instead of showing the placeholder', async () => {
    await renderIn('en');

    // One configured target: singular, and no `{{count}}` on screen.
    expect(screen.getByRole('button', { name: 'Run audit (1 target)' })).toBeVisible();
    expect(screen.queryByText(/\{\{count\}\}/)).toBeNull();
  });

  it('pluralises the audit target count', async () => {
    state.targets = [
      { label: 'EMR', address: '10.0.0.5' },
      { label: 'PACS', address: '10.0.0.6' },
    ];
    await renderIn('en');

    expect(screen.getByRole('button', { name: 'Run audit (2 targets)' })).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    for (const english of [
      'Multi-factor authentication',
      'No second factor enrolled',
      'Guest Network Audit',
      'Insecure Port Scan',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    for (const english of ['Set up TOTP', 'Add passkey', 'Scan', 'Scan for devices']) {
      expect(screen.queryByRole('button', { name: english })).toBeNull();
    }

    expect(screen.getByText('Autenticación de múltiples factores')).toBeVisible();
    expect(screen.getByText('Sin segundo factor registrado')).toBeVisible();
    expect(screen.getByText('Auditoría de red de invitados')).toBeVisible();
    expect(screen.getByText('Escaneo de puertos inseguros')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Configurar TOTP' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Ejecutar auditoría (1 objetivo)' })).toBeVisible();
    expect(screen.queryByText(/\{\{count\}\}/)).toBeNull();
  });

  it('leaves the protocol names alone under es', async () => {
    await renderIn('es');

    // Bluetooth and TOTP are the names of the things, not copy.
    expect(screen.getByText('Bluetooth')).toBeVisible();
    expect(screen.getByRole('button', { name: /TOTP/ })).toBeVisible();
  });
});
