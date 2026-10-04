/**
 * DnsCard: whose resolvers the card is showing (#2690, slice 2).
 *
 * The card used to print the host's resolvers whatever interface was selected,
 * beside a gateway taken from the system default route. The list alone cannot
 * say which of three things happened, so these cover all three: the interface
 * has resolvers, it has none, or this host cannot attribute resolvers to an
 * interface at all.
 */

import { fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { DnsCard, type DnsData } from './DnsCard';

function dnsData(over: Partial<DnsData> = {}): DnsData {
  return {
    server: '10.0.0.1',
    servers: ['10.0.0.1'],
    testHostname: 'example.com',
    forward: {
      outcome: 'resolved',
      time: 12,
      timeMs: 12,
      status: 'success',
      resolved: ['93.184.216.34'],
    },
    reverse: null,
    ...over,
  };
}

describe('DnsCard resolver scope', () => {
  it('names the resolvers as the interface’s when they are scoped to it', () => {
    render(<DnsCard data={dnsData({ serverScope: 'interface' })} />);
    expect(screen.getByText('Resolvers for this interface')).toBeInTheDocument();
    expect(screen.getByText('10.0.0.1')).toBeInTheDocument();
  });

  it('says a system-wide list is system-wide instead of implying it is the link’s', () => {
    render(<DnsCard data={dnsData({ serverScope: 'system' })} />);
    expect(screen.getByText('DNS Servers')).toBeInTheDocument();
    expect(
      screen.getByText('Shown for the whole host, not for the selected interface.'),
    ).toBeInTheDocument();
  });

  /* An interface with no resolvers of its own shows no measurement, because
     one taken through another interface would describe that link. The card
     has to say that, or the reader sees an empty card and assumes a bug. */
  it('explains an interface with no resolvers instead of showing an empty list', () => {
    render(<DnsCard data={dnsData({ serverScope: 'interface', servers: [], forward: null })} />);
    expect(screen.getByText('No resolvers on this interface')).toBeInTheDocument();
    expect(screen.queryByText('Forward (A)')).not.toBeInTheDocument();
  });

  /* A payload from before the scoping carries no scope at all; it must read as
     the system-wide answer it is, not as an absence. */
  it('treats a payload without a scope as system-wide', () => {
    render(<DnsCard data={dnsData({ servers: [] })} />);
    expect(screen.queryByText('No resolvers on this interface')).not.toBeInTheDocument();
    expect(screen.getByText('DNS Servers')).toBeInTheDocument();
  });
});

/* A per-server row reads "N/A" when the lookup answered nothing. The answer is
   in `resolved`: empty means nothing came back. */
describe('DnsCard per-server rows', () => {
  const lookup = (resolved?: string[]): NonNullable<DnsData['forward']> => ({
    outcome: resolved ? 'resolved' : 'noRecord',
    time: 7,
    timeMs: 7,
    status: resolved ? 'success' : 'warning',
    ...(resolved ? { resolved } : {}),
  });

  it('marks an empty answer N/A', () => {
    render(
      <DnsCard
        data={dnsData({
          perServerResults: [
            {
              server: '192.0.2.53',
              forward: lookup(),
              forwardIpv6: lookup(['2001:db8::1']),
              status: 'warning',
              avgTimeMs: 7,
            },
          ],
        })}
      />,
    );
    fireEvent.click(screen.getByText('Server Tests'));

    expect(screen.getAllByText('N/A')).toHaveLength(1);
  });
});

/* The daemon sends a code for what each lookup came back with; the card words
   it in the reader's language (#2844). It used to print the daemon's English
   ("No AAAA record", "Failed") whatever the locale. */
describe('DnsCard lookup outcomes', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  const outcomes: DnsData = {
    server: '',
    servers: [],
    testHostname: 'example.com',
    forward: {
      outcome: 'resolved',
      time: 9,
      timeMs: 9,
      status: 'success',
      resolved: ['192.0.2.10'],
    },
    forwardIpv6: { outcome: 'noRecord', time: 9, timeMs: 9, status: 'warning' },
    reverse: { outcome: 'failed', time: 9, timeMs: 9, status: 'error', error: 'timeout' },
    reverseIpv6: null,
  };

  it.each([
    ['en', 'No AAAA record', 'Lookup failed', 'System resolver'],
    ['es', 'Sin registro AAAA', 'La consulta falló', 'Resolutor del sistema'],
  ])('words every outcome in %s', async (lng, noRecord, failed, systemResolver) => {
    await i18n.changeLanguage(lng);
    const { container } = render(<DnsCard data={outcomes} />);
    const card = within(container);

    expect(card.getByText('192.0.2.10')).toBeInTheDocument();
    expect(card.getByText(noRecord)).toBeInTheDocument();
    expect(card.getByText(failed)).toBeInTheDocument();
    expect(card.getByText(systemResolver)).toBeInTheDocument();
    for (const code of ['resolved', 'noRecord', 'failed', 'timeout']) {
      expect(card.queryByText(code)).not.toBeInTheDocument();
    }
  });
});
