/**
 * Link is the one Card grid page that opens with a rollup, so the rule under
 * test is the one the rollup exists for: nothing that has not been measured
 * may read as healthy. A page that says "all clear" while the interface has
 * reported nothing is the failure mode, not a cosmetic issue.
 *
 * The second rule is the archetype's: a card the interface cannot produce
 * explains itself, and one that is simply not needed does not.
 */
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LinkData } from '../components/cards/LinkCard';
import { LinkPage } from './LinkPage';

const useAppContext = vi.fn();

vi.mock('../contexts/AppContext', () => ({
  useAppContext: () => useAppContext(),
}));

vi.mock('../components/cards/LinkCard', () => ({
  LinkCard: () => <div data-testid="link-card" />,
}));
vi.mock('../components/cards/CableCard', () => ({
  CableCard: () => <div data-testid="cable-card" />,
}));
vi.mock('../components/cards/WiFiCard', () => ({
  WiFiCard: () => <div data-testid="wifi-card" />,
}));

function link(over: Partial<LinkData> = {}): LinkData {
  return {
    linkUp: true,
    carrier: true,
    hasIp: true,
    speed: '1000Mb/s',
    duplex: 'full',
    advertisedSpeeds: [],
    mtu: 1500,
    flapCount24h: 0,
    ...over,
  };
}

function context(over: Record<string, unknown> = {}) {
  return {
    cards: { link: link(), cable: null, wifi: null },
    loading: false,
    isWifi: false,
    displayOptions: { unitSystem: 'metric' },
    ...over,
  };
}

function rollup(): HTMLElement {
  const el = document.querySelector<HTMLElement>('section[aria-live="polite"][data-state]');
  if (el === null) {
    throw new Error('status rollup not rendered');
  }
  return el;
}

beforeEach(() => {
  useAppContext.mockReset();
});

describe('LinkPage', () => {
  it('reports a healthy link calmly, with its figures', () => {
    useAppContext.mockReturnValue(context());
    render(<LinkPage />);

    expect(rollup()).toHaveAttribute('data-state', 'ok');
    expect(rollup()).toHaveTextContent('1000Mb/s');
  });

  it('leads with no carrier, which is the sentence someone needs first', () => {
    useAppContext.mockReturnValue(
      context({
        cards: { link: link({ carrier: false, linkUp: false }), cable: null, wifi: null },
      }),
    );
    render(<LinkPage />);

    expect(rollup()).toHaveAttribute('data-state', 'crit');
    expect(rollup()).toHaveTextContent('No carrier');
  });

  it('calls a link with no address degraded, not healthy', () => {
    useAppContext.mockReturnValue(
      context({ cards: { link: link({ hasIp: false }), cable: null, wifi: null } }),
    );
    render(<LinkPage />);

    expect(rollup()).toHaveAttribute('data-state', 'warn');
  });

  it('refuses to read as healthy while nothing has arrived', () => {
    useAppContext.mockReturnValue(context({ cards: { link: null, cable: null, wifi: null } }));
    render(<LinkPage />);

    expect(rollup()).toHaveAttribute('data-state', 'unknown');
    expect(rollup()).toHaveTextContent('not arriving');
  });

  it('never reads as ok while still loading', () => {
    useAppContext.mockReturnValue(
      context({ loading: true, cards: { link: null, cable: null, wifi: null } }),
    );
    render(<LinkPage />);

    expect(rollup()).not.toHaveAttribute('data-state', 'ok');
  });

  it('shows the cable card only when the link is down, and says nothing when it is up', () => {
    useAppContext.mockReturnValue(context());
    const { unmount } = render(<LinkPage />);
    expect(screen.queryByTestId('cable-card')).not.toBeInTheDocument();
    expect(document.querySelector('[data-testid^="card-absent-"]')).toBeNull();
    unmount();

    useAppContext.mockReturnValue(
      context({ cards: { link: link({ linkUp: false }), cable: null, wifi: null } }),
    );
    render(<LinkPage />);
    expect(screen.getByTestId('cable-card')).toBeInTheDocument();
  });

  it('says why the wired cards are missing on a wireless interface', () => {
    useAppContext.mockReturnValue(
      context({ isWifi: true, cards: { link: null, cable: null, wifi: null } }),
    );
    render(<LinkPage />);

    expect(screen.queryByTestId('link-card')).not.toBeInTheDocument();
    expect(screen.getByTestId('card-absent-wired-link')).toHaveTextContent('wireless');
    expect(screen.getByTestId('wifi-card')).toBeInTheDocument();
  });
});
