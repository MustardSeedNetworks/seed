/**
 * BonjourCard.i18n.test.tsx — the card renders real locale copy (S1-14's bar).
 *
 * The behaviour suite beside this one asserts on English strings, so it would
 * pass unchanged if every `es` value were English — the failure mode S1-14b
 * found on nine surfaces. This renders in both locales and requires the copy
 * to differ, which is the only assertion that can see that.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));
vi.mock('../../api', () => ({ api: { get: (p: string): Promise<unknown> => mockGet(p) } }));

const { BonjourCard } = await import('./BonjourCard');

const reflected = {
  interface: 'en0',
  localPrefixes: ['192.168.20.0/24'],
  serviceTypes: ['_ipp._tcp'],
  services: [
    {
      instance: 'Front Desk Printer',
      type: '_ipp._tcp',
      host: 'printer-vlan40.local',
      port: 631,
      addresses: ['10.44.40.61'],
      origin: 'off-segment',
      sourceAddresses: ['192.168.20.1'],
    },
  ],
  reflector: {
    state: 'reflected',
    remoteSubnets: ['10.44.40.0/24'],
    forwardedBy: ['192.168.20.1'],
  },
  responsesObserved: 4,
  truncated: false,
  durationMs: 4000,
};

async function renderIn(language: string): Promise<{ verdict: string; origin: string }> {
  await i18n.changeLanguage(language);
  mockGet.mockResolvedValue(reflected);
  const view = render(<BonjourCard />);
  await userEvent.click(screen.getByTestId('bonjour-browse'));
  const verdict = await screen.findByTestId('bonjour-reflector');
  const origin = (await screen.findByTestId('bonjour-services')).textContent ?? '';
  const text = verdict.textContent ?? '';
  view.unmount();
  return { verdict: text, origin };
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('BonjourCard copy', () => {
  it('renders the reflector verdict and the origin column in each locale', async () => {
    const en = await renderIn('en');
    const es = await renderIn('es');

    // The evidence is data and stays identical; only the prose around it moves.
    for (const rendered of [en.verdict, es.verdict]) {
      expect(rendered).toContain('10.44.40.0/24');
      expect(rendered).toContain('192.168.20.1');
    }
    expect(es.verdict).not.toBe(en.verdict);
    expect(es.origin).not.toBe(en.origin);

    // Pin the two that matter rather than only asserting difference: a
    // translated string that happens to differ by punctuation would pass that.
    expect(en.verdict).toContain('reflector');
    expect(es.verdict).toContain('subredes');
    expect(en.origin).toContain('Another subnet');
    expect(es.origin).toContain('Otra subred');
  });
});
